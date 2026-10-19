package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestConfigure_MissingAPIKey(t *testing.T) {
	t.Setenv("SPACESHIP_API_KEY", "")
	t.Setenv("SPACESHIP_API_SECRET", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      `data "spaceship_domain_list" "test" {}`,
				ExpectError: regexp.MustCompile("Missing Spaceship API key"),
			},
		},
	})
}

func TestConfigure_MissingAPISecret(t *testing.T) {
	t.Setenv("SPACESHIP_API_KEY", "some-key")
	t.Setenv("SPACESHIP_API_SECRET", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      `data "spaceship_domain_list" "test" {}`,
				ExpectError: regexp.MustCompile("Missing Spaceship API secret"),
			},
		},
	})
}

// Credentials taken from a value known only after apply arrive in Configure as
// unknown. They must not fall back to the environment: plan would run with the
// env account while apply uses the configured one.
func TestConfigure_UnknownCredentialsDoNotFallBackToEnv(t *testing.T) {
	t.Setenv("SPACESHIP_API_KEY", "env-key")
	t.Setenv("SPACESHIP_API_SECRET", "env-secret")
	// Any request that slips through fails fast instead of reaching the real API.
	t.Setenv("SPACESHIP_BASE_URL", "http://127.0.0.1:1")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
resource "terraform_data" "credentials" {
  input = "configured"
}

provider "spaceship" {
  api_key    = terraform_data.credentials.output
  api_secret = terraform_data.credentials.output
}

resource "spaceship_dns_record" "test" {
  domain  = "example.com"
  type    = "A"
  name    = "www"
  address = "192.0.2.1"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ExpectError:        regexp.MustCompile("Unknown Spaceship API key"),
			},
		},
	})
}

func TestConfigure_MissingBothCredentials(t *testing.T) {
	t.Setenv("SPACESHIP_API_KEY", "")
	t.Setenv("SPACESHIP_API_SECRET", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      `data "spaceship_domain_list" "test" {}`,
				ExpectError: regexp.MustCompile("Missing Spaceship API"),
			},
		},
	})
}
