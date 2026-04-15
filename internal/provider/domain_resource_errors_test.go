package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mockDomainAPIWithError returns a mock that always fails domain GET requests.
func mockDomainAPIWithError(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/domains/") {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "domain read error"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	return server
}

// mockDomainAPIWithUpdateError returns a mock where GET succeeds but
// PUT to autorenew or nameservers fails.
func mockDomainAPIWithUpdateError(t *testing.T, domain client.DomainInfo) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/domains/"):
			_ = json.NewEncoder(w).Encode(domain)
		case r.Method == http.MethodPut:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "update failed"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// mockDomainAPIWithStaleRefresh returns a mock where GET and PUT succeed,
// but after an update the re-read of domain info fails.
func mockDomainAPIWithStaleRefresh(t *testing.T, domain client.DomainInfo) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	putSeen := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/autorenew"):
			putSeen = true
			_ = json.NewEncoder(w).Encode(client.AutoRenewalResponse{IsEnabled: true})

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/domains/"):
			// After the PUT, the re-read in Update fails.
			if putSeen {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error": "refresh error"}`))
				return
			}
			_ = json.NewEncoder(w).Encode(domain)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

const domainConfig = `
provider "spaceship" {}

resource "spaceship_domain" "test" {
  domain = "example.com"
}
`

// TestDomainResource_CreateGetInfoError verifies that Create returns an error
// when GetDomainInfo fails.
func TestDomainResource_CreateGetInfoError(t *testing.T) {
	server := mockDomainAPIWithError(t)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      domainConfig,
				ExpectError: regexp.MustCompile(`Unable to read domain info`),
			},
		},
	})
}

// TestDomainResource_UpdateAutoRenewError verifies that Update returns an
// error when the autorenew PUT fails.
func TestDomainResource_UpdateAutoRenewError(t *testing.T) {
	domain := baseDomainInfo()
	server := mockDomainAPIWithUpdateError(t, domain)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create (GET succeeds, no PUT needed).
			{
				Config: domainConfig,
			},
			// Step 2: Set auto_renew — PUT fails.
			{
				Config: `
provider "spaceship" {}

resource "spaceship_domain" "test" {
  domain     = "example.com"
  auto_renew = true
}
`,
				ExpectError: regexp.MustCompile(`Error updating domain auto_renew`),
			},
		},
	})
}

// TestDomainResource_UpdateNameserversError verifies that Update returns
// an error when the nameservers PUT fails.
func TestDomainResource_UpdateNameserversError(t *testing.T) {
	domain := baseDomainInfo()
	server := mockDomainAPIWithUpdateError(t, domain)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create with default nameservers.
			{
				Config: domainConfig,
			},
			// Step 2: Change nameservers — PUT fails.
			{
				Config: `
provider "spaceship" {}

resource "spaceship_domain" "test" {
  domain = "example.com"
  nameservers = {
    provider = "custom"
    hosts    = ["ns1.example.com", "ns2.example.com"]
  }
}
`,
				ExpectError: regexp.MustCompile(`Failed to update domain nameservers`),
			},
		},
	})
}

// TestDomainResource_UpdateRefreshError verifies that Update returns an
// error when the post-update GetDomainInfo re-read fails.
func TestDomainResource_UpdateRefreshError(t *testing.T) {
	domain := baseDomainInfo()
	server := mockDomainAPIWithStaleRefresh(t, domain)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: domainConfig,
			},
			{
				Config: `
provider "spaceship" {}

resource "spaceship_domain" "test" {
  domain     = "example.com"
  auto_renew = true
}
`,
				ExpectError: regexp.MustCompile(`Unable to read domain info`),
			},
		},
	})
}
