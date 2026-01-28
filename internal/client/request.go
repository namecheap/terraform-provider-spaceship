package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (c *Client) doJSON(ctx context.Context, method, endpoint string, payload any, out any) (int, error) {
	status := 0
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return status, fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return status, fmt.Errorf("create request: %w", err)
	}

	c.applyAuth(req)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return status, fmt.Errorf("execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	status = resp.StatusCode
	if resp.StatusCode >= 300 {
		return status, c.errorFromResponse(resp)
	}

	if out == nil {
		return status, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return status, fmt.Errorf("decode response: %w", err)
	}

	return status, nil
}
