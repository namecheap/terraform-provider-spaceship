package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDomain_autorenewal(t *testing.T) {
	// TODO
	// consider creating separate cleanup funtion
	// to reset state to default
	// t.Cleanup(func() {
	// 	client := Client.UpdateAutoRenew(context.Background(), "dmytrovovk.com", false)
	// })

	//t.Setenv("TF_LOG", "INFO")

	creationConfig := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"
}
`

	autoRenewSetFalse := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

	auto_renew = false
}
`

	autoRenewSetTrue := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

	auto_renew = true
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// stage 1
			// adopt on creation
			{
				Config: creationConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("spaceship_domain.this", "auto_renew"),
					resource.TestCheckResourceAttr("spaceship_domain.this", "name", "dmytrovovk.com"),
				),
			},
			// stage 2
			// verify no changes
			{
				Config: autoRenewSetFalse,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "auto_renew", "false"),
				),
			},
			// stage 3
			// apply changes to the value
			{
				Config: autoRenewSetTrue,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "auto_renew", "true"),
				),
			},
			// stage 4
			// reset to default
			{
				Config: autoRenewSetFalse,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "auto_renew", "false"),
				),
			},
		},
	})
}

func TestAccDomain_basic(t *testing.T) {

	//t.Setenv("TF_LOG", "DEBUG")

	template := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: template,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "name", "dmytrovovk.com"),
					resource.TestCheckResourceAttr("spaceship_domain.this", "unicode_name", "dmytrovovk.com"),
					resource.TestCheckResourceAttr("spaceship_domain.this", "is_premium", "false"),
					resource.TestCheckResourceAttrSet("spaceship_domain.this", "registration_date"),
					resource.TestCheckResourceAttrSet("spaceship_domain.this", "expiration_date"),
					resource.TestCheckResourceAttrSet("spaceship_domain.this", "lifecycle_status"),
					resource.TestCheckResourceAttrSet("spaceship_domain.this", "verification_status"),
					expectListCountAtLeast("spaceship_domain.this", "epp_statuses.#", 0),
					resource.TestCheckResourceAttr("spaceship_domain.this", "suspensions.#", "0"),
					//contact checks
					resource.TestCheckResourceAttrSet("spaceship_domain.this", "contacts.admin"),
					expectNonEmptyAttr("spaceship_domain.this", "contacts.registrant"),
					expectListCountAtLeast("spaceship_domain.this", "contacts.attributes.#", 0),
					//privacy protection settings are adopted
					resource.TestCheckResourceAttrSet("spaceship_domain.this", "privacy_protection.contact_form"),
					resource.TestCheckResourceAttrSet("spaceship_domain.this", "privacy_protection.level"),
					//nameservers
					expectListCountAtLeast("spaceship_domain.this", "nameservers.hosts.#", 1),
					expectNonEmptyAttr("spaceship_domain.this", "nameservers.provider"),
				),
			},
		},
	})
}

func TestAccDomain_privacyProtection(t *testing.T) {
	t.Setenv("TF_LOG", "DEBUG")

	creationConfig := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"
}
`
	ppDefaultConfig := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

	privacy_protection = {
		contact_form = true
		level = "high"
	}
}
`

	ppContactFormFalseConfig := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

	privacy_protection = {
		contact_form = false
		level = "high"
	}
}
`
	ppLevelPublicConfig := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

	privacy_protection =  {
		contact_form = false
		level = "public"
	}
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// step 1
			// adopt on creation
			{
				Config: creationConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					//resource.TestCheckResourceAttr("spaceship_domain.this", "privacy_protection.contact_form", "true"),
					resource.TestCheckResourceAttrSet("spaceship_domain.this", "privacy_protection.contact_form"),
					//resource.TestCheckResourceAttr("spaceship_domain.this", "privacy_protection.level", "high"),
					resource.TestCheckResourceAttrSet("spaceship_domain.this", "privacy_protection.level"),
				),
			},
			// Step 2
			// verify no changes
			// default config
			{
				Config:             ppDefaultConfig,
				ExpectNonEmptyPlan: false,
			},
			// Step 3
			// Update contact form to false
			{
				Config: ppContactFormFalseConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "privacy_protection.contact_form", "false"),
				),
			},
			// Step 4
			// update level to public
			{
				Config: ppLevelPublicConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "privacy_protection.level", "public"),
				),
			},
			// Step 5
			// reset to default
			{
				Config: ppDefaultConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "privacy_protection.contact_form", "true"),
					resource.TestCheckResourceAttr("spaceship_domain.this", "privacy_protection.level", "high"),
				),
			},
		},
	})
}

func TestAccDomain_nameservers(t *testing.T) {

	//t.Setenv("TF_LOG", "DEBUG")

	creationConfig := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"
}
`

	customDefaultNsConfig := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

	nameservers = {
		provider = "basic"
		hosts = [
			"launch1.spaceship.net", 
			"launch2.spaceship.net",
		]
	}
}
`

	customHostsNsConfig := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

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
`

	basicNsConfig := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

	nameservers = {
		provider = "basic"
	}
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// step 1
			// adopt on creation
			{
				Config: creationConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					expectListCountAtLeast("spaceship_domain.this", "nameservers.hosts.#", 1),
					expectNonEmptyAttr("spaceship_domain.this", "nameservers.provider"),
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
					resource.TestCheckResourceAttr("spaceship_domain.this", "nameservers.provider", "basic"),
				),
			},
			// Step 4
			// update hosts only
			{
				Config: customHostsNsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemAttr("spaceship_domain.this", "nameservers.hosts.*", "ns-1063.awsdns-04.org"),
					resource.TestCheckTypeSetElemAttr("spaceship_domain.this", "nameservers.hosts.*", "ns-1578.awsdns-05.co.uk"),
					resource.TestCheckTypeSetElemAttr("spaceship_domain.this", "nameservers.hosts.*", "ns-401.awsdns-50.com"),
					resource.TestCheckTypeSetElemAttr("spaceship_domain.this", "nameservers.hosts.*", "ns-669.awsdns-19.net"),
					resource.TestCheckResourceAttr("spaceship_domain.this", "nameservers.provider", "custom"),
				),
			},
			// Step 5
			// reset to default
			{
				Config: customDefaultNsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "nameservers.provider", "basic"),
					resource.TestCheckTypeSetElemAttr("spaceship_domain.this", "nameservers.hosts.*", "launch1.spaceship.net"),
				),
			},
		},
	})
}

func TestAccDomain_nameserversValidationError(t *testing.T) {
	nsProviderBasicWithHosts := `
	provider "spaceship" {}

	resource "spaceship_domain" "this" {
		domain = "dmytrovovk.com"

		nameservers = {
			provider = "basic"
			hosts = ["ns1.example.com", "ns2.example.com"]
		}
	}
	`

	nsProviderCustomWithNoHosts := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

	nameservers = {
		provider = "custom"
	}
}
`

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
	//t.Setenv("TF_LOG", "DEBUG")

	creationConfig := `
	provider "spaceship" {}

	resource "spaceship_domain" "this" {
		domain = "dmytrovovk.com"
	}
	`

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
				ResourceName:                         "spaceship_domain.this",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        "dmytrovovk.com",
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
	creationConfig := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"
}
`

	transferLockFalse := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

	transfer_lock = false
}
`

	transferLockTrue := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

	transfer_lock = true
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// stage 1
			// adopt on creation
			{
				Config: creationConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("spaceship_domain.this", "transfer_lock"),
					resource.TestCheckResourceAttr("spaceship_domain.this", "name", "dmytrovovk.com"),
				),
			},
			// stage 2
			// verify no changes
			{
				Config: transferLockFalse,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "transfer_lock", "false"),
				),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// stage 3
			// apply changes to the value
			{
				Config: transferLockTrue,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "transfer_lock", "true"),
				),
			},
			// stage 4
			// reset to default
			{
				Config: transferLockFalse,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "transfer_lock", "false"),
				),
			},
		},
	})
}
*/
