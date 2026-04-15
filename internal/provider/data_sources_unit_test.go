package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mockDataSourceAPI creates a mock Spaceship API server that handles:
//   - GET /v1/domains/{domain} — returns a single DomainInfo
//   - GET /v1/domains          — returns a DomainList with the provided items
func mockDataSourceAPI(t *testing.T, domains []client.DomainInfo) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// GET /v1/domains?take=...&skip=...&orderBy=... — domain list
		if r.URL.Path == "/v1/domains" {
			resp := client.DomainList{
				Items: domains,
				Total: int64(len(domains)),
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// GET /v1/domains/{name} — single domain info
		// The path has the form /v1/domains/<name>
		for _, d := range domains {
			if r.URL.Path == "/v1/domains/"+d.Name {
				_ = json.NewEncoder(w).Encode(d)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	t.Cleanup(server.Close)
	return server
}

func TestDomainInfoDataSource_Read(t *testing.T) {
	domain := baseDomainInfo()
	server := mockDataSourceAPI(t, []client.DomainInfo{domain})

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "spaceship" {}

data "spaceship_domain_info" "test" {
  domain = "example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.spaceship_domain_info.test", "domain", "example.com"),
					resource.TestCheckResourceAttr("data.spaceship_domain_info.test", "name", "example.com"),
					resource.TestCheckResourceAttr("data.spaceship_domain_info.test", "lifecycle_status", "registered"),
					resource.TestCheckResourceAttr("data.spaceship_domain_info.test", "auto_renew", "false"),
					resource.TestCheckResourceAttr("data.spaceship_domain_info.test", "registration_date", "2024-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr("data.spaceship_domain_info.test", "expiration_date", "2025-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr("data.spaceship_domain_info.test", "nameservers.provider", "basic"),
					resource.TestCheckResourceAttr("data.spaceship_domain_info.test", "contacts.registrant", "handle-123"),
					resource.TestCheckResourceAttr("data.spaceship_domain_info.test", "privacy_protection.level", "high"),
				),
			},
		},
	})
}

func TestDomainListDataSource_Read(t *testing.T) {
	domain := baseDomainInfo()
	server := mockDataSourceAPI(t, []client.DomainInfo{domain})

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "spaceship" {}

data "spaceship_domain_list" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.spaceship_domain_list.test", "total", "1"),
					resource.TestCheckResourceAttr("data.spaceship_domain_list.test", "items.#", "1"),
					resource.TestCheckResourceAttr("data.spaceship_domain_list.test", "items.0.name", "example.com"),
					resource.TestCheckResourceAttr("data.spaceship_domain_list.test", "items.0.lifecycle_status", "registered"),
					resource.TestCheckResourceAttr("data.spaceship_domain_list.test", "items.0.auto_renew", "false"),
				),
			},
		},
	})
}
