package client

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestSpaceshipApiError_Error_WithMessage(t *testing.T) {
	err := &SpaceshipApiError{Status: 400, Message: "bad request"}
	expected := "spaceship api error (status 400): bad request"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestSpaceshipApiError_Error_WithoutMessage(t *testing.T) {
	err := &SpaceshipApiError{Status: 500}
	expected := "spaceship api error (status 500)"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestSpaceshipApiError_Error_Nil(t *testing.T) {
	var err *SpaceshipApiError
	if err.Error() != "<nil>" {
		t.Errorf("expected %q, got %q", "<nil>", err.Error())
	}
}

func TestIsNotFoundError_True(t *testing.T) {
	err := &SpaceshipApiError{Status: http.StatusNotFound, Message: "not found"}
	if !IsNotFoundError(err) {
		t.Error("expected IsNotFoundError to return true for 404")
	}
}

func TestIsNotFoundError_False_OtherStatus(t *testing.T) {
	err := &SpaceshipApiError{Status: http.StatusBadRequest}
	if IsNotFoundError(err) {
		t.Error("expected IsNotFoundError to return false for 400")
	}
}

func TestIsNotFoundError_False_NonApiError(t *testing.T) {
	err := fmt.Errorf("some other error")
	if IsNotFoundError(err) {
		t.Error("expected IsNotFoundError to return false for non-API error")
	}
}

func TestIsNotFoundError_WrappedError(t *testing.T) {
	apiErr := &SpaceshipApiError{Status: http.StatusNotFound}
	wrapped := fmt.Errorf("wrapped: %w", apiErr)
	if !IsNotFoundError(wrapped) {
		t.Error("expected IsNotFoundError to return true for wrapped 404")
	}
}

func TestIsNotFoundError_Nil(t *testing.T) {
	if IsNotFoundError(nil) {
		t.Error("expected IsNotFoundError to return false for nil")
	}
}

func TestSpaceshipApiError_ImplementsError(t *testing.T) {
	var err error = &SpaceshipApiError{Status: 500}
	var apiErr *SpaceshipApiError
	if !errors.As(err, &apiErr) {
		t.Error("expected SpaceshipApiError to implement error interface")
	}
}
