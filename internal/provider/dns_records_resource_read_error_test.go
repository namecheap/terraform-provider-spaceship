package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mockDNSAPIWithReadNon404Error returns a mock where create succeeds but
// subsequent Read GETs return a non-404 error (500). This covers the
// Read error branch that is NOT a 404 (the IsNotFoundError false path).
func mockDNSAPIWithReadNon404Error(t *testing.T) *httptest.Server {
	t.Helper()
	var records []client.DNSRecord
	var getCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if !strings.Contains(r.URL.Path, "/dns/records/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			n := getCalls.Add(1)
			// Create: 2 GETs + post-create Read: 1 GET = 3.
			// Next Read (plan refresh in step 2): 4th → fail with 500.
			// After that, succeed for cleanup.
			if n == 4 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error": "server error"}`))
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
			records = nil
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// TestDNSRecordsResource_ReadNon404Error verifies that when Read encounters
// a non-404 error (e.g. 500), it returns a diagnostic error rather than
// removing the resource from state.
func TestDNSRecordsResource_ReadNon404Error(t *testing.T) {
	server := mockDNSAPIWithReadNon404Error(t)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create succeeds normally.
			{
				Config: dnsTestConfig,
			},
			// Step 2: The Read during plan refresh returns 500.
			{
				Config:      dnsTestConfig,
				ExpectError: regexp.MustCompile(`Failed to read DNS records`),
			},
		},
	})
}
