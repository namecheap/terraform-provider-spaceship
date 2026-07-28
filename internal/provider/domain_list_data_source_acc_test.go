package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const domainListDataSourceName = "data.spaceship_domain_list.this"

type attrExpectation struct {
	Attribute string
	Value     string
}

func TestAccDatasourceDomainList_basic(t *testing.T) {
	domainName := testAccDomainValue()
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
					withListedDomain(domainName, func(index int) resource.TestCheckFunc {
						return resource.ComposeAggregateTestCheckFunc(
							domainBasicsChecks(index, domainName),
							privacyProtectionChecks(index),
							nameserverChecks(index),
							contactChecks(index),
						)
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

// domainListSummaryChecks asserts the list is non-empty and internally
// consistent, without pinning the account to a specific domain count.
func domainListSummaryChecks() resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		expectListCountAtLeast(domainListDataSourceName, "total", 1),
		resource.TestCheckResourceAttrPair(domainListDataSourceName, "total", domainListDataSourceName, "items.#"),
	)
}

// withListedDomain locates domain in the data source's items list and runs the
// checks built for its index, so per-domain assertions hold on any account
// regardless of how many domains it contains or how they are ordered.
func withListedDomain(domain string, buildChecks func(index int) resource.TestCheckFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[domainListDataSourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", domainListDataSourceName)
		}
		count, err := strconv.Atoi(rs.Primary.Attributes["items.#"])
		if err != nil {
			return fmt.Errorf("expected items.# to be an integer, got %q: %w", rs.Primary.Attributes["items.#"], err)
		}
		for i := range count {
			if rs.Primary.Attributes[domainAttr(i, "name")] == domain {
				return buildChecks(i)(s)
			}
		}
		return fmt.Errorf("domain %q not found among the %d domains in the account", domain, count)
	}
}

func domainBasicsChecks(index int, domain string) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		expectAttrValues(domainListDataSourceName, []attrExpectation{
			{Attribute: domainAttr(index, "name"), Value: domain},
			{Attribute: domainAttr(index, "unicode_name"), Value: domain},
			{Attribute: domainAttr(index, "is_premium"), Value: "false"},
		}),
		expectNonEmptyAttrs(domainListDataSourceName, []string{
			domainAttr(index, "registration_date"),
			domainAttr(index, "expiration_date"),
			domainAttr(index, "lifecycle_status"),
			domainAttr(index, "verification_status"),
			domainAttr(index, "auto_renew"),
		}),
		expectListCountAtLeast(domainListDataSourceName, domainAttr(index, "epp_statuses.#"), 0),
		expectListCount(domainListDataSourceName, domainAttr(index, "suspensions.#"), 0),
	)
}

func privacyProtectionChecks(index int) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttrSet(domainListDataSourceName, nestedAttr(index, "privacy_protection", "contact_form")),
		resource.TestCheckResourceAttrSet(domainListDataSourceName, nestedAttr(index, "privacy_protection", "level")),
	)
}

func nameserverChecks(index int) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		expectListCountAtLeast(domainListDataSourceName, nestedAttr(index, "nameservers", "hosts.#"), 1),
		expectNonEmptyAttr(domainListDataSourceName, nestedAttr(index, "nameservers", "hosts.0")),
		expectNonEmptyAttr(domainListDataSourceName, nestedAttr(index, "nameservers", "provider")),
	)
}

func contactChecks(index int) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		expectNonEmptyAttr(domainListDataSourceName, nestedAttr(index, "contacts", "registrant")),
		expectListCountAtLeast(domainListDataSourceName, nestedAttr(index, "contacts", "attributes.#"), 0),
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
