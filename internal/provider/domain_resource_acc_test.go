package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	providerName            = "spaceship"
	domainResourceRef       = providerName + "_domain"
	domainResouceName       = "this"
	domainResourceReference = domainResourceRef + "." + domainResouceName
)

func TestAccDomain_basic(t *testing.T) {
	domainName := testAccDomainValue()

	template := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
}
`, providerName, domainResourceRef, domainResouceName, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// creation and deletion
			{
				Config: template,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceReference, "name", domainName),
					resource.TestCheckResourceAttr(domainResourceReference, "unicode_name", domainName),
				),
			},
		},
	})

}
