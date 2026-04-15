package provider

import (
	"testing"

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
