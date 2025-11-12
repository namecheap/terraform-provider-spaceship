package provider

import (
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
