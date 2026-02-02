package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// doJSON makes a single HTTP request with no retry logic.
// Used for one-shot calls (e.g. the GetDomainInfo fallback path).
func (c *Client) doJSON(ctx context.Context, method, endpoint string, payload any, out any) (int, error) {
	data, err := marshalPayload(payload)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, readerOf(data))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	c.applyAuth(req)
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return resp.StatusCode, c.errorFromResponse(resp)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}
	}

	return resp.StatusCode, nil
}

// doJSONWithRetry makes an HTTP request, retrying transparently after 429
// rate-limit responses. All concurrent callers share a single wait period via
// the client's rateLimiter — when any goroutine hits a 429, the others pause
// too and all resume together after the Retry-After window.
//
// The supplied context controls both individual HTTP requests and wait periods:
// cancelling it (e.g. Ctrl+C or a Terraform timeout) unblocks the goroutine
// immediately and returns ctx.Err().
func (c *Client) doJSONWithRetry(ctx context.Context, method, endpoint string, payload any, out any) (int, error) {
	data, err := marshalPayload(payload)
	if err != nil {
		return 0, err
	}

	for {
		// Gate: wait out any active global rate-limit pause before sending.
		if waitCh := c.rl.peek(); waitCh != nil {
			select {
			case <-waitCh:
			case <-ctx.Done():
				return 0, &RateLimitTimeoutError{Cause: ctx.Err()}
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, endpoint, readerOf(data))
		if err != nil {
			return 0, fmt.Errorf("create request: %w", err)
		}

		c.applyAuth(req)
		if data != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("execute request: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := parseRetryAfter(resp)
			_ = resp.Body.Close()

			waitCh := c.rl.activate(retryAfter)
			select {
			case <-waitCh:
				continue // retry the request
			case <-ctx.Done():
				return http.StatusTooManyRequests, &RateLimitTimeoutError{Cause: ctx.Err()}
			}
		}

		if resp.StatusCode >= 300 {
			statusCode := resp.StatusCode
			err := c.errorFromResponse(resp)
			_ = resp.Body.Close()
			return statusCode, err
		}

		if out == nil {
			_ = resp.Body.Close()
			return resp.StatusCode, nil
		}

		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			_ = resp.Body.Close()
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}

		_ = resp.Body.Close()
		return resp.StatusCode, nil
	}
}

// marshalPayload encodes payload as JSON, returning nil bytes when payload is nil.
func marshalPayload(payload any) ([]byte, error) {
	if payload == nil {
		return nil, nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return data, nil
}

// readerOf wraps pre-marshaled bytes in a fresh Reader for each request attempt.
// Returns nil when data is nil (no request body).
func readerOf(data []byte) io.Reader {
	if data == nil {
		return nil
	}
	return bytes.NewReader(data)
}

// parseRetryAfter reads the Retry-After response header (seconds) and returns
// the corresponding duration. Falls back to 60 seconds if the header is absent
// or unparseable to avoid a busy-retry loop.
func parseRetryAfter(resp *http.Response) time.Duration {
	const fallback = 60 * time.Second
	s := resp.Header.Get("Retry-After")
	if s == "" {
		return fallback
	}
	secs, err := strconv.Atoi(s)
	if err != nil || secs <= 0 {
		return fallback
	}
	return time.Duration(secs) * time.Second
}
