package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestAccDomain_ApiClient_basic(t *testing.T) {
	ctx := context.Background()

	apiKey := os.Getenv("SPACESHIP_API_KEY")
	apiSecret := os.Getenv("SPACESHIP_API_SECRET")
	domain := os.Getenv("SPACESHIP_TEST_DOMAIN")

	client, _ := NewClient(DefaultBaseURL, apiKey, apiSecret)

	autoRenewalTestCases := []bool{true, false, true, false, true, false, true}

	t.Run(
		"ratelimit", func(t *testing.T) {
			for _, state := range autoRenewalTestCases {
				t.Log("running update with value ", state)
				resp, error := client.UpdateAutoRenew(ctx, domain, state)

				t.Log(resp.IsEnabled)

				if error != nil {
					t.FailNow()
				}
			}
		})

}

// TestGetDomainInfo_Success verifies the happy path: the single-domain endpoint
// returns 200 and the response is decoded correctly.
func TestGetDomainInfo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"example.com","unicodeName":"example.com","autoRenew":true}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	info, err := c.GetDomainInfo(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Name != "example.com" {
		t.Fatalf("expected name %q, got %q", "example.com", info.Name)
	}
	if !info.AutoRenew {
		t.Fatal("expected autoRenew to be true")
	}
}

// TestGetDomainInfo_RateLimitFallback_Found verifies that when the single-domain
// endpoint returns 429, GetDomainInfo falls back to GetDomainList and finds the
// domain in the list.
func TestGetDomainInfo_RateLimitFallback_Found(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/domains/example.com" {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		// List endpoint returns the domain.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[{"name":"example.com","autoRenew":true}],"total":1}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	info, err := c.GetDomainInfo(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Name != "example.com" {
		t.Fatalf("expected name %q, got %q", "example.com", info.Name)
	}
}

// TestGetDomainInfo_RateLimitFallback_NotFound verifies that when the single-domain
// endpoint returns 429 and the domain is not present in the list fallback, a
// not-found error is returned instead of the original 429 error.
func TestGetDomainInfo_RateLimitFallback_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/domains/missing.com" {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		// List endpoint returns domains, but not the one we're looking for.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[{"name":"other.com"}],"total":1}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetDomainInfo(context.Background(), "missing.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Fatalf("expected not-found error, got: %v", err)
	}
}

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	c, err := NewClient(baseURL, "test-key", "test-secret")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return c
}

// TestDoJSON_ConcurrentRateLimit verifies that when goroutine A hits a 429 and
// activates the shared rate limiter, goroutine B (started afterwards) sees the
// active wait via peek() and holds off without making its own request. Both
// goroutines resume together after the shared timer fires.
func TestDoJSON_ConcurrentRateLimit(t *testing.T) {
	var callCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := callCount.Add(1)
		if current == 1 {
			// First call: goroutine A gets rate-limited.
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		// All subsequent calls succeed.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"isEnabled": true}`))
	}))
	defer server.Close()

	fakeClock := NewFakeClock(time.Now())
	client := newTestClientWithClock(t, server.URL, fakeClock)

	// Goroutine A: will hit 429 and activate the shared rate limiter.
	doneA := make(chan result, 1)
	go func() {
		var resp AutoRenewalResponse
		statusCode, err := client.doJSONWithRetry(context.Background(), http.MethodPut, server.URL, nil, &resp)
		doneA <- result{statusCode: statusCode, err: err}
	}()

	// Wait until goroutine A has registered its sleep with the fake clock,
	// meaning the rate limiter is now active.
	fakeClock.WaitForWaiters(1)

	// Goroutine B: started after the rate limiter is active.
	// It should block on peek() without making an API request.
	doneB := make(chan result, 1)
	go func() {
		var resp AutoRenewalResponse
		statusCode, err := client.doJSONWithRetry(context.Background(), http.MethodPut, server.URL, nil, &resp)
		doneB <- result{statusCode: statusCode, err: err}
	}()

	// Give goroutine B a moment to reach the peek() gate and block.
	time.Sleep(10 * time.Millisecond)

	// Neither goroutine should have finished yet.
	if len(doneA) > 0 || len(doneB) > 0 {
		t.Fatal("goroutines should still be blocked on the rate limiter")
	}

	// Advance the fake clock past the Retry-After window.
	fakeClock.Advance(60 * time.Second)

	// Collect results from both goroutines.
	for i, ch := range []chan result{doneA, doneB} {
		select {
		case res := <-ch:
			if res.err != nil {
				t.Fatalf("goroutine %d: unexpected error: %v", i, res.err)
			}
			if res.statusCode != http.StatusOK {
				t.Fatalf("goroutine %d: expected 200, got %d", i, res.statusCode)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("goroutine %d: timed out waiting for completion", i)
		}
	}

	// A: 1 failed attempt + 1 successful retry = 2 calls
	// B: held at gate, then 1 successful call = 1 call
	// Total: 3
	if got := callCount.Load(); got != 3 {
		t.Fatalf("expected 3 API calls, got %d", got)
	}
}

func TestDoJSON_SingleRetry(t *testing.T) {
	var callCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := callCount.Add(1)

		if current == 1 {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		if current == 2 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"isEnabled": true}`))
			return
		}

	}))
	defer server.Close()

	fakeClock := NewFakeClock(time.Now())
	client := newTestClientWithClock(t, server.URL, fakeClock)

	done := make(chan result, 1)
	go func() {
		var resp AutoRenewalResponse
		statusCode, err := client.doJSONWithRetry(context.Background(), http.MethodPut, server.URL, nil, &resp)
		done <- result{
			statusCode: statusCode,
			err:        err,
		}
	}()

	// Wait until the goroutine is actually sleeping on the fake clock.
	// 1 means "wait for 1 waiter", NOT "wait 60 of anything".
	fakeClock.WaitForWaiters(1)

	select {
	case <-done:
		t.Fatal("should still be waiting")
	default:
	}

	fakeClock.Advance(60 * time.Second)

	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("unexpected error: %v", res.err)
		}
		if res.statusCode != http.StatusOK {
			t.Fatalf("Expected status code of 200 but got %d", res.statusCode)
		}

	case <-time.After(100 * time.Second):
		t.Fatal("timed out waiting for completion")
	}

	expectedHitCount := 2
	if callCount.Load() != int32(expectedHitCount) {
		t.Fatalf("expected %d calls, but got %d", expectedHitCount, callCount.Load())
	}
}

// TestDoJSON_MaxRetryWaitBudget verifies that when no context deadline is set,
// the client-level maxRetryWait budget kicks in and stops the retry loop.
func TestDoJSON_MaxRetryWaitBudget(t *testing.T) {
	var callCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "test-key", "test-secret",
		WithMaxRetryWait(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// context.Background() has no deadline — the client-level budget is the only safety net.
	var resp AutoRenewalResponse
	_, err = c.doJSONWithRetry(context.Background(), http.MethodPut, server.URL, nil, &resp)

	if err == nil {
		t.Fatal("expected error from maxRetryWait budget, got nil")
	}

	var rlErr *RateLimitTimeoutError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected RateLimitTimeoutError, got: %v", err)
	}

	if callCount.Load() < 1 {
		t.Fatal("expected at least 1 API call")
	}
}

func newTestClientWithClock(t *testing.T, baseURL string, clock Clock) *Client {
	t.Helper()
	client, err := NewClient(baseURL, "test-key", "test-secret", WithClock(clock))

	if err != nil {
		t.Fatalf("failed to create client %v", err)
	}
	return client
}

type result struct {
	statusCode int
	err        error
}
