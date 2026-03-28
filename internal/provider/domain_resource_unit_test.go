package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mockSpaceshipAPIWithStaleReads creates a mock Spaceship API that simulates
// eventual consistency: after a PUT to autorenew or nameservers, the first
// GET still returns the old value (stale read). Subsequent GETs return the
// updated value. This reproduces the real-world race condition.
func mockSpaceshipAPIWithStaleReads(t *testing.T, initialDomain client.DomainInfo) *httptest.Server {
	t.Helper()

	var mu sync.Mutex
	currentDomain := initialDomain
	// Track pending updates: after a PUT, the next GET returns stale data,
	// then the update is applied for subsequent GETs.
	autoRenewPending := (*bool)(nil)
	nsPending := (*client.Nameservers)(nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		// PUT /v1/domains/{domain}/autorenew
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/autorenew"):
			var body struct {
				IsEnabled bool `json:"isEnabled"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			// Stage the change — will be visible after one stale GET
			autoRenewPending = &body.IsEnabled
			_ = json.NewEncoder(w).Encode(client.AutoRenewalResponse{IsEnabled: body.IsEnabled})

		// PUT /v1/domains/{domain}/nameservers
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/nameservers"):
			var body struct {
				Provider string   `json:"provider"`
				Hosts    []string `json:"hosts,omitempty"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			nsPending = &client.Nameservers{Provider: body.Provider, Hosts: body.Hosts}
			w.WriteHeader(http.StatusOK)

		// GET /v1/domains/{domain}
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/domains/"):
			// Return current (possibly stale) data first, then apply pending updates
			snapshot := currentDomain
			_ = json.NewEncoder(w).Encode(snapshot)

			// After serving the stale response, apply pending updates so the
			// next GET returns fresh data (simulating eventual consistency)
			if autoRenewPending != nil {
				currentDomain.AutoRenew = *autoRenewPending
				autoRenewPending = nil
			}
			if nsPending != nil {
				currentDomain.Nameservers = *nsPending
				nsPending = nil
			}

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	t.Cleanup(server.Close)
	return server
}

func testMockProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"spaceship": func() (tfprotov6.ProviderServer, error) {
			return providerserver.NewProtocol6WithError(New("test")())()
		},
	}
}

func baseDomainInfo() client.DomainInfo {
	return client.DomainInfo{
		Name:             "example.com",
		UnicodeName:      "example.com",
		AutoRenew:        false,
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
}

// TestDomainResource_AutoRenewEventualConsistency verifies that the provider
// handles eventual consistency when updating auto_renew. The API PUT endpoint
// confirms the change, but the first subsequent GET returns stale data. The
// provider must trust the plan value instead of the stale read.
//
// Without the fix in Update(), Terraform reports:
//
//	Error: Provider produced inconsistent result after apply
//	.auto_renew: was cty.True, but now cty.False.
func TestDomainResource_AutoRenewEventualConsistency(t *testing.T) {
	server := mockSpaceshipAPIWithStaleReads(t, baseDomainInfo())

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
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
			// Step 2: Set auto_renew=true — PUT succeeds, but the first GET still
			// returns false (stale). Without the fix, this produces
			// "inconsistent result after apply".
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
			// Step 3: Set auto_renew=false — verifies the reverse direction
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
// but the first subsequent GET returns stale nameserver data.
func TestDomainResource_NameserversEventualConsistency(t *testing.T) {
	server := mockSpaceshipAPIWithStaleReads(t, baseDomainInfo())

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
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
			// Step 2: Switch to custom nameservers — PUT succeeds, first GET
			// returns stale basic. Without the fix, this produces
			// "inconsistent result after apply".
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
