package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// func getTestClient() *Client {
// 	return (
// 		os.Getenv("SPACESHIP_API_KEY"),
// 		os.Getenv("SPACESHIP_API_SECRET"),
// 	)
// }

func TestAccDomain_autorenewal(t *testing.T) {
	// t.Cleanup(func() {
	// 	client := Client.UpdateAutoRenew(context.Background(), "dmytrovovk.com", false)
	// })

	//t.Setenv("TF_LOG", "INFO")

	createTemplate := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

}
`
	diffTemplate := `
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = "dmytrovovk.com"

	auto_renew = false
}
`

	autoRenewUpdateTemplate := `
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
				Config: createTemplate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("spaceship_domain.this", "auto_renew"),
					resource.TestCheckResourceAttr("spaceship_domain.this", "name", "dmytrovovk.com"),
				),
			},
			//stage 2
			// verify no changes
			{
				Config: diffTemplate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("spaceship_domain.this", "auto_renew"),
					//resource.TestCheckResourceAttr("spaceship_domain.this", "auto_renew", "false"),
				),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			//stage 3
			// apply changes to the value
			{
				Config: autoRenewUpdateTemplate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "auto_renew", "true"),
				),
			},
			//stage 4
			// reset to default
			{
				Config: diffTemplate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.this", "auto_renew", "false"),
				),
			},
		},
	})
}

// func TestAccDomain_autorenewal_has_changes(t *testing.T) {
// 	template := `
// provider "spaceship" {}

// resource "spaceship_domain" "this" {
// 	domain = "dmytrovovk.com"

// 	auto_renew = true
// }
// `

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:                 func() { testAccPreCheck(t) },
// 		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: template,
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttr("spaceship_domain.this", "auto_renew", "true"),
// 				),
// 				PlanOnly: true,
// 			},
// 		},
// 	})
// }

// func TestAccDomain_basic(t *testing.T) {
// 	template := `
// provider "spaceship" {}

// resource "spaceship_domain" "this" {
// 	domain = "dmytrovovk.com"
// }
// `

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:                 func() { testAccPreCheck(t) },
// 		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: template,
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttr("spaceship_domain.this", "name", "dmytrovovk.com"),
// 				),
// 			},
// 		},
// 	})

// }
