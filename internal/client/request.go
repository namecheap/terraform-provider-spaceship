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
	var bodyBytes []byte
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, fmt.Errorf("marshal payload: %w", err)
		}
		bodyBytes = data
	}

	for attempt := range maxRetries {
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

		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRetries-1 {
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

// retryDelay reads the Retry-After header from a 429 response and returns
// the appropriate delay, capped to maxRetryDelay. Falls back to
// defaultRetryDelay if the header is missing or unparseable.
func retryDelay(resp *http.Response) time.Duration {
	if v := resp.Header.Get("Retry-After"); v != "" {
		if seconds, err := strconv.Atoi(v); err == nil && seconds > 0 {
			delay := time.Duration(seconds) * time.Second
			if delay > maxRetryDelay {
				return maxRetryDelay
			}
			return delay
		}
	}
	return defaultRetryDelay
}
