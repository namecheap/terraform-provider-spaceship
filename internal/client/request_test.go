package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func testServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-key", "test-secret")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	c.httpClient = server.Client()
	return c, server
}

func TestDoJSON_GetRequest(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("expected X-API-Key header")
		}
		if r.Header.Get("X-API-Secret") != "test-secret" {
			t.Errorf("expected X-API-Secret header")
		}
		if r.Header.Get("Content-Type") != "" {
			t.Errorf("GET should not set Content-Type")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"name": "test"})
	})

	var out map[string]string
	status, err := c.doJSON(context.Background(), http.MethodGet, c.endpointURL(nil, nil), nil, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 {
		t.Errorf("expected status 200, got %d", status)
	}
	if out["name"] != "test" {
		t.Errorf("expected name=test, got %q", out["name"])
	}
}

func TestDoJSON_PostWithPayload(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json")
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "value" {
			t.Errorf("expected key=value in body, got %v", body)
		}
		w.WriteHeader(http.StatusCreated)
	})

	status, err := c.doJSON(context.Background(), http.MethodPost, c.endpointURL(nil, nil), map[string]string{"key": "value"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 201 {
		t.Errorf("expected status 201, got %d", status)
	}
}

func TestDoJSON_ErrorResponse(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid request"))
	})

	_, err := c.doJSON(context.Background(), http.MethodGet, c.endpointURL(nil, nil), nil, nil)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !IsNotFoundError(err) && err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestDoJSON_NilOutput(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	status, err := c.doJSON(context.Background(), http.MethodDelete, c.endpointURL(nil, nil), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 204 {
		t.Errorf("expected status 204, got %d", status)
	}
}

func TestDoJSON_RetriesOn429(t *testing.T) {
	var attempts atomic.Int32
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("rate limited"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	var out map[string]string
	status, err := c.doJSON(context.Background(), http.MethodGet, c.endpointURL(nil, nil), nil, &out)
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	if status != 200 {
		t.Errorf("expected status 200, got %d", status)
	}
	if out["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", out["status"])
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts (2 retries + 1 success), got %d", attempts.Load())
	}
}

func TestDoJSON_429ExhaustsRetries(t *testing.T) {
	var attempts atomic.Int32
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limited"))
	})

	_, err := c.doJSON(context.Background(), http.MethodGet, c.endpointURL(nil, nil), nil, nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if attempts.Load() != int32(maxRetries) {
		t.Errorf("expected %d attempts, got %d", maxRetries, attempts.Load())
	}
}

func TestDoJSON_429RespectsContextCancellation(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limited"))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := c.doJSON(ctx, http.MethodGet, c.endpointURL(nil, nil), nil, nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if elapsed > 5*time.Second {
		t.Errorf("expected quick cancellation, took %s", elapsed)
	}
}

func TestDoJSON_429WithPostPayloadRetries(t *testing.T) {
	var attempts atomic.Int32
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "value" {
			t.Errorf("attempt %d: expected key=value in body, got %v", n, body)
		}

		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("rate limited"))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	status, err := c.doJSON(context.Background(), http.MethodPost, c.endpointURL(nil, nil), map[string]string{"key": "value"}, nil)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	if status != 204 {
		t.Errorf("expected status 204, got %d", status)
	}
	if attempts.Load() != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts.Load())
	}
}

func TestDoJSON_Non429ErrorNotRetried(t *testing.T) {
	var attempts atomic.Int32
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	})

	_, err := c.doJSON(context.Background(), http.MethodGet, c.endpointURL(nil, nil), nil, nil)
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if attempts.Load() != 1 {
		t.Errorf("expected 1 attempt (no retry for 500), got %d", attempts.Load())
	}
}

func TestRetryDelay(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected time.Duration
	}{
		{"valid header", "3", 3 * time.Second},
		{"missing header", "", defaultRetryDelay},
		{"non-numeric header", "abc", defaultRetryDelay},
		{"zero header", "0", defaultRetryDelay},
		{"negative header", "-1", defaultRetryDelay},
		{"exceeds max", "120", maxRetryDelay},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{Header: http.Header{}}
			if tc.header != "" {
				resp.Header.Set("Retry-After", tc.header)
			}
			got := retryDelay(resp)
			if got != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, got)
			}
		})
	}
}
