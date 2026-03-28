package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mockSpaceshipAPI creates a mock Spaceship API server that simulates
// eventual consistency: the autorenew PUT endpoint confirms the change,
// but the domain GET endpoint returns stale data for autoRenew.
func mockSpaceshipAPI(t *testing.T, staleDomainInfo client.DomainInfo) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// PUT /v1/domains/{domain}/autorenew — always confirms the requested value
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/autorenew"):
			var body struct {
				IsEnabled bool `json:"isEnabled"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(client.AutoRenewalResponse{IsEnabled: body.IsEnabled})

		// PUT /v1/domains/{domain}/nameservers — always succeeds
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/nameservers"):
			w.WriteHeader(http.StatusOK)

		// GET /v1/domains/{domain} — returns stale data (simulates eventual consistency)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/domains/"):
			_ = json.NewEncoder(w).Encode(staleDomainInfo)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	t.Cleanup(server.Close)
	return server
}

func testMockProviderFactories(serverURL string) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"spaceship": func() (tfprotov6.ProviderServer, error) {
			return providerserver.NewProtocol6WithError(New("test")())()
		},
	}
}

// TestDomainResource_AutoRenewEventualConsistency verifies that the provider
// handles eventual consistency when updating auto_renew. The API PUT endpoint
// confirms the change, but the subsequent GET may return stale data. The
// provider must trust the plan value instead of the stale read.
func TestDomainResource_AutoRenewEventualConsistency(t *testing.T) {
	// Mock API always returns autoRenew=false on GET (simulates stale read)
	staleDomain := client.DomainInfo{
		Name:            "example.com",
		UnicodeName:     "example.com",
		AutoRenew:       false, // stale: stays false even after PUT sets it to true
		RegistrationDate: "2024-01-01T00:00:00Z",
		ExpirationDate:   "2025-01-01T00:00:00Z",
		LifecycleStatus:  "registered",
		EPPStatuses:      []string{},
		Suspensions:      []client.ReasonCode{},
		Nameservers: client.Nameservers{
			Provider: "basic",
			Hosts:    []string{"launch1.spaceship.net", "launch2.spaceship.net"},
		},
		Contacts: client.Contacts{
			Registrant: "handle-123",
		},
		PrivacyProtection: client.PrivacyProtection{
			Level: "high",
		},
	}

	server := mockSpaceshipAPI(t, staleDomain)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(server.URL),
		Steps: []resource.TestStep{
			// Step 1: Import the domain with auto_renew unset — reads current state (false)
			{
				Config: `
provider "spaceship" {}

resource "spaceship_domain" "test" {
  domain = "example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.test", "auto_renew", "false"),
				),
			},
			// Step 2: Set auto_renew=true — PUT succeeds, but GET still returns false.
			// Without the fix, this produces "inconsistent result after apply" error.
			// With the fix, the provider trusts the plan value.
			{
				Config: `
provider "spaceship" {}

resource "spaceship_domain" "test" {
  domain     = "example.com"
  auto_renew = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.test", "auto_renew", "true"),
				),
			},
			// Step 3: Set auto_renew=false — verifies the reverse direction also works
			{
				Config: `
provider "spaceship" {}

resource "spaceship_domain" "test" {
  domain     = "example.com"
  auto_renew = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.test", "auto_renew", "false"),
				),
			},
		},
	})
}

// TestDomainResource_NameserversEventualConsistency verifies that the provider
// handles eventual consistency when updating nameservers. The API PUT succeeds,
// but the subsequent GET returns stale nameserver data.
func TestDomainResource_NameserversEventualConsistency(t *testing.T) {
	// Mock API always returns basic nameservers on GET (simulates stale read)
	staleDomain := client.DomainInfo{
		Name:            "example.com",
		UnicodeName:     "example.com",
		AutoRenew:       false,
		RegistrationDate: "2024-01-01T00:00:00Z",
		ExpirationDate:   "2025-01-01T00:00:00Z",
		LifecycleStatus:  "registered",
		EPPStatuses:      []string{},
		Suspensions:      []client.ReasonCode{},
		Nameservers: client.Nameservers{
			Provider: "basic",
			Hosts:    []string{"launch1.spaceship.net", "launch2.spaceship.net"},
		},
		Contacts: client.Contacts{
			Registrant: "handle-123",
		},
		PrivacyProtection: client.PrivacyProtection{
			Level: "high",
		},
	}

	server := mockSpaceshipAPI(t, staleDomain)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(server.URL),
		Steps: []resource.TestStep{
			// Step 1: Import with basic nameservers
			{
				Config: `
provider "spaceship" {}

resource "spaceship_domain" "test" {
  domain = "example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.test", "nameservers.provider", "basic"),
				),
			},
			// Step 2: Switch to custom nameservers — PUT succeeds, GET returns stale basic.
			// Without the fix, this produces "inconsistent result after apply".
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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.test", "nameservers.provider", "custom"),
					resource.TestCheckTypeSetElemAttr("spaceship_domain.test", "nameservers.hosts.*", "ns1.example.com"),
					resource.TestCheckTypeSetElemAttr("spaceship_domain.test", "nameservers.hosts.*", "ns2.example.com"),
				),
			},
		},
	})
}
