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

const (
	maxRetries        = 3
	defaultRetryDelay = 5 * time.Second
	maxRetryDelay     = 30 * time.Second
)

func (c *Client) doJSON(ctx context.Context, method, endpoint string, payload any, out any) (int, error) {
	return c.doJSONWithRetries(ctx, method, endpoint, payload, out, maxRetries)
}

// doJSONOnce performs a single HTTP request without retrying on 429.
// Use this when the caller has its own fallback logic for rate limits.
func (c *Client) doJSONOnce(ctx context.Context, method, endpoint string, payload any, out any) (int, error) {
	return c.doJSONWithRetries(ctx, method, endpoint, payload, out, 1)
}

func (c *Client) doJSONWithRetries(ctx context.Context, method, endpoint string, payload any, out any, retries int) (int, error) {
	var bodyBytes []byte
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, fmt.Errorf("marshal payload: %w", err)
		}
		bodyBytes = data
	}

	for attempt := range retries {
		var body io.Reader
		if bodyBytes != nil {
			body = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
		if err != nil {
			return 0, fmt.Errorf("create request: %w", err)
		}

		c.applyAuth(req)
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("execute request: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < retries-1 {
			_ = resp.Body.Close()

			delay := retryDelay(resp)
			select {
			case <-ctx.Done():
				return resp.StatusCode, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode >= 300 {
			return resp.StatusCode, c.errorFromResponse(resp)
		}

		if out == nil {
			return resp.StatusCode, nil
		}

		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}

		return resp.StatusCode, nil
	}

	// unreachable — the last attempt always falls through to the status/error handling above
	return 0, fmt.Errorf("exhausted retries")
}

// retryDelay determines how long to wait before retrying a 429 response.
// It checks headers in order of specificity:
//  1. Retry-After — explicit wait time from the server
//  2. X-RateLimit-Reset — seconds until the rate limit window resets
//
// The result is capped to maxRetryDelay. Falls back to defaultRetryDelay
// if neither header is present or parseable.
func retryDelay(resp *http.Response) time.Duration {
	for _, header := range []string{"Retry-After", "X-RateLimit-Reset"} {
		if v := resp.Header.Get(header); v != "" {
			if seconds, err := strconv.Atoi(v); err == nil && seconds > 0 {
				delay := time.Duration(seconds) * time.Second
				if delay > maxRetryDelay {
					return maxRetryDelay
				}
				return delay
			}
		}
	}
	return defaultRetryDelay
}
