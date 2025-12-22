package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	domainResourceType         = "spaceship_domain"
	domainResourceInstanceName = "this"
	domainResourceName         = domainResourceType + "." + domainResourceInstanceName
)

func TestAccDomain_autorenewal(t *testing.T) {
	domainName := testAccDomainName(t)
	// TODO
	// consider creating separate cleanup funtion
	// to reset state to default
	// t.Cleanup(func() {
	// 	client := Client.UpdateAutoRenew(context.Background(), domainName, false)
	// })

	//t.Setenv("TF_LOG", "INFO")

	creationConfig := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"
}
`, domainName)

	autoRenewSetFalse := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"

	auto_renew = false
}
`, domainName)

	autoRenewSetTrue := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"

	auto_renew = true
}
`, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// stage 1
			// adopt on creation
			{
				Config: creationConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(domainResourceName, "auto_renew"),
					resource.TestCheckResourceAttr(domainResourceName, "name", domainName),
				),
			},
			// stage 2
			// verify no changes
			{
				Config: autoRenewSetFalse,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceName, "auto_renew", "false"),
				),
			},
			// stage 3
			// apply changes to the value
			{
				Config: autoRenewSetTrue,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceName, "auto_renew", "true"),
				),
			},
			// stage 4
			// reset to default
			{
				Config: autoRenewSetFalse,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceName, "auto_renew", "false"),
				),
			},
		},
	})
}

func TestAccDomain_basic(t *testing.T) {
	domainName := testAccDomainName(t)

	//t.Setenv("TF_LOG", "DEBUG")

	template := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"
}
`, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: template,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceName, "name", domainName),
					resource.TestCheckResourceAttr(domainResourceName, "unicode_name", domainName),
					resource.TestCheckResourceAttr(domainResourceName, "is_premium", "false"),
					resource.TestCheckResourceAttrSet(domainResourceName, "registration_date"),
					resource.TestCheckResourceAttrSet(domainResourceName, "expiration_date"),
					resource.TestCheckResourceAttrSet(domainResourceName, "lifecycle_status"),
					resource.TestCheckResourceAttrSet(domainResourceName, "verification_status"),
					expectListCountAtLeast(domainResourceName, "epp_statuses.#", 0),
					resource.TestCheckResourceAttr(domainResourceName, "suspensions.#", "0"),
					//contact checks
					resource.TestCheckResourceAttrSet(domainResourceName, "contacts.admin"),
					expectNonEmptyAttr(domainResourceName, "contacts.registrant"),
					expectListCountAtLeast(domainResourceName, "contacts.attributes.#", 0),
					//privacy protection settings are adopted
					resource.TestCheckResourceAttrSet(domainResourceName, "privacy_protection.contact_form"),
					resource.TestCheckResourceAttrSet(domainResourceName, "privacy_protection.level"),
					//nameservers
					expectListCountAtLeast(domainResourceName, "nameservers.hosts.#", 1),
					expectNonEmptyAttr(domainResourceName, "nameservers.provider"),
				),
			},
		},
	})
}

/*
func TestAccDomain_privacyProtection(t *testing.T) {
	domainName := testAccDomainName(t)
	t.Setenv("TF_LOG", "DEBUG")

	creationConfig := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"
}
`, domainName)

	ppDefaultConfig := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"

	privacy_protection = {
		contact_form = false
		level = "high"
	}
}
`, domainName)

	// 	ppContactFormFalseConfig := fmt.Sprintf(`
	// provider "spaceship" {}

	// resource "spaceship_domain" "this" {
	// 	domain = "%s"

	// 	privacy_protection = {
	// 		contact_form = false
	// 		level = "high"
	// 	}
	// }
	// `, domainName)
	ppLevelPublicConfig := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"

	privacy_protection =  {
		contact_form = true
		level = "public"
	}
}
`, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// step 1
			// adopt on creation
			{
				Config: creationConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(domainResourceName, "privacy_protection.contact_form"),
					resource.TestCheckResourceAttrSet(domainResourceName, "privacy_protection.level"),
				),
			},
			// Step 2
			// verify no changes
			// default config
			{
				Config: ppDefaultConfig,
				//ExpectNonEmptyPlan: false,
			},
			// Step 3
			// Update contact form to false
			// {
			// 	Config: ppContactFormFalseConfig,
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestCheckResourceAttr(domainResourceName, "privacy_protection.contact_form", "false"),
			// 	),
			// },
			// Step 4
			// update level to public
			{
				Config: ppLevelPublicConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceName, "privacy_protection.level", "public"),
				),
			},
			// {
			// 	Config:             ppLevelPublicConfig,
			// 	ExpectNonEmptyPlan: false,
			// },
			// Step 5
			// reset to default
			{
				Config: ppDefaultConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceName, "privacy_protection.contact_form", "true"),
					resource.TestCheckResourceAttr(domainResourceName, "privacy_protection.level", "high"),
				),
			},
		},
	})
}
*/

func TestAccDomain_nameservers(t *testing.T) {
	domainName := testAccDomainName(t)

	//t.Setenv("TF_LOG", "DEBUG")

	creationConfig := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"
}
`, domainName)

	customDefaultNsConfig := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"

	nameservers = {
		provider = "basic"
	}
}
`, domainName)

	customHostsNsConfig := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
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
`, domainName)

	basicNsConfig := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"

	nameservers = {
		provider = "basic"
	}
}
`, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// step 1
			// adopt on creation
			{
				Config: creationConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					expectListCountAtLeast(domainResourceName, "nameservers.hosts.#", 1),
					expectNonEmptyAttr(domainResourceName, "nameservers.provider"),
				),
			},
			// Step 2
			// verify no changes
			// default config
			{
				Config:             customDefaultNsConfig,
				ExpectNonEmptyPlan: false,
			},
			// Step 3
			// Update ns to basic
			{
				Config: basicNsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceName, "nameservers.provider", "basic"),
				),
			},
			// Step 4
			// update hosts only
			{
				Config: customHostsNsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemAttr(domainResourceName, "nameservers.hosts.*", "ns-1063.awsdns-04.org"),
					resource.TestCheckTypeSetElemAttr(domainResourceName, "nameservers.hosts.*", "ns-1578.awsdns-05.co.uk"),
					resource.TestCheckTypeSetElemAttr(domainResourceName, "nameservers.hosts.*", "ns-401.awsdns-50.com"),
					resource.TestCheckTypeSetElemAttr(domainResourceName, "nameservers.hosts.*", "ns-669.awsdns-19.net"),
					resource.TestCheckResourceAttr(domainResourceName, "nameservers.provider", "custom"),
				),
			},
			// Step 5
			// reset to default
			{
				Config: customDefaultNsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceName, "nameservers.provider", "basic"),
					resource.TestCheckTypeSetElemAttr(domainResourceName, "nameservers.hosts.*", "launch1.spaceship.net"),
				),
			},
		},
	})
}

func TestAccDomain_nameserversValidationError(t *testing.T) {
	domainName := testAccDomainName(t)

	nsProviderBasicWithHosts := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"

	nameservers = {
		provider = "basic"
		hosts = ["ns1.example.com", "ns2.example.com"]
	}
}
`, domainName)

	nsProviderCustomWithNoHosts := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"

	nameservers = {
		provider = "custom"
	}
}
`, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      nsProviderCustomWithNoHosts,
				ExpectError: regexp.MustCompile("The 'hosts' field is required when provider is 'custom'."),
			},
			{
				Config:      nsProviderBasicWithHosts,
				ExpectError: regexp.MustCompile("The 'hosts' field must be omitted when provider is 'basic'."),
			},
		},
	})

}

func TestAccDomain_resourceImport(t *testing.T) {
	domainName := testAccDomainName(t)
	//t.Setenv("TF_LOG", "DEBUG")

	creationConfig := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"
}
`, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			//step 1 creation
			{
				Config: creationConfig,
			},
			// import
			{
				ResourceName:                         domainResourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        domainName,
				ImportStateVerifyIdentifierAttribute: "domain",
			},
		},
	})
}

/*
maybe sometime later when api would support providing
transfer lock status for a domain in domain list,
those resources would make sense
or getting transfer lock at least

func TestAccDomain_transferLock(t *testing.T) {
	domainName := testAccDomainName(t)
	creationConfig := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"
}
`, domainName)

	transferLockFalse := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"

	transfer_lock = false
}
`, domainName)

	transferLockTrue := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "%s"

	transfer_lock = true
}
`, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// stage 1
			// adopt on creation
			{
				Config: creationConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(domainResourceName, "transfer_lock"),
					resource.TestCheckResourceAttr(domainResourceName, "name", domainName),
				),
			},
			// stage 2
			// verify no changes
			{
				Config: transferLockFalse,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceName, "transfer_lock", "false"),
				),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// stage 3
			// apply changes to the value
			{
				Config: transferLockTrue,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceName, "transfer_lock", "true"),
				),
			},
			// stage 4
			// reset to default
			{
				Config: transferLockFalse,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceName, "transfer_lock", "false"),
				),
			},
		},
	})
}
*/
