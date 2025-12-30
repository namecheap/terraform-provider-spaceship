package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// represents an error response from the spaceship api
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
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
		return &APIError{
			Status: resp.StatusCode,
		}
	}

	return &APIError{
		Status:  resp.StatusCode,
		Message: strings.TrimSpace(string(data)),
	}
}

// returns true if the err represents 404 response
func IsNotFoundError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Status == http.StatusNotFound
}
