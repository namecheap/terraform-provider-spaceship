package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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

	templateDomainNameChanged := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
}
`, providerName, domainResourceRef, domainResouceName, "spaceship.com")

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
			// test for recreation on domain name change
			{
				Config: templateDomainNameChanged,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(domainResourceReference, plancheck.ResourceActionReplace),
					},
				},
				// workaround when I have only one domain in account
				// and cant use another one for now
				ExpectError: regexp.MustCompile("spaceship api error"),
			},
		},
	})

}

func TestAccDomain_autoRenewal(t *testing.T) {
	domainName := testAccDomainValue()

	config := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
}
`, providerName, domainResourceRef, domainResouceName, domainName)

	configAutoRenewTrue := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
	
	auto_renew = "true"
}
`, providerName, domainResourceRef, domainResouceName, domainName)

	configAutoRenewFalse := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
	
	auto_renew = "false"
}
`, providerName, domainResourceRef, domainResouceName, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// resource contains current autorenew value
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(domainResourceReference, "auto_renew"),
				),
			},
			// resource has auto_renew value true
			{
				Config: configAutoRenewTrue,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceReference, "auto_renew", "true"),
				),
			},
			// auto_renew value false
			{
				Config: configAutoRenewFalse,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceReference, "auto_renew", "false"),
				),
			},
		}})
}
