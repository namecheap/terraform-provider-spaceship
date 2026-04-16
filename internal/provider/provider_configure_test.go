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
