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
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

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
					resource.TestCheckResourceAttr(domainResourceFullName, "name", testAccDomainValue()),
					resource.TestCheckResourceAttr(domainResourceFullName, "unicode_name", testAccDomainValue()),
					resource.TestCheckResourceAttr(domainResourceFullName, "is_premium", "false"),
					resource.TestCheckResourceAttrSet(domainResourceFullName, "registration_date"),
					resource.TestCheckResourceAttrSet(domainResourceFullName, "expiration_date"),
					resource.TestCheckResourceAttrSet(domainResourceFullName, "lifecycle_status"),
					resource.TestCheckResourceAttrSet(domainResourceFullName, "verification_status"),
					expectListCountAtLeast(domainResourceFullName, "epp_statuses.#", 0),
					resource.TestCheckResourceAttr(domainResourceFullName, "suspensions.#", "0"),
					//contact checks
					resource.TestCheckResourceAttrSet(domainResourceFullName, "contacts.admin"),
					expectNonEmptyAttr(domainResourceFullName, "contacts.registrant"),
					expectListCountAtLeast(domainResourceFullName, "contacts.attributes.#", 0),
					//privacy protection settings are adopted
					resource.TestCheckResourceAttrSet(domainResourceFullName, "privacy_protection.contact_form"),
					resource.TestCheckResourceAttrSet(domainResourceFullName, "privacy_protection.level"),
					//nameservers
					expectListCountAtLeast(domainResourceFullName, "nameservers.hosts.#", 1),
					expectNonEmptyAttr(domainResourceFullName, "nameservers.provider"),
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
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	configAutoRenewFalse := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
	
	auto_renew = false
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

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

func TestAccDomain_nameservers(t *testing.T) {
	nsProviderBasicConfig := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
	
	nameservers = {
		provider = "basic"
	}
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	nsProviderCustomConfig := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
	
	nameservers = {
		provider = "custom"
		hosts = [
			"ns-669.awsdns-19.net",
			"ns-1578.awsdns-05.co.uk",
			"ns-401.awsdns-50.com",
			"ns-1063.awsdns-04.org",
		]
	}
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1
			// adopt on creation
			{
				Config: emptyDomainResourceConfiguration,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(domainResourceFullName, "nameservers.hosts.0"),
					resource.TestCheckResourceAttrSet(domainResourceFullName, "nameservers.provider"),
				),
			},
			// Step 2
			// verify no changes, changes in code only
			{
				Config:             nsProviderBasicConfig,
				ExpectNonEmptyPlan: false,
			},
			// Step 3
			// update nameservers to custom
			{
				Config: nsProviderCustomConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemAttr(domainResourceFullName, "nameservers.hosts.*", "ns-1063.awsdns-04.org"),
					resource.TestCheckTypeSetElemAttr(domainResourceFullName, "nameservers.hosts.*", "ns-1578.awsdns-05.co.uk"),
					resource.TestCheckTypeSetElemAttr(domainResourceFullName, "nameservers.hosts.*", "ns-401.awsdns-50.com"),
					resource.TestCheckTypeSetElemAttr(domainResourceFullName, "nameservers.hosts.*", "ns-669.awsdns-19.net"),
					resource.TestCheckResourceAttr(domainResourceFullName, "nameservers.provider", "custom"),
				),
			},

			// Step 4
			// reset to basic back
			{
				Config: nsProviderBasicConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "nameservers.provider", "basic"),
					resource.TestCheckTypeSetElemAttr(domainResourceFullName, "nameservers.hosts.*", "launch1.spaceship.net"),
				),
			},
		},
	})
}

func TestAccDomain_nameserversValidationErrors(t *testing.T) {
	nsProviderBasicWithHosts := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
	
	nameservers = {
		provider = "basic"
		hosts = ["ns1.example.com", "ns2.example.com"]
	}
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	nsProviderCustomWithNoHosts := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"

	nameservers = {
		provider = "custom"
	}
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	nsProviderCustomWithDefaultHosts := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"

	nameservers = {
		provider = "custom"
		hosts = ["launch1.spaceship.net", "launch2.spaceship.net"]
	}
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test errors on wrong configuration
			{
				Config:      nsProviderCustomWithNoHosts,
				ExpectError: regexp.MustCompile("The 'hosts' field is required when provider is 'custom'."),
			},
			{
				Config:      nsProviderBasicWithHosts,
				ExpectError: regexp.MustCompile("The 'hosts' field must be omitted when provider is 'basic'."),
			},
			{
				Config:      nsProviderCustomWithDefaultHosts,
				ExpectError: regexp.MustCompile("The default Spaceship nameservers can only be used with provider \"basic\"."),
			},
		},
	})
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
				ImportStateId:                        testAccDomainValue(),
				ImportStateVerifyIdentifierAttribute: "domain",
			},
		},
	})
}
