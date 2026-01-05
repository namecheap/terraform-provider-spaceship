package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

const (
	providerName           = "spaceship"
	domainResourceRef      = providerName + "_domain"
	domainResourceName     = "this"
	domainResourceFullName = domainResourceRef + "." + domainResourceName
)

var emptyDomainResourceConfiguration = fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
}
`, providerName, domainResourceRef, domainResourceName, domainName)

func TestAccDomain_basic(t *testing.T) {

	templateDomainNameChanged := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
}
`, providerName, domainResourceRef, domainResourceName, "spaceship.com")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// creation and deletion
			{
				Config: emptyDomainResourceConfiguration,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "name", domainName),
					resource.TestCheckResourceAttr(domainResourceFullName, "unicode_name", domainName),
				),
			},
			// test for recreation on domain name change
			{
				Config: templateDomainNameChanged,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(domainResourceFullName, plancheck.ResourceActionReplace),
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

	configAutoRenewTrue := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
	
	auto_renew = true
}
`, providerName, domainResourceRef, domainResourceName, domainName)

	configAutoRenewFalse := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
	
	auto_renew = false
}
`, providerName, domainResourceRef, domainResourceName, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// resource contains current autorenew value
			{
				Config: emptyDomainResourceConfiguration,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(domainResourceFullName, "auto_renew"),
				),
			},
			// resource has auto_renew value true
			{
				Config: configAutoRenewTrue,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "auto_renew", "true"),
				),
			},
			// auto_renew value false
			{
				Config: configAutoRenewFalse,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "auto_renew", "false"),
				),
			},
		}})
}

func TestAccDomain_resourceImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			//step 1 creation
			{
				Config: emptyDomainResourceConfiguration,
			},
			// import
			{
				ResourceName:                         domainResourceFullName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        domainName,
				ImportStateVerifyIdentifierAttribute: "domain",
			},
		},
	})
}
