package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// real test case
func TestAccDomain_ApiClient_basic(t *testing.T) {
	//var ctx context.Context
	ctx := context.Background()

	apiKey := os.Getenv("SPACESHIP_API_KEY")
	apiSecret := os.Getenv("SPACESHIP_API_SECRET")
	domain := os.Getenv("SPACESHIP_TEST_DOMAIN")

	client, _ := NewClient(defaultBaseURL, apiKey, apiSecret)

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

func composeRateLimitApiResponse(retryAfterSeconds string) *http.Response {
	resp := &http.Response{
		StatusCode: 429,
		Status:     "429 Too Many Requests",
		Proto:      "HTTP/2.0",
		ProtoMajor: 2,
		ProtoMinor: 0,
		Header: http.Header{
			"Content-Type":           []string{"application/problem+json"},
			"Retry-After":            []string{retryAfterSeconds}, // seconds to wait
			"Spaceship-Operation-Id": []string{"71814014307c5e96"},
			"Spaceship-Error-Code":   []string{"application.rateLimit"},
		},
	}
	return resp
}

// func TestWaiting(t *testing.T) {
// 	resp := composeRateLimitApiResponse("2")

// 	t.Run()
// }

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
		w.Write([]byte(`{"isEnabled": true}`))
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
	// why atomic and not just 1?
	// probably due to concurency
	var callCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := callCount.Add(1)

		if current == 1 {
			w.Header().Set("Retry-After", "60") // 60 seconds
			w.WriteHeader(http.StatusTooManyRequests)
			w.Header().Set("Content-Type", "application/problem+json")

			// 	strToEncode := `"detail":"Request rate limit
			// exceeded.","data":{"rateLimitRule":"The limit for updating the autorenewal
			// state for a domain is 5 requests per domain, within 300
			// seconds.","limit":5,"windowInSeconds":300}`

			// 	data, _ := json.Marshal(strToEncode)
			//w.Write([]byte(data))
			return
		}

		if current == 2 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"isEnabled": true}`))
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
	// why this default is so imortant
	// does not work without it at all
	default:
	}

	fakeClock.Advance(60 * time.Second)

	select {
	// why res here?
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
