package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Represents an error response from the Spaceship API.
type SpaceshipApiError struct {
	Status  int
	Message string
}

func (e *SpaceshipApiError) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Message != "" {
		return fmt.Sprintf("spaceship api error (status %d): %s", e.Status, e.Message)
	}

	return fmt.Sprintf("spaceship api error (status %d)", e.Status)
}

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

// returns true if the err represents 404 response
func IsNotFoundError(err error) bool {
	var apiErr *SpaceshipApiError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Status == http.StatusNotFound
}
