package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	providerName            = "spaceship"
	domainResource          = providerName + "_domain"
	domainResouceName       = "this"
	domainResourceReference = domainResouceName + "." + domainResouceName
)

func TestAccDomain_basic(t *testing.T) {
	domainName := testAccDomainValue()

	template := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
}
`, providerName, domainResource, domainResouceName, domainName)

	emptyProviderTemplate := fmt.Sprintf(`
provider "%s" {}
`, providerName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// On creation
			{
				Config: template,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceReference, "name", domainName),
					resource.TestCheckResourceAttr(domainResourceReference, "unicode_name", domainName),
				),
			},
			// on deletion
			// should happen only from state
			// TODO test it correctly
			{
				Config: emptyProviderTemplate,
			},
		},
	})

}
