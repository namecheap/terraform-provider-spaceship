package client

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestClearDNSRecords_DeletesAll(t *testing.T) {
	var calls []string
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method)
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"type": "A", "name": "@", "ttl": 3600, "address": "1.2.3.4"},
				},
				"total": 1,
			})
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		}
	})

	err := c.ClearDNSRecords(t.Context(), "example.com", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 || calls[0] != http.MethodGet || calls[1] != http.MethodDelete {
		t.Errorf("expected GET then DELETE, got %v", calls)
	}
}

func TestClearDNSRecords_NotFoundIgnored(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	err := c.ClearDNSRecords(t.Context(), "example.com", true)
	if err != nil {
		t.Fatalf("expected 404 to be ignored, got: %v", err)
	}
}

func TestClearDNSRecords_GetError(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	})

	err := c.ClearDNSRecords(t.Context(), "example.com", true)
	if err == nil {
		t.Fatal("expected error")
	}
}
