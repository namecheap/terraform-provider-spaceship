package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	dataSource               = "spaceship_domain_info"
	dataSourceResourceName   = "this"
	domainInfoDataSourceName = "data." + dataSource + "." + dataSourceResourceName
)

func TestAccDomainInfo_basic(t *testing.T) {
	template := `
provider "spaceship" {}

data "%s" "%s" {
	domain = "%s"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(template, dataSource, dataSourceResourceName, testAccDomainValue()),
				Check: resource.ComposeAggregateTestCheckFunc(
					domainInfoBasicsChecks(),
					domainInfoPrivacyProtectionChecks(),
					domainInfoNameserverChecks(),
					domainInfoContactChecks(),
					domainInfoSuspensionChecks(),
				),
			},
		},
	})

}

func domainInfoBasicsChecks() resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		expectAttrValues(domainInfoDataSourceName, []attrExpectation{
			{Attribute: "name", Value: testAccDomainValue()},
			{Attribute: "unicode_name", Value: testAccDomainValue()},
			{Attribute: "is_premium", Value: "false"},
		}),
		expectNonEmptyAttrs(domainInfoDataSourceName, []string{
			"registration_date",
			"expiration_date",
			"lifecycle_status",
			"verification_status",
			"auto_renew",
		}),
		expectListCountAtLeast(domainInfoDataSourceName, "epp_statuses.#", 0),
	)
}

func domainInfoPrivacyProtectionChecks() resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		expectNonEmptyAttr(domainInfoDataSourceName, "privacy_protection.contact_form"),
		expectNonEmptyAttr(domainInfoDataSourceName, "privacy_protection.level"),
	)
}

func domainInfoNameserverChecks() resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		expectListCountAtLeast(domainInfoDataSourceName, "nameservers.hosts.#", 1),
		expectNonEmptyAttr(domainInfoDataSourceName, "nameservers.hosts.0"),
		expectNonEmptyAttr(domainInfoDataSourceName, "nameservers.provider"),
	)
}

func domainInfoContactChecks() resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		expectNonEmptyAttr(domainInfoDataSourceName, "contacts.registrant"),
		expectListCountAtLeast(domainInfoDataSourceName, "contacts.attributes.#", 0),
	)
}

func domainInfoSuspensionChecks() resource.TestCheckFunc {
	return expectListCount(domainInfoDataSourceName, "suspensions.#", 0)
}

func TestAccDomainInfo_wrongDomainInfo(t *testing.T) {
	cfg := `
provider "spaceship" {}

data "spaceship_domain_info" "this" {
	domain = "a.c"
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      cfg,
				ExpectError: regexp.MustCompile("Invalid Attribute Value Length"),
			},
		},
	})

}
