package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const dnsUpdateStep1Config = `
provider "spaceship" {}

resource "spaceship_dns_records" "test" {
  domain = "example.com"

  records = [
    {
      type    = "A"
      name    = "www"
      address = "1.2.3.4"
      ttl     = 3600
    },
  ]
}
`

const dnsUpdateStep2Config = `
provider "spaceship" {}

resource "spaceship_dns_records" "test" {
  domain = "example.com"

  records = [
    {
      type    = "A"
      name    = "www"
      address = "5.6.7.8"
      ttl     = 3600
    },
  ]
}
`

// mockDNSAPIUpdateGetError creates a mock where the create step works
// but the first GET during Update (read existing records) fails.
func mockDNSAPIUpdateGetError(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	records := []client.DNSRecord{}
	var getCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		if !strings.Contains(r.URL.Path, "/dns/records/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			n := getCalls.Add(1)
			// Create: 2 GETs + post-create Read: 1 GET = 3 GETs.
			// Update pre-apply Read: 1 GET = 4th.
			// Update method's GetDNSRecords: 5th → fail.
			// GETs after that succeed for cleanup.
			if n == 5 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error": "get records error"}`))
				return
			}
			_ = json.NewEncoder(w).Encode(struct {
				Items []client.DNSRecord `json:"items"`
				Total int                `json:"total"`
			}{Items: records, Total: len(records)})
		case http.MethodPut:
			var body struct {
				Force bool               `json:"force"`
				Items []client.DNSRecord `json:"items"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			records = body.Items
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// mockDNSAPIUpdateUpsertError creates a mock where the create step works
// but the PUT during Update (upsert changed records) fails.
func mockDNSAPIUpdateUpsertError(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	records := []client.DNSRecord{}
	var putCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		if !strings.Contains(r.URL.Path, "/dns/records/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(struct {
				Items []client.DNSRecord `json:"items"`
				Total int                `json:"total"`
			}{Items: records, Total: len(records)})
		case http.MethodPut:
			n := putCalls.Add(1)
			if n > 1 {
				// First PUT is from Create, second is from Update → fail.
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error": "upsert failed"}`))
				return
			}
			var body struct {
				Force bool               `json:"force"`
				Items []client.DNSRecord `json:"items"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			records = body.Items
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// mockDNSAPIUpdateRefreshError creates a mock where the create step and the
// Update diff/upsert work, but the refresh GET after upsert in Update fails.
func mockDNSAPIUpdateRefreshError(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	records := []client.DNSRecord{}
	var getCalls atomic.Int32
	var putCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		if !strings.Contains(r.URL.Path, "/dns/records/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			n := getCalls.Add(1)
			// Create: 2 GETs + post-create Read: 1 GET = 3.
			// Update pre-apply Read: 1 GET = 4th.
			// Update GetDNSRecords (existing): 5th.
			// Update GetDNSRecords (refresh after upsert): 6th → fail.
			// GETs after that succeed for cleanup.
			if n == 6 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error": "refresh error"}`))
				return
			}
			_ = json.NewEncoder(w).Encode(struct {
				Items []client.DNSRecord `json:"items"`
				Total int                `json:"total"`
			}{Items: records, Total: len(records)})
		case http.MethodPut:
			putCalls.Add(1)
			var body struct {
				Force bool               `json:"force"`
				Items []client.DNSRecord `json:"items"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			records = body.Items
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// TestDNSRecordsResource_UpdateGetRecordsError verifies that Update returns
// an error when the initial GetDNSRecords call (read existing) fails.
func TestDNSRecordsResource_UpdateGetRecordsError(t *testing.T) {
	server := mockDNSAPIUpdateGetError(t)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: dnsUpdateStep1Config,
			},
			{
				Config:      dnsUpdateStep2Config,
				ExpectError: regexp.MustCompile(`failed to read existing DNS Records`),
			},
		},
	})
}

// TestDNSRecordsResource_UpdateUpsertError verifies that Update returns
// an error when the UpsertDNSRecords call fails.
func TestDNSRecordsResource_UpdateUpsertError(t *testing.T) {
	server := mockDNSAPIUpdateUpsertError(t)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: dnsUpdateStep1Config,
			},
			{
				Config:      dnsUpdateStep2Config,
				ExpectError: regexp.MustCompile(`Failed to update DNS records`),
			},
		},
	})
}

// TestDNSRecordsResource_UpdateRefreshError verifies that Update returns
// an error when the post-upsert refresh GetDNSRecords call fails.
func TestDNSRecordsResource_UpdateRefreshError(t *testing.T) {
	server := mockDNSAPIUpdateRefreshError(t)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: dnsUpdateStep1Config,
			},
			{
				Config:      dnsUpdateStep2Config,
				ExpectError: regexp.MustCompile(`Failed to refresh DNS records`),
			},
		},
	})
}
