package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestDomainResource_ImportState verifies that a domain resource can be
// imported by its domain name and that the state is populated correctly.
func TestDomainResource_ImportState(t *testing.T) {
	server := mockDomainAPIReadOnly(t, baseDomainInfo())

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create the resource so there is state to import into.
			{
				Config: `
provider "spaceship" {}

resource "spaceship_domain" "test" {
  domain = "example.com"
}
`,
			},
			// Step 2: Import by domain name — exercises ImportState.
			{
				ResourceName:                         "spaceship_domain.test",
				ImportState:                          true,
				ImportStateId:                        "example.com",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "domain",
			},
		},
	})
}

// mockDomainAPIReadOnly creates a simple deterministic mock that always
// returns the same domain info for GET requests. No state mutation.
func mockDomainAPIReadOnly(t *testing.T, domain client.DomainInfo) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/domains/") {
			_ = json.NewEncoder(w).Encode(domain)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	return server
}
