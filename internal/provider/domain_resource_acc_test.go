package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/namecheap/go-spaceship-sdk/client"
)

const (
	providerName           = "spaceship"
	domainResourceRef      = providerName + "_domain"
	domainResourceName     = "this"
	domainResourceFullName = domainResourceRef + "." + domainResourceName
)

var emptyDomainResourceConfiguration = fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

func TestAccDomain_basic(t *testing.T) {

	templateDomainNameChanged := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
}
`, providerName, domainResourceRef, domainResourceName, "spaceship.com")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// creation and deletion
			{
				Config: emptyDomainResourceConfiguration,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "name", testAccDomainValue()),
					resource.TestCheckResourceAttr(domainResourceFullName, "unicode_name", testAccDomainValue()),
					resource.TestCheckResourceAttr(domainResourceFullName, "is_premium", "false"),
					resource.TestCheckResourceAttrSet(domainResourceFullName, "registration_date"),
					resource.TestCheckResourceAttrSet(domainResourceFullName, "expiration_date"),
					resource.TestCheckResourceAttrSet(domainResourceFullName, "lifecycle_status"),
					resource.TestCheckResourceAttrSet(domainResourceFullName, "verification_status"),
					expectListCountAtLeast(domainResourceFullName, "epp_statuses.#", 0),
					resource.TestCheckResourceAttr(domainResourceFullName, "suspensions.#", "0"),
					//contact checks
					resource.TestCheckResourceAttrSet(domainResourceFullName, "contacts.admin"),
					expectNonEmptyAttr(domainResourceFullName, "contacts.registrant"),
					expectListCountAtLeast(domainResourceFullName, "contacts.attributes.#", 0),
					//privacy protection settings are adopted
					resource.TestCheckResourceAttrSet(domainResourceFullName, "privacy_protection.contact_form"),
					resource.TestCheckResourceAttrSet(domainResourceFullName, "privacy_protection.level"),
					//nameservers
					expectListCountAtLeast(domainResourceFullName, "nameservers.hosts.#", 1),
					expectNonEmptyAttr(domainResourceFullName, "nameservers.provider"),
				),
			},
			// test for recreation on domain name change
			{
				Config: templateDomainNameChanged,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(domainResourceFullName, plancheck.ResourceActionReplace),
					},
				},
				// workaround when I have only one domain in account
				// and cant use another one for now
				ExpectError: regexp.MustCompile("spaceship api error"),
			},
		},
	})

}

func TestAccDomain_autoRenewal(t *testing.T) {

	configAutoRenewTrue := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
	
	auto_renew = true
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	configAutoRenewFalse := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
	
	auto_renew = false
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// resource contains current autorenew value
			{
				Config: emptyDomainResourceConfiguration,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(domainResourceFullName, "auto_renew"),
				),
			},
			// resource has auto_renew value true
			{
				Config: configAutoRenewTrue,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "auto_renew", "true"),
				),
			},
			// auto_renew value false
			{
				Config: configAutoRenewFalse,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "auto_renew", "false"),
				),
			},
		}})
}

// TestAccDomain_autoRenewalMismatchOnCreate verifies the first apply converges
// auto_renew when the config value differs from the domain's actual setting.
func TestAccDomain_autoRenewalMismatchOnCreate(t *testing.T) {
	configAutoRenewTrue := domainConfigWithAutoRenew(true)
	configAutoRenewFalse := domainConfigWithAutoRenew(false)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// infra false, config true — Create must push the change
			{
				PreConfig: func() { testAccSetAutoRenew(t, false) },
				Config:    configAutoRenewTrue,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "auto_renew", "true"),
				),
			},
			// converged for real: refresh-backed empty plan
			{
				Config: configAutoRenewTrue,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			// reverse direction on a forced recreate: infra true, config false
			{
				PreConfig: func() { testAccSetAutoRenew(t, true) },
				Taint:     []string{domainResourceFullName},
				Config:    configAutoRenewFalse,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "auto_renew", "false"),
				),
			},
			// converged: re-plan is empty
			{
				Config: configAutoRenewFalse,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// testAccSetAutoRenew sets the test domain's auto_renew out-of-band via the SDK.
func testAccSetAutoRenew(t *testing.T, value bool) {
	t.Helper()

	testClient, err := testAccClient()
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}
	if _, err := testClient.UpdateAutoRenew(context.Background(), testAccDomainValue(), value); err != nil {
		t.Fatalf("failed to set auto_renew=%v: %v", value, err)
	}
}

func domainConfigWithAutoRenew(value bool) string {
	return fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"

	auto_renew = %t
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue(), value)
}

func TestAccDomain_nameservers(t *testing.T) {
	nsProviderBasicConfig := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
	
	nameservers = {
		provider = "basic"
	}
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	nsProviderCustomConfig := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
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
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1
			// adopt on creation
			{
				Config: emptyDomainResourceConfiguration,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(domainResourceFullName, "nameservers.hosts.0"),
					resource.TestCheckResourceAttrSet(domainResourceFullName, "nameservers.provider"),
				),
			},
			// Step 2
			// verify no changes, changes in code only
			{
				Config:             nsProviderBasicConfig,
				ExpectNonEmptyPlan: false,
			},
			// Step 3
			// update nameservers to custom
			{
				Config: nsProviderCustomConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemAttr(domainResourceFullName, "nameservers.hosts.*", "ns-1063.awsdns-04.org"),
					resource.TestCheckTypeSetElemAttr(domainResourceFullName, "nameservers.hosts.*", "ns-1578.awsdns-05.co.uk"),
					resource.TestCheckTypeSetElemAttr(domainResourceFullName, "nameservers.hosts.*", "ns-401.awsdns-50.com"),
					resource.TestCheckTypeSetElemAttr(domainResourceFullName, "nameservers.hosts.*", "ns-669.awsdns-19.net"),
					resource.TestCheckResourceAttr(domainResourceFullName, "nameservers.provider", "custom"),
				),
			},

			// Step 4
			// reset to basic back
			{
				Config: nsProviderBasicConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "nameservers.provider", "basic"),
					resource.TestCheckTypeSetElemAttr(domainResourceFullName, "nameservers.hosts.*", "launch1.spaceship.net"),
				),
			},
		},
	})
}

// TestAccDomain_nameserversMismatchOnCreate verifies the first apply converges
// nameservers when the config differs from the domain's actual setting.
func TestAccDomain_nameserversMismatchOnCreate(t *testing.T) {
	customHosts := []string{
		"ns-669.awsdns-19.net",
		"ns-1578.awsdns-05.co.uk",
	}

	configCustom := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"

	nameservers = {
		provider = "custom"
		hosts = [
			"ns-669.awsdns-19.net",
			"ns-1578.awsdns-05.co.uk",
		]
	}
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	configBasic := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"

	nameservers = {
		provider = "basic"
	}
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// infra basic, config custom — Create must push the change
			{
				PreConfig: func() { testAccSetNameservers(t, "basic", nil) },
				Config:    configCustom,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "nameservers.provider", "custom"),
					resource.TestCheckTypeSetElemAttr(domainResourceFullName, "nameservers.hosts.*", "ns-669.awsdns-19.net"),
					resource.TestCheckTypeSetElemAttr(domainResourceFullName, "nameservers.hosts.*", "ns-1578.awsdns-05.co.uk"),
				),
			},
			// converged for real: refresh-backed empty plan
			{
				Config: configCustom,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			// reverse direction on a forced recreate: infra custom, config basic
			{
				PreConfig: func() { testAccSetNameservers(t, "custom", customHosts) },
				Taint:     []string{domainResourceFullName},
				Config:    configBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "nameservers.provider", "basic"),
					resource.TestCheckTypeSetElemAttr(domainResourceFullName, "nameservers.hosts.*", "launch1.spaceship.net"),
				),
			},
			// converged: re-plan is empty
			{
				Config: configBasic,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// testAccSetNameservers sets the test domain's nameservers out-of-band via the
// SDK. The API rejects updates matching the current provider, so it skips when
// the domain is already in the desired state.
func testAccSetNameservers(t *testing.T, nsProvider string, hosts []string) {
	t.Helper()

	testClient, err := testAccClient()
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	info, err := testClient.GetDomainInfo(context.Background(), testAccDomainValue())
	if err != nil {
		t.Fatalf("failed to read domain info: %v", err)
	}
	if strings.EqualFold(info.Nameservers.Provider, nsProvider) &&
		(strings.EqualFold(nsProvider, "basic") || sameHostSet(info.Nameservers.Hosts, hosts)) {
		return
	}

	err = testClient.UpdateDomainNameServers(context.Background(), testAccDomainValue(), client.UpdateNameserverRequest{
		Provider: client.NameserverProvider(nsProvider),
		Hosts:    hosts,
	})
	if err != nil {
		t.Fatalf("failed to set nameservers provider=%s: %v", nsProvider, err)
	}
}

func sameHostSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]struct{}, len(a))
	for _, h := range a {
		set[strings.ToLower(h)] = struct{}{}
	}
	for _, h := range b {
		if _, ok := set[strings.ToLower(h)]; !ok {
			return false
		}
	}
	return true
}

func TestAccDomain_nameserversValidationErrors(t *testing.T) {
	nsProviderBasicWithHosts := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"
	
	nameservers = {
		provider = "basic"
		hosts = ["ns1.example.com", "ns2.example.com"]
	}
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	nsProviderCustomWithNoHosts := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"

	nameservers = {
		provider = "custom"
	}
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	nsProviderCustomWithDefaultHosts := fmt.Sprintf(`
provider "%s" {}

resource "%s" "%s" {
	domain = "%s"

	nameservers = {
		provider = "custom"
		hosts = ["launch1.spaceship.net", "launch2.spaceship.net"]
	}
}
`, providerName, domainResourceRef, domainResourceName, testAccDomainValue())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test errors on wrong configuration
			{
				Config:      nsProviderCustomWithNoHosts,
				ExpectError: regexp.MustCompile("The 'hosts' field is required when provider is 'custom'."),
			},
			{
				Config:      nsProviderBasicWithHosts,
				ExpectError: regexp.MustCompile("The 'hosts' field must be omitted when provider is 'basic'."),
			},
			{
				Config:      nsProviderCustomWithDefaultHosts,
				ExpectError: regexp.MustCompile("The default Spaceship nameservers can only be used with provider \"basic\"."),
			},
		},
	})
}

func TestAccDomain_resourceImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			//step 1 creation
			{
				Config: emptyDomainResourceConfiguration,
			},
			// import
			{
				ResourceName:                         domainResourceFullName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        testAccDomainValue(),
				ImportStateVerifyIdentifierAttribute: "domain",
			},
		},
	})
}
