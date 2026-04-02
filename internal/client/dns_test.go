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

func TestGetDNSRecords_FiltersOutNonCustomGroups(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"type": "A", "name": "@", "ttl": 3600, "address": "1.2.3.4", "group": map[string]any{"type": "custom"}},
				{"type": "A", "name": "*", "ttl": 3600, "address": "15.197.162.184", "group": map[string]any{"type": "product"}},
				{"type": "A", "name": "www", "ttl": 3600, "address": "5.6.7.8", "group": map[string]any{"type": "custom"}},
				{"type": "NS", "name": "@", "ttl": 3600, "nameserver": "ns1.example.com", "group": map[string]any{"type": "personalNS"}},
			},
			"total": 4,
		})
	})

	records, err := c.GetDNSRecords(t.Context(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 custom records, got %d", len(records))
	}
	if records[0].Address != "1.2.3.4" {
		t.Errorf("expected first record address 1.2.3.4, got %q", records[0].Address)
	}
	if records[1].Address != "5.6.7.8" {
		t.Errorf("expected second record address 5.6.7.8, got %q", records[1].Address)
	}
}

func TestGetDNSRecords_KeepsRecordsWithoutGroup(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"type": "A", "name": "@", "ttl": 3600, "address": "1.2.3.4"},
				{"type": "TXT", "name": "@", "ttl": 3600, "value": "v=spf1"},
			},
			"total": 2,
		})
	})

	records, err := c.GetDNSRecords(t.Context(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records (no group = kept), got %d", len(records))
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

func TestFilterCustomDNSRecords(t *testing.T) {
	records := []DNSRecord{
		{Type: "A", Name: "@", Address: "1.1.1.1", Group: &RecordGroup{Type: "custom"}},
		{Type: "A", Name: "*", Address: "15.197.162.184", Group: &RecordGroup{Type: "product"}},
		{Type: "TXT", Name: "@", Value: "v=spf1"},
		{Type: "NS", Name: "@", Nameserver: "ns1.example.com", Group: &RecordGroup{Type: "personalNS"}},
		{Type: "MX", Name: "@", Exchange: "mail.example.com", Group: &RecordGroup{Type: "custom"}},
	}

	filtered := filterCustomDNSRecords(records)
	if len(filtered) != 3 {
		t.Fatalf("expected 3 records (2 custom + 1 nil group), got %d", len(filtered))
	}
	if filtered[0].Address != "1.1.1.1" {
		t.Errorf("expected first record to be custom A, got address %q", filtered[0].Address)
	}
	if filtered[1].Value != "v=spf1" {
		t.Errorf("expected second record to be ungrouped TXT, got value %q", filtered[1].Value)
	}
	if filtered[2].Exchange != "mail.example.com" {
		t.Errorf("expected third record to be custom MX, got exchange %q", filtered[2].Exchange)
	}
}

func TestFilterCustomDNSRecords_AllFiltered(t *testing.T) {
	records := []DNSRecord{
		{Type: "A", Name: "@", Address: "15.197.162.184", Group: &RecordGroup{Type: "product"}},
		{Type: "NS", Name: "@", Nameserver: "ns1.example.com", Group: &RecordGroup{Type: "personalNS"}},
	}

	filtered := filterCustomDNSRecords(records)
	if len(filtered) != 0 {
		t.Fatalf("expected 0 records, got %d", len(filtered))
	}
}

func TestFilterCustomDNSRecords_Empty(t *testing.T) {
	filtered := filterCustomDNSRecords(nil)
	if len(filtered) != 0 {
		t.Fatalf("expected 0 records for nil input, got %d", len(filtered))
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
