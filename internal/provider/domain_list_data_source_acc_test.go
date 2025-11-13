package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatasourceDomainList_basic(t *testing.T) {
	cfg := `
provider "spaceship" {}

data "spaceship_domain_list" "this"{}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.spaceship_domain_list.this", "total", "1"),
					resource.TestCheckResourceAttr("data.spaceship_domain_list.this", "items.#", "1"),
					resource.TestCheckResourceAttr("data.spaceship_domain_list.this", "items.0.name", "dmytrovovk.com"),
				),
			},
		},
	})
}

func TestAccDomainListDataSource_Unconfigured(t *testing.T) {
	cfg := `
data "spaceship_domain_list" "this" {}
`

	t.Run("missing_api_key", func(t *testing.T) {
		t.Setenv("SPACESHIP_API_KEY", "")
		t.Setenv("SPACESHIP_API_SECRET", "some-secret")

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      cfg,
					ExpectError: regexp.MustCompile("Missing Spaceship API key"),
				},
			},
		})
	})

	t.Run("missing_api_secret", func(t *testing.T) {
		t.Setenv("SPACESHIP_API_KEY", "some-key")
		t.Setenv("SPACESHIP_API_SECRET", "")

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      cfg,
					ExpectError: regexp.MustCompile("Missing Spaceship API secret"),
				},
			},
		})
	})

	t.Run("missing_both", func(t *testing.T) {
		t.Setenv("SPACESHIP_API_KEY", "")
		t.Setenv("SPACESHIP_API_SECRET", "")

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      cfg,
					ExpectError: regexp.MustCompile("Missing Spaceship API (key|secret)"),
				},
			},
		})
	})
}
