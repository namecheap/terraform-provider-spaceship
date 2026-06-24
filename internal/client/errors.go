package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SpaceshipApiError represents a non-2xx error response from the Spaceship API.
// Status is the HTTP status code; Message is the trimmed response body and may
// be empty.
type SpaceshipApiError struct {
	Status  int
	Message string
}

// Error implements the error interface.
func (e *SpaceshipApiError) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Message != "" {
		return fmt.Sprintf("spaceship api error (status %d): %s", e.Status, e.Message)
	}

	return fmt.Sprintf("spaceship api error (status %d)", e.Status)
}

// errorFromResponse builds a *SpaceshipApiError from a non-2xx response. The
// body is read up to 64 KiB (LimitReader guards against an unexpectedly large
// body) and used, trimmed, as the error Message.
func (c *Client) errorFromResponse(resp *http.Response) error {
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return &SpaceshipApiError{
			Status: resp.StatusCode,
		}
	}

	return &SpaceshipApiError{
		Status:  resp.StatusCode,
		Message: strings.TrimSpace(string(data)),
	}
}

// IsNotFoundError reports whether err is a *SpaceshipApiError with HTTP 404.
// Callers use it to treat "already gone" as success on delete paths.
func IsNotFoundError(err error) bool {
	var apiErr *SpaceshipApiError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Status == http.StatusNotFound
}
