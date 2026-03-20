package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-key", "test-secret")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	c.httpClient = server.Client()
	return c, server
}

func TestDoJSON_GetRequest(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("expected X-API-Key header")
		}
		if r.Header.Get("X-API-Secret") != "test-secret" {
			t.Errorf("expected X-API-Secret header")
		}
		if r.Header.Get("Content-Type") != "" {
			t.Errorf("GET should not set Content-Type")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"name": "test"})
	})

	var out map[string]string
	status, err := c.doJSON(context.Background(), http.MethodGet, c.endpointURL(nil, nil), nil, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 {
		t.Errorf("expected status 200, got %d", status)
	}
	if out["name"] != "test" {
		t.Errorf("expected name=test, got %q", out["name"])
	}
}

func TestDoJSON_PostWithPayload(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json")
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "value" {
			t.Errorf("expected key=value in body, got %v", body)
		}
		w.WriteHeader(http.StatusCreated)
	})

	status, err := c.doJSON(context.Background(), http.MethodPost, c.endpointURL(nil, nil), map[string]string{"key": "value"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 201 {
		t.Errorf("expected status 201, got %d", status)
	}
}

func TestDoJSON_ErrorResponse(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid request"))
	})

	_, err := c.doJSON(context.Background(), http.MethodGet, c.endpointURL(nil, nil), nil, nil)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !IsNotFoundError(err) && err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestDoJSON_NilOutput(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	status, err := c.doJSON(context.Background(), http.MethodDelete, c.endpointURL(nil, nil), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 204 {
		t.Errorf("expected status 204, got %d", status)
	}
}
