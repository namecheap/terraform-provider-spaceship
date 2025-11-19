package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatasourceDomainList_basic(t *testing.T) {
	cfg := `
provider "spaceship" {}

data "spaceship_domain_list" "this"{}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.spaceship_domain_list.this", "total", "1"),
					resource.TestCheckResourceAttr("data.spaceship_domain_list.this", "items.#", "1"),
					resource.TestCheckResourceAttr("data.spaceship_domain_list.this", "items.0.name", "dmytrovovk.com"),
					resource.TestCheckResourceAttr("data.spaceship_domain_list.this", "items.0.unicode_name", "dmytrovovk.com"),
					resource.TestCheckResourceAttr("data.spaceship_domain_list.this", "items.0.is_premium", "false"),
					resource.TestCheckResourceAttr("data.spaceship_domain_list.this", "items.0.auto_renew", "false"),
					resource.TestCheckResourceAttrWith(
						"data.spaceship_domain_list.this",
						"items.0.registration_date",
						func(value string) error {
							if value == "" {
								return fmt.Errorf("expected registration_date to be a non-empty string")
							}
							return nil
						},
					),
					resource.TestCheckResourceAttrWith(
						"data.spaceship_domain_list.this",
						"items.0.expiration_date",
						func(value string) error {
							if value == "" {
								return fmt.Errorf("expected expiration_date to be a non-empty string")
							}
							return nil
						},
					),
					resource.TestCheckResourceAttrWith(
						"data.spaceship_domain_list.this",
						"items.0.lifecycle_status",
						func(value string) error {
							if value == "" {
								return fmt.Errorf("expected lifecycle_status to be a non-empty string")
							}
							return nil
						},
					),
					resource.TestCheckResourceAttrWith(
						"data.spaceship_domain_list.this",
						"items.0.verification_status",
						func(value string) error {
							if value == "" {
								return fmt.Errorf("expected verification_status to be a non-empty string")
							}
							return nil
						},
					),
					resource.TestCheckResourceAttr("data.spaceship_domain_list.this", "items.0.epp_statuses.#", "1"),
					resource.TestCheckResourceAttrWith(
						"data.spaceship_domain_list.this",
						"items.0.epp_statuses.0",
						func(value string) error {
							if value == "" {
								return fmt.Errorf("expected epp_statuse item to be a non-empty string")
							}
							return nil
						},
					),
					// empty list for this domain
					resource.TestCheckResourceAttr("data.spaceship_domain_list.this", "items.0.suspensions.#", "0"),
					// TODO will fail when domain is not suspended
					// resource.TestCheckResourceAttrWith(
					// 	"data.spaceship_domain_list.this",
					// 	"items.0.suspensions.0.reason_code",
					// 	func(value string) error {
					// 		if value == "" {
					// 			return fmt.Errorf("expected suspensions reason_code item to be a non-empty string")
					// 		}
					// 		return nil
					// 	},
					// ),

					//Privacy protection checks
					resource.TestCheckResourceAttr("data.spaceship_domain_list.this", "items.0.privacy_protection.contact_form", "true"),
					resource.TestCheckResourceAttr("data.spaceship_domain_list.this", "items.0.privacy_protection.level", "high"),

					//Nameservers checks
					resource.TestCheckResourceAttrWith("data.spaceship_domain_list.this", "items.0.nameservers.hosts.0", func(value string) error {
						if value == "" {
							return fmt.Errorf("expected hosts item to be a non-empty string")
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("data.spaceship_domain_list.this", "items.0.nameservers.provider", func(value string) error {
						if value == "" {
							return fmt.Errorf("expected nameservers provider to be non-empty string")
						}
						return nil
					}),

					//Contact checks
					resource.TestCheckResourceAttrWith("data.spaceship_domain_list.this", "items.0.contacts.registrant", func(value string) error {
						if value == "" {
							return fmt.Errorf("expected registrant to be a non-empty string")
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccDomainListDataSource_Unconfigured(t *testing.T) {
	cfg := `
data "spaceship_domain_list" "this" {}
`

	t.Run("missing_api_key", func(t *testing.T) {
		t.Setenv("SPACESHIP_API_KEY", "")
		t.Setenv("SPACESHIP_API_SECRET", "some-secret")

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      cfg,
					ExpectError: regexp.MustCompile("Missing Spaceship API key"),
				},
			},
		})
	})

	t.Run("missing_api_secret", func(t *testing.T) {
		t.Setenv("SPACESHIP_API_KEY", "some-key")
		t.Setenv("SPACESHIP_API_SECRET", "")

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      cfg,
					ExpectError: regexp.MustCompile("Missing Spaceship API secret"),
				},
			},
		})
	})

	t.Run("missing_both", func(t *testing.T) {
		t.Setenv("SPACESHIP_API_KEY", "")
		t.Setenv("SPACESHIP_API_SECRET", "")

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      cfg,
					ExpectError: regexp.MustCompile("Missing Spaceship API (key|secret)"),
				},
			},
		})
	})
}
