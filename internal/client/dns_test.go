package client

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetDNSRecords_SinglePage(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"type": "A", "name": "@", "ttl": 3600, "address": "1.2.3.4"},
			},
			"total": 1,
		})
	})

	records, err := c.GetDNSRecords(t.Context(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Address != "1.2.3.4" {
		t.Errorf("expected address 1.2.3.4, got %q", records[0].Address)
	}
}

func TestGetDNSRecords_Pagination(t *testing.T) {
	callCount := 0
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			items := make([]map[string]any, maxListPageSize)
			for i := range items {
				items[i] = map[string]any{"type": "A", "name": "@", "ttl": 3600, "address": "1.2.3.4"}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": items,
				"total": maxListPageSize + 1,
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"type": "A", "name": "extra", "ttl": 3600, "address": "5.6.7.8"},
				},
				"total": maxListPageSize + 1,
			})
		}
	})

	records, err := c.GetDNSRecords(t.Context(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != maxListPageSize+1 {
		t.Fatalf("expected %d records, got %d", maxListPageSize+1, len(records))
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls for pagination, got %d", callCount)
	}
}

func TestGetDNSRecords_Error(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	})

	_, err := c.GetDNSRecords(t.Context(), "example.com")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpsertDNSRecords_Empty(t *testing.T) {
	err := (&Client{}).UpsertDNSRecords(t.Context(), "example.com", true, nil)
	if err != nil {
		t.Fatalf("expected no error for empty records, got: %v", err)
	}
}

func TestUpsertDNSRecords_SendsPayload(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var body struct {
			Force bool        `json:"force"`
			Items []DNSRecord `json:"items"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !body.Force {
			t.Error("expected force=true")
		}
		if len(body.Items) != 1 {
			t.Errorf("expected 1 item, got %d", len(body.Items))
		}
		w.WriteHeader(http.StatusOK)
	})

	err := c.UpsertDNSRecords(t.Context(), "example.com", true, []DNSRecord{
		{Type: "A", Name: "@", TTL: 3600, Address: "1.2.3.4"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteDNSRecords_Empty(t *testing.T) {
	err := (&Client{}).DeleteDNSRecords(t.Context(), "example.com", nil)
	if err != nil {
		t.Fatalf("expected no error for empty records, got: %v", err)
	}
}

func TestDeleteDNSRecords_NotFoundIgnored(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	err := c.DeleteDNSRecords(t.Context(), "example.com", []DNSRecord{
		{Type: "A", Name: "@"},
	})
	if err != nil {
		t.Fatalf("expected 404 to be ignored, got: %v", err)
	}
}

func TestDeleteDNSRecords_OtherErrorReturned(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	})

	err := c.DeleteDNSRecords(t.Context(), "example.com", []DNSRecord{
		{Type: "A", Name: "@"},
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
