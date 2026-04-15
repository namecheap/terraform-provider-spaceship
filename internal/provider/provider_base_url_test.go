package provider

import (
	"testing"
)

func TestBaseURL_Default(t *testing.T) {
	t.Setenv("SPACESHIP_BASE_URL", "")

	got := baseURL()
	if got != defaultBaseURL {
		t.Errorf("expected %q, got %q", defaultBaseURL, got)
	}
}

func TestBaseURL_FromEnv(t *testing.T) {
	t.Setenv("SPACESHIP_BASE_URL", "http://localhost:8080/v1")

	got := baseURL()
	if got != "http://localhost:8080/v1" {
		t.Errorf("expected %q, got %q", "http://localhost:8080/v1", got)
	}
}

func TestBaseURL_InvalidURLStillReturned(t *testing.T) {
	t.Setenv("SPACESHIP_BASE_URL", "not-a-url")

	got := baseURL()
	if got != "not-a-url" {
		t.Errorf("expected %q, got %q", "not-a-url", got)
	}
}
