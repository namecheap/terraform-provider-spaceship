package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDomain_autorenewal(t *testing.T) {
	template := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

	auto_renew = false
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: template,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "auto_renew", "false"),
				),
			},
			{
				Config:             template,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})

}
