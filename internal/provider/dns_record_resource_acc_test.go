package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDNSRecord_create(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}

	resourceName := "spaceship_dns_record.test"

	config := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_dns_record" "test" {
	domain  = %q
	type    = "A"
	name    = "@"
	address = "1.1.1.1"
	
}
	`, domain)

	//TODO how chose checks actually work?
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "A"),
					resource.TestCheckResourceAttr(resourceName, "name", "@"),
					resource.TestCheckResourceAttr(resourceName, "address", "1.1.1.1"),
					resource.TestCheckResourceAttr(resourceName, "ttl", "3600"),
				),
				// ConfigPlanChecks: resource.ConfigPlanChecks{
				// 	pre
				// },
				// WHAT do I want to test here?
				// what record is being created and saved to state
			},
		},
	})

}
