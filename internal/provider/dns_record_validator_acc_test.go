package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// These tests prove that the per-type record validators in
// recordTypeObjectValidators() — originally written for the multi-record
// resource's nested block — also fire on the singular dns_record resource,
// via the singularRecordValidator adapter wired up in ConfigValidators().
//
// Each case picks a different failure mode to exercise the breadth of the
// adapter, not the depth of any single validator:
//   - MX semantic check on exchange value (calls into clientrecords.MX.ValidateExchange)
//   - MX cross-field required-field check (preference)
//   - CAA cross-field required-field check (tag) on a different record type
//
// All cases use bad config that never reaches apply — the failure is
// reported at validate/plan time, with a path that points at the singular
// resource's flat attribute (e.g. `exchange`, not `records[0].exchange`).

func TestAccDNSRecord_mxValidatesExchangeAtPlanTime(t *testing.T) {
	testAccPreCheck(t)
	domain := testAccDomainValue()

	config := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_dns_record" "test" {
  domain     = %q
  type       = "MX"
  name       = "tfacc-validate-mx-exchange"
  exchange   = "@"
  preference = 10
}
`, domain)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("Invalid Exchange Value"),
			},
		},
	})
}

func TestAccDNSRecord_mxMissingPreferenceAtPlanTime(t *testing.T) {
	testAccPreCheck(t)
	domain := testAccDomainValue()

	config := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_dns_record" "test" {
  domain   = %q
  type     = "MX"
  name     = "tfacc-validate-mx-pref"
  exchange = "mail.example.com"
}
`, domain)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("'preference' field is required for MX records"),
			},
		},
	})
}

func TestAccDNSRecord_caaMissingTagAtPlanTime(t *testing.T) {
	testAccPreCheck(t)
	domain := testAccDomainValue()

	config := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_dns_record" "test" {
  domain = %q
  type   = "CAA"
  name   = "tfacc-validate-caa-tag"
  flag   = 0
  value  = "letsencrypt.org"
}
`, domain)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("'tag' field is required for CAA records"),
			},
		},
	})
}
