package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mockDNSAPI creates a mock Spaceship DNS API server that stores records
// in memory and handles GET/PUT/DELETE on /v1/dns/records/{domain}.
func mockDNSAPI(t *testing.T, initialRecords []client.DNSRecord) *httptest.Server {
	t.Helper()

	var mu sync.Mutex
	records := make([]client.DNSRecord, len(initialRecords))
	copy(records, initialRecords)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		// Match /v1/dns/records/{domain}
		if !strings.HasPrefix(r.URL.Path, "/v1/dns/records/") {
			t.Logf("mock: unhandled path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			// Return current records with pagination wrapper
			resp := struct {
				Items []client.DNSRecord `json:"items"`
				Total int                `json:"total"`
			}{
				Items: records,
				Total: len(records),
			}
			_ = json.NewEncoder(w).Encode(resp)

		case http.MethodPut:
			// Upsert: decode the request and merge into stored records
			var body struct {
				Force bool               `json:"force"`
				Items []client.DNSRecord `json:"items"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Logf("mock: PUT decode error: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			for _, incoming := range body.Items {
				found := false
				for i, existing := range records {
					if strings.EqualFold(existing.Type, incoming.Type) &&
						strings.EqualFold(existing.Name, incoming.Name) {
						records[i] = incoming
						found = true
						break
					}
				}
				if !found {
					records = append(records, incoming)
				}
			}
			w.WriteHeader(http.StatusOK)

		case http.MethodDelete:
			// Delete: decode the records to remove
			var toDelete []client.DNSRecord
			if err := json.NewDecoder(r.Body).Decode(&toDelete); err != nil {
				t.Logf("mock: DELETE decode error: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			for _, del := range toDelete {
				for i, existing := range records {
					if strings.EqualFold(existing.Type, del.Type) &&
						strings.EqualFold(existing.Name, del.Name) &&
						strings.EqualFold(existing.Address, del.Address) &&
						strings.EqualFold(existing.CName, del.CName) {
						records = append(records[:i], records[i+1:]...)
						break
					}
				}
			}
			w.WriteHeader(http.StatusOK)

		default:
			t.Logf("mock: unhandled method %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	t.Cleanup(server.Close)
	return server
}

func TestDNSRecordsResource_CreateReadDelete(t *testing.T) {
	server := mockDNSAPI(t, []client.DNSRecord{})

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create an A record
			{
				Config: `
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
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "domain", "example.com"),
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "id", "example.com"),
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.#", "1"),
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.0.type", "A"),
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.0.name", "www"),
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.0.address", "1.2.3.4"),
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.0.ttl", "3600"),
				),
			},
			// Step 2: Import by domain name
			{
				ResourceName:            "spaceship_dns_records.test",
				ImportState:             true,
				ImportStateId:           "example.com",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force"},
			},
		},
	})
}

func TestDNSRecordsResource_Update(t *testing.T) {
	server := mockDNSAPI(t, []client.DNSRecord{})

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create an A record pointing to 1.2.3.4
			{
				Config: `
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
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.0.address", "1.2.3.4"),
				),
			},
			// Step 2: Update the A record to point to 5.6.7.8
			{
				Config: `
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
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.#", "1"),
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.0.address", "5.6.7.8"),
				),
			},
		},
	})
}

func TestDNSRecordsResource_MultipleRecordTypes(t *testing.T) {
	server := mockDNSAPI(t, []client.DNSRecord{})

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
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
    {
      type  = "CNAME"
      name  = "mail"
      cname = "mail.example.com"
      ttl   = 3600
    },
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.#", "2"),
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.0.type", "A"),
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.0.name", "www"),
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.0.address", "1.2.3.4"),
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.1.type", "CNAME"),
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.1.name", "mail"),
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.1.cname", "mail.example.com"),
				),
			},
		},
	})
}
