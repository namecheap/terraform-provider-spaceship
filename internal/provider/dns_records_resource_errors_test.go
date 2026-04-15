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

// mockDNSAPIWithGetError returns a mock server where GET /v1/dns/records/
// always fails with 500.
func mockDNSAPIWithGetError(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/dns/records/") {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "internal server error"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	return server
}

// mockDNSAPIWithUpsertError returns a mock server where GET succeeds (empty
// records) but PUT /v1/dns/records/ fails with 500.
func mockDNSAPIWithUpsertError(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if !strings.Contains(r.URL.Path, "/dns/records/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(struct {
				Items []client.DNSRecord `json:"items"`
				Total int                `json:"total"`
			}{Items: []client.DNSRecord{}, Total: 0})
		case http.MethodPut:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "upsert failed"}`))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// mockDNSAPIWithDeleteError returns a mock server where GET and PUT succeed
// but DELETE (used when diffing records) fails with 500.
func mockDNSAPIWithDeleteError(t *testing.T, initialRecords []client.DNSRecord) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	records := make([]client.DNSRecord, len(initialRecords))
	copy(records, initialRecords)

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
		case http.MethodDelete:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "delete failed"}`))
		case http.MethodPut:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// mockDNSAPIWithClearError returns a mock server that works normally for
// create (GET + PUT) but fails on the first DELETE during ClearDNSRecords.
// Subsequent DELETEs succeed so post-test cleanup works.
func mockDNSAPIWithClearError(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	records := []client.DNSRecord{}
	var deleteFailed atomic.Bool

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
			// Fail the first DELETE to test error handling in ClearDNSRecords,
			// then succeed on subsequent DELETEs so post-test cleanup works.
			if !deleteFailed.Load() {
				deleteFailed.Store(true)
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error": "server error during clear"}`))
				return
			}
			records = nil
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// mockDNSAPIWithNotFound returns a mock server that works normally for create
// but returns 404 on subsequent GETs (simulating domain deletion).
func mockDNSAPIWithNotFound(t *testing.T) *httptest.Server {
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
			// Create calls GET twice + Read-after-create once = 3 GETs.
			// After that return 404 to simulate domain removal.
			if n > 3 {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`not found`))
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

// mockDNSAPIWithReadError returns a mock server that works normally for
// create (GET + PUT) but fails with a non-404 error on the Read-after-create
// GET (the 3rd GET call).
func mockDNSAPIWithReadError(t *testing.T) *httptest.Server {
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
			// Let the first Create GET succeed (read existing records).
			// Fail on the 2nd GET (refresh after upsert inside Create).
			if n > 1 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error": "read error"}`))
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

const dnsTestConfig = `
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

// TestDNSRecordsResource_CreateGetRecordsError verifies that Create returns a
// diagnostic error when the initial GetDNSRecords call (to read existing
// records) fails.
func TestDNSRecordsResource_CreateGetRecordsError(t *testing.T) {
	server := mockDNSAPIWithGetError(t)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      dnsTestConfig,
				ExpectError: regexp.MustCompile(`failed to read existing DNS records`),
			},
		},
	})
}

// TestDNSRecordsResource_CreateUpsertError verifies that Create returns a
// diagnostic error when the UpsertDNSRecords call fails (GET succeeds, PUT
// fails).
func TestDNSRecordsResource_CreateUpsertError(t *testing.T) {
	server := mockDNSAPIWithUpsertError(t)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      dnsTestConfig,
				ExpectError: regexp.MustCompile(`Spaceship API error`),
			},
		},
	})
}

// TestDNSRecordsResource_CreateDeleteDiffError verifies that Create returns a
// diagnostic error when the DeleteDNSRecords call (for removing records that
// exist on the server but are not in the plan) fails.
func TestDNSRecordsResource_CreateDeleteDiffError(t *testing.T) {
	// Pre-populate with a record that is NOT in the plan, so diffDNSRecords
	// produces a non-empty toDelete list.
	existingRecords := []client.DNSRecord{
		{
			Type:    "CNAME",
			Name:    "old",
			CName:   "old.example.com",
			TTL:     3600,
		},
	}
	server := mockDNSAPIWithDeleteError(t, existingRecords)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      dnsTestConfig,
				ExpectError: regexp.MustCompile(`Failed to delete DNS records`),
			},
		},
	})
}

// TestDNSRecordsResource_CreateRefreshError verifies that Create returns a
// diagnostic error when the post-upsert refresh GetDNSRecords call fails.
// The Create method calls GET three times:
//  1. Read existing records (before diff)
//  2. Refresh after upsert
//
// This test makes the second GET fail.
func TestDNSRecordsResource_CreateRefreshError(t *testing.T) {
	server := mockDNSAPIWithReadError(t)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      dnsTestConfig,
				ExpectError: regexp.MustCompile(`Failed to refresh DNS records`),
			},
		},
	})
}

// TestDNSRecordsResource_ReadNotFound verifies that when GetDNSRecords returns
// a 404 during Read, the resource is removed from state (rather than
// returning an error). This covers the IsNotFoundError branch in Read.
func TestDNSRecordsResource_ReadNotFound(t *testing.T) {
	server := mockDNSAPIWithNotFound(t)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create records normally (first 3 GETs succeed).
			{
				Config: dnsTestConfig,
			},
			// Step 2: Refresh state only (no apply). Read will call GET
			// which now returns 404 and removes the resource from state,
			// causing Terraform to plan re-creation.
			{
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestDNSRecordsResource_DeleteClearError verifies that Delete returns a
// diagnostic error when ClearDNSRecords fails (because the underlying GET
// during clear returns 500).
func TestDNSRecordsResource_DeleteClearError(t *testing.T) {
	server := mockDNSAPIWithClearError(t)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create records normally.
			{
				Config: dnsTestConfig,
			},
			// Step 2: Destroy. The ClearDNSRecords call will fail because
			// the mock returns 500 on GET after the first 3 calls.
			{
				Config:  dnsTestConfig,
				Destroy: true,
				ExpectError: regexp.MustCompile(`Failed to clear DNS records`),
			},
		},
	})
}
