package provider

import (
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
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
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
				),
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
