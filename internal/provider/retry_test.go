package provider

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/namecheap/go-spaceship-sdk/client"
)

// fakeSleep swaps retrySleep for an instant recorder and retryNow for a fake
// clock the recorder advances, and resets the shared limiter so buckets from
// other tests cannot leak in. Tests using it must not run in parallel
// (package-level overrides). The mutex guards the fake clock so a test
// driving concurrent goroutines through withRetry stays race-free.
func fakeSleep(t *testing.T) *[]time.Duration {
	t.Helper()
	var mu sync.Mutex
	var recorded []time.Duration
	now := time.Unix(0, 0)
	origSleep, origNow, origLimiter := retrySleep, retryNow, retryLimiter
	retryNow = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}
	retrySleep = func(_ context.Context, d time.Duration) error {
		mu.Lock()
		defer mu.Unlock()
		recorded = append(recorded, d)
		now = now.Add(d)
		return nil
	}
	retryLimiter = newRateLimiter()
	t.Cleanup(func() { retrySleep, retryNow, retryLimiter = origSleep, origNow, origLimiter })
	return &recorded
}

func rateLimitErr(retryAfter time.Duration) error {
	return &client.SpaceshipApiError{Status: http.StatusTooManyRequests, RetryAfter: retryAfter}
}

// Succeeding immediately calls fn once and never sleeps.
func TestWithRetry_SuccessFirstTry(t *testing.T) {
	waits := fakeSleep(t)
	calls := 0
	err := withRetry(context.Background(), "op", "example.com", func() error {
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
	err := withRetry(context.Background(), "op", "example.com", func() error {
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
	err := withRetry(context.Background(), "op", "example.com", func() error {
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
	err := withRetry(context.Background(), "op", "example.com", func() error {
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
	err := withRetry(ctx, "read domain info", "example.com", func() error {
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

// A wait that nominally fits the deadline but leaves no headroom for the
// retried call itself fails fast with the actionable message instead of
// sleeping into a guaranteed "context deadline exceeded".
func TestWithRetry_FailsFastWhenWaitLeavesNoHeadroom(t *testing.T) {
	waits := fakeSleep(t)
	ctx, cancel := context.WithTimeout(context.Background(), 32*time.Second)
	defer cancel()
	calls := 0
	err := withRetry(ctx, "read domain info", "example.com", func() error {
		calls++
		return rateLimitErr(30 * time.Second) // wait 31s fits 32s, headroom does not
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected actionable timeout message, got %q", err.Error())
	}
	if calls != 1 || len(*waits) != 0 {
		t.Errorf("expected fail-fast with no sleep, got %d calls, %d sleeps", calls, len(*waits))
	}
}

// A bucket re-blocked while a waiter sleeps (a sibling goroutine's 429) is
// waited out again before the next attempt instead of burning a request the
// limiter already knows is doomed.
func TestWithRetry_WaitsAgainWhenBucketReblockedDuringSleep(t *testing.T) {
	waits := fakeSleep(t)
	const key = "read domain info|example.com"
	retryLimiter.block(key, 10*time.Second)

	inner := retrySleep
	reblocked := false
	retrySleep = func(ctx context.Context, d time.Duration) error {
		err := inner(ctx, d)
		if !reblocked {
			reblocked = true
			retryLimiter.block(key, 20*time.Second)
		}
		return err
	}

	calls := 0
	err := withRetry(context.Background(), "read domain info", "example.com", func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call after both waits, got %d", calls)
	}
	if len(*waits) != 2 || (*waits)[0] != 10*time.Second || (*waits)[1] != 20*time.Second {
		t.Errorf("expected sleeps of 10s then 20s, got %v", *waits)
	}
}

// Cancelling ctx during the wait aborts promptly with ctx.Err() (Ctrl-C path).
// Uses the real sleepContext and clock.
func TestWithRetry_CancelledDuringSleep(t *testing.T) {
	origLimiter := retryLimiter
	retryLimiter = newRateLimiter()
	t.Cleanup(func() { retryLimiter = origLimiter })

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	err := withRetry(ctx, "op", "example.com", func() error {
		return rateLimitErr(30 * time.Second)
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("cancellation took %s, expected prompt abort", elapsed)
	}
}

// A caller entering a bucket another goroutine already blocked waits out the
// remaining window before its first attempt instead of burning a request.
func TestWithRetry_JoinsWaitStartedByAnotherCaller(t *testing.T) {
	waits := fakeSleep(t)
	retryLimiter.block("read domain info|example.com", 50*time.Second)

	calls := 0
	err := withRetry(context.Background(), "read domain info", "example.com", func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
	if len(*waits) != 1 || (*waits)[0] != 50*time.Second {
		t.Errorf("expected one joined sleep of 50s, got %v", *waits)
	}
}

// Buckets are keyed per operation and domain: one throttled domain must not
// pause work on another.
func TestWithRetry_IndependentBucketsDoNotWait(t *testing.T) {
	waits := fakeSleep(t)
	retryLimiter.block("read domain info|throttled.com", 300*time.Second)

	calls := 0
	err := withRetry(context.Background(), "read domain info", "other.com", func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 || len(*waits) != 0 {
		t.Errorf("expected no wait for an unrelated bucket, got %d calls, %v", calls, *waits)
	}
}
