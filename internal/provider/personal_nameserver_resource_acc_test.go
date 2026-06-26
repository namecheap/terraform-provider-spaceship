package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/namecheap/go-spaceship-sdk/client"
)

// Full lifecycle: create, empty-plan re-apply, IP update in place, host rename
// in place, import, destroy.
func TestAccPersonalNameserver_lifecycle(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	// host is a label relative to the domain, not an FQDN — the registry joins
	// "ns1" + domain into "ns1.<domain>".
	host1 := "ns1"
	host2 := "ns2"
	resourceName := "spaceship_personal_nameserver.test"

	config := func(host string, ips string) string {
		return fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_personal_nameserver" "test" {
  domain = %[1]q
  host   = %[2]q
  ips    = %[3]s
}
`, domain, host, ips)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPersonalNameserverAbsent(domain, host1, host2),
		Steps: []resource.TestStep{
			{
				Config: config(host1, `["1.2.3.4"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", domain+"/"+host1),
					resource.TestCheckResourceAttr(resourceName, "host", host1),
					resource.TestCheckResourceAttr(resourceName, "ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ips.*", "1.2.3.4"),
				),
			},
			{
				// Re-apply same config — must converge with no diff.
				Config: config(host1, `["1.2.3.4"]`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				// IP-only update stays on the same host; re-plan after refresh
				// must converge with no diff.
				Config: config(host1, `["5.6.7.8", "9.10.11.12"]`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "host", host1),
					resource.TestCheckResourceAttr(resourceName, "ips.#", "2"),
				),
			},
			{
				// Host rename updates in place (no destroy/create); re-plan after
				// refresh must converge with no diff.
				Config: config(host2, `["5.6.7.8", "9.10.11.12"]`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", domain+"/"+host2),
					resource.TestCheckResourceAttr(resourceName, "host", host2),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// External-removal reconciliation: when the host is deleted outside Terraform,
// Read must drop it from state so the next plan recreates it instead of erroring.
func TestAccPersonalNameserver_disappears(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := "ns1"
	resourceName := "spaceship_personal_nameserver.test"

	config := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_personal_nameserver" "test" {
  domain = %[1]q
  host   = %[2]q
  ips    = ["1.2.3.4"]
}
`, domain, host)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPersonalNameserverAbsent(domain, host),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr(resourceName, "id", domain+"/"+host),
			},
			{
				// Delete the host out-of-band, then refresh-only (reuses the
				// prior config). Read removes it from state, leaving a non-empty
				// recreate plan rather than a hard error.
				PreConfig: func() {
					testClient, err := testAccClient()
					if err != nil {
						t.Fatalf("build client: %s", err)
					}
					if err := testClient.DeletePersonalNameserver(context.Background(), domain, host); err != nil {
						t.Fatalf("out-of-band delete: %s", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckPersonalNameserverAbsent(domain string, hosts ...string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		testClient, err := testAccClient()
		if err != nil {
			return err
		}
		list, err := testClient.ListPersonalNameservers(context.Background(), domain)
		if err != nil {
			if client.IsNotFoundError(err) {
				return nil
			}
			return err
		}
		for _, ns := range list.Records {
			for _, host := range hosts {
				if strings.EqualFold(ns.Host, host) {
					return fmt.Errorf("personal nameserver %s still present in domain %s", ns.Host, domain)
				}
			}
		}
		return nil
	}
}
