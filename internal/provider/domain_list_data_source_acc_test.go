package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	domainListDataSourceName = "data.spaceship_domain_list.this"
	firstDomainIndex         = 0
)

type attrExpectation struct {
	Attribute string
	Value     string
}

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
					domainListSummaryChecks(),
					domainBasicsChecks(),
					privacyProtectionChecks(),
					nameserverChecks(),
					contactChecks(),
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

func domainListSummaryChecks() resource.TestCheckFunc {
	return expectAttrValues(domainListDataSourceName, []attrExpectation{
		{Attribute: "total", Value: "1"},
		{Attribute: "items.#", Value: "1"},
	})
}

func domainBasicsChecks() resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		expectAttrValues(domainListDataSourceName, []attrExpectation{
			{Attribute: domainAttr(firstDomainIndex, "name"), Value: "dmytrovovk.com"},
			{Attribute: domainAttr(firstDomainIndex, "unicode_name"), Value: "dmytrovovk.com"},
			{Attribute: domainAttr(firstDomainIndex, "is_premium"), Value: "false"},
		}),
		expectNonEmptyAttrs(domainListDataSourceName, []string{
			domainAttr(firstDomainIndex, "registration_date"),
			domainAttr(firstDomainIndex, "expiration_date"),
			domainAttr(firstDomainIndex, "lifecycle_status"),
			domainAttr(firstDomainIndex, "verification_status"),
			domainAttr(firstDomainIndex, "auto_renew"),
		}),
		expectListCountAtLeast(domainListDataSourceName, domainAttr(firstDomainIndex, "epp_statuses.#"), 0),
		expectListCount(domainListDataSourceName, domainAttr(firstDomainIndex, "suspensions.#"), 0),
	)
}

func privacyProtectionChecks() resource.TestCheckFunc {
	return expectAttrValues(domainListDataSourceName, []attrExpectation{
		{Attribute: nestedAttr(firstDomainIndex, "privacy_protection", "contact_form"), Value: "true"},
		{Attribute: nestedAttr(firstDomainIndex, "privacy_protection", "level"), Value: "high"},
	})
}

func nameserverChecks() resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		expectListCountAtLeast(domainListDataSourceName, nestedAttr(firstDomainIndex, "nameservers", "hosts.#"), 1),
		expectNonEmptyAttr(domainListDataSourceName, nestedAttr(firstDomainIndex, "nameservers", "provider")),
	)
}

func contactChecks() resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		expectNonEmptyAttr(domainListDataSourceName, nestedAttr(firstDomainIndex, "contacts", "registrant")),
		expectListCountAtLeast(domainListDataSourceName, nestedAttr(firstDomainIndex, "contacts", "attributes.#"), 0),
	)
}

func expectAttrValues(resourceName string, expectations []attrExpectation) resource.TestCheckFunc {
	checks := make([]resource.TestCheckFunc, len(expectations))
	for i, exp := range expectations {
		checks[i] = resource.TestCheckResourceAttr(resourceName, exp.Attribute, exp.Value)
	}
	return resource.ComposeTestCheckFunc(checks...)
}

func expectNonEmptyAttr(resourceName, attribute string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(resourceName, attribute, func(value string) error {
		if value == "" {
			return fmt.Errorf("expected %s to be a non-empty string", attribute)
		}
		return nil
	})
}

func expectNonEmptyAttrs(resourceName string, attributes []string) resource.TestCheckFunc {
	checks := make([]resource.TestCheckFunc, len(attributes))
	for i, attr := range attributes {
		checks[i] = expectNonEmptyAttr(resourceName, attr)
	}
	return resource.ComposeTestCheckFunc(checks...)
}

func expectListCount(resourceName, attribute string, expected int) resource.TestCheckFunc {
	return resource.TestCheckResourceAttr(resourceName, attribute, strconv.Itoa(expected))
}

func expectListCountAtLeast(resourceName, attribute string, min int) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(resourceName, attribute, func(value string) error {
		count, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("expected %s to be an integer, got %q: %w", attribute, value, err)
		}
		if count < min {
			return fmt.Errorf("expected %s to be >= %d, got %d", attribute, min, count)
		}
		return nil
	})
}

func domainAttr(index int, attribute string) string {
	return fmt.Sprintf("items.%d.%s", index, attribute)
}

func nestedAttr(index int, block, attribute string) string {
	return fmt.Sprintf("items.%d.%s.%s", index, block, attribute)
}
