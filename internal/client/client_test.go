package client

import (
	"net/http"
	"testing"
)

func TestNewClient_ValidURL(t *testing.T) {
	c, err := NewClient("https://example.com/api/v1", "key", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.apiKey != "key" {
		t.Errorf("expected apiKey %q, got %q", "key", c.apiKey)
	}
	if c.apiSecret != "secret" {
		t.Errorf("expected apiSecret %q, got %q", "secret", c.apiSecret)
	}
	if c.httpClient == nil {
		t.Fatal("expected httpClient to be set")
	}
}

func TestNewClient_InvalidURL(t *testing.T) {
	_, err := NewClient("://bad", "key", "secret")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestApplyAuth(t *testing.T) {
	c, _ := NewClient("https://example.com", "mykey", "mysecret")
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	c.applyAuth(req)

	if got := req.Header.Get("X-API-Key"); got != "mykey" {
		t.Errorf("expected X-API-Key %q, got %q", "mykey", got)
	}
	if got := req.Header.Get("X-API-Secret"); got != "mysecret" {
		t.Errorf("expected X-API-Secret %q, got %q", "mysecret", got)
	}
}

func TestEndpointURL(t *testing.T) {
	c, _ := NewClient("https://api.example.com/v1", "k", "s")

	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{"no parts", nil, "https://api.example.com/v1"},
		{"single part", []string{"domains"}, "https://api.example.com/v1/domains"},
		{"multiple parts", []string{"domains", "example.com"}, "https://api.example.com/v1/domains/example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := c.endpointURL(tc.parts, nil)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
