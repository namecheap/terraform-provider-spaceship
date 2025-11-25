package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDomainInfo_basic(t *testing.T) {
	cfg := `
provider spaceship{}

data "spaceship_domain_info" "this" {
	domain = "dmytrovovk.com"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.spaceship_domain_info.this", "name", "dmytrovovk.com"),
				),
			},
		},
	})

}
