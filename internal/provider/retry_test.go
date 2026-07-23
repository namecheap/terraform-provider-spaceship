package provider

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/namecheap/go-spaceship-sdk/client"
)

// fakeSleep swaps retrySleep for an instant recorder. Tests using it must not
// run in parallel (package-level override).
func fakeSleep(t *testing.T) *[]time.Duration {
	t.Helper()
	var recorded []time.Duration
	orig := retrySleep
	retrySleep = func(_ context.Context, d time.Duration) error {
		recorded = append(recorded, d)
		return nil
	}
	t.Cleanup(func() { retrySleep = orig })
	return &recorded
}

func rateLimitErr(retryAfter time.Duration) error {
	return &client.SpaceshipApiError{Status: http.StatusTooManyRequests, RetryAfter: retryAfter}
}

// Succeeding immediately calls fn once and never sleeps.
func TestWithRetry_SuccessFirstTry(t *testing.T) {
	waits := fakeSleep(t)
	calls := 0
	err := withRetry(context.Background(), "op", func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 || len(*waits) != 0 {
		t.Errorf("expected 1 call and 0 sleeps, got %d calls, %d sleeps", calls, len(*waits))
	}
}

// A 429 with Retry-After sleeps that duration plus the margin, then retries.
func TestWithRetry_RetriesRateLimitThenSucceeds(t *testing.T) {
	waits := fakeSleep(t)
	calls := 0
	err := withRetry(context.Background(), "op", func() error {
		calls++
		if calls == 1 {
			return rateLimitErr(120 * time.Second)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
	if len(*waits) != 1 || (*waits)[0] != 121*time.Second {
		t.Errorf("expected one sleep of 121s, got %v", *waits)
	}
}

// Non-429 errors return unchanged with no retry.
func TestWithRetry_NonRateLimitErrorPassesThrough(t *testing.T) {
	waits := fakeSleep(t)
	sentinel := errors.New("boom")
	calls := 0
	err := withRetry(context.Background(), "op", func() error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if calls != 1 || len(*waits) != 0 {
		t.Errorf("expected no retry, got %d calls, %d sleeps", calls, len(*waits))
	}
}

// A 429 without Retry-After uses the default wait plus margin.
func TestWithRetry_MissingRetryAfterUsesDefault(t *testing.T) {
	waits := fakeSleep(t)
	calls := 0
	err := withRetry(context.Background(), "op", func() error {
		calls++
		if calls == 1 {
			return rateLimitErr(0)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*waits) != 1 || (*waits)[0] != 31*time.Second {
		t.Errorf("expected one sleep of 31s, got %v", *waits)
	}
}

// A wait that cannot fit before the ctx deadline fails immediately, without
// sleeping, wrapping the original API error.
func TestWithRetry_FailsFastWhenWaitExceedsDeadline(t *testing.T) {
	waits := fakeSleep(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	calls := 0
	err := withRetry(ctx, "read domain info", func() error {
		calls++
		return rateLimitErr(300 * time.Second)
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.SpaceshipApiError
	if !errors.As(err, &apiErr) {
		t.Errorf("expected error to wrap the API error, got %v", err)
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected actionable timeout message, got %q", err.Error())
	}
	if calls != 1 || len(*waits) != 0 {
		t.Errorf("expected fail-fast with no sleep, got %d calls, %d sleeps", calls, len(*waits))
	}
}

// Cancelling ctx during the wait aborts promptly with ctx.Err() (Ctrl-C path).
// Uses the real sleepContext.
func TestWithRetry_CancelledDuringSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	err := withRetry(ctx, "op", func() error {
		return rateLimitErr(30 * time.Second)
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("cancellation took %s, expected prompt abort", elapsed)
	}
}
