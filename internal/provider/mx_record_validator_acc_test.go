package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// mxHost returns a unique record name per test so parallel/repeat runs
// against the same SPACESHIP_TEST_DOMAIN don't collide.
func mxHost(suffix string) string {
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	return fmt.Sprintf("%s-mx-%s", prefix, suffix)
}

// Verifies that plan-time validation rejects exchange values the API would
// 422 at apply time: leading-dot hostname, apex "@", and wildcard "*".
func TestAccDNSRecords_mxValidation(t *testing.T) {
	testAccPreCheck(t)
	domain := testAccDomainValue()

	cases := []struct {
		suffix   string
		exchange string
	}{
		{"baddot", ".mail." + domain},
		{"apex", "@"},
		{"wildcard", "*"},
	}

	steps := make([]resource.TestStep, 0, len(cases))
	for _, tc := range cases {
		records := []testAccDNSRecord{
			{
				Type:        "MX",
				Name:        mxHost(tc.suffix),
				TTL:         intPointer(3600),
				StringAttrs: map[string]string{"exchange": tc.exchange},
				IntAttrs:    map[string]int{"preference": 10},
			},
		}
		steps = append(steps, resource.TestStep{
			Config:      testAccDNSRecordsConfig(domain, records),
			ExpectError: regexp.MustCompile("Invalid Exchange Value"),
		})
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    steps,
	})
}

// Verifies MX lifecycle on a single host: create, update preference
// (delete+upsert path), update TTL, empty re-plan, and import.
func TestAccDNSRecords_mxCreateAndUpdate(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := mxHost("lifecycle")
	exchange := fmt.Sprintf("mail.%s", domain)
	resourceName := "spaceship_dns_records.test"

	mkRecord := func(ttl, pref int) []testAccDNSRecord {
		return []testAccDNSRecord{{
			Type:        "MX",
			Name:        host,
			TTL:         intPointer(ttl),
			StringAttrs: map[string]string{"exchange": exchange},
			IntAttrs:    map[string]int{"preference": pref},
		}}
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord(3600, 10)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "MX"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.exchange", exchange),
					resource.TestCheckResourceAttr(resourceName, "records.0.preference", "10"),
					resource.TestCheckResourceAttr(resourceName, "records.0.ttl", "3600"),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord(3600, 20)),
				Check:  resource.TestCheckResourceAttr(resourceName, "records.0.preference", "20"),
			},
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord(300, 20)),
				Check:  resource.TestCheckResourceAttr(resourceName, "records.0.ttl", "300"),
			},
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord(300, 20)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force", "records"},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "MX", host),
	})
}

// Verifies three edge scenarios coexisting in one config: preference=0
// round-trips, mixed-case exchange converges (case-insensitive matching),
// and multiple MX records on the same host with different exchanges.
func TestAccDNSRecords_mxEdgeValues(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	zeroHost := mxHost("zero")
	caseHost := mxHost("case")
	multiHost := mxHost("multi")
	resourceName := "spaceship_dns_records.test"

	records := []testAccDNSRecord{
		{
			Type:        "MX",
			Name:        zeroHost,
			TTL:         intPointer(3600),
			StringAttrs: map[string]string{"exchange": fmt.Sprintf("mail.%s", domain)},
			IntAttrs:    map[string]int{"preference": 0},
		},
		{
			Type:        "MX",
			Name:        caseHost,
			TTL:         intPointer(3600),
			StringAttrs: map[string]string{"exchange": fmt.Sprintf("Mail.%s", domain)},
			IntAttrs:    map[string]int{"preference": 10},
		},
		{
			Type:        "MX",
			Name:        multiHost,
			TTL:         intPointer(3600),
			StringAttrs: map[string]string{"exchange": fmt.Sprintf("mail1.%s", domain)},
			IntAttrs:    map[string]int{"preference": 10},
		},
		{
			Type:        "MX",
			Name:        multiHost,
			TTL:         intPointer(3600),
			StringAttrs: map[string]string{"exchange": fmt.Sprintf("mail2.%s", domain)},
			IntAttrs:    map[string]int{"preference": 20},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, records),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "records.0.preference", "0"),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, records),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckDNSRecordAbsent(domain, "MX", zeroHost),
			testAccCheckDNSRecordAbsent(domain, "MX", caseHost),
			testAccCheckDNSRecordAbsent(domain, "MX", multiHost),
		),
	})
}

// Verifies that an MX record created directly via the API can be adopted by
// Terraform and that a subsequent plan is empty.
func TestAccDNSRecords_mxRecordImportPreExisting(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := mxHost("import")
	exchange := fmt.Sprintf("mail.%s", domain)
	preference := 10

	preExistingRecord := client.DNSRecord{
		Type:       "MX",
		Name:       host,
		TTL:        3600,
		Exchange:   exchange,
		Preference: &preference,
	}

	preConfig := func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to create test client: %v", err)
		}
		if err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Fatalf("failed to pre-seed MX record: %v", err)
		}
	}

	t.Cleanup(func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Logf("cleanup: failed to create test client: %v", err)
			return
		}
		if err := testClient.DeleteDNSRecords(context.Background(), domain, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Logf("cleanup: failed to delete pre-seeded MX record: %v", err)
		}
	})

	records := []testAccDNSRecord{
		{
			Type: "MX",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": exchange,
			},
			IntAttrs: map[string]int{
				"preference": preference,
			},
		},
	}

	resourceName := "spaceship_dns_records.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: preConfig,
				Config:    testAccDNSRecordsConfig(domain, records),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "MX"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.exchange", exchange),
					resource.TestCheckResourceAttr(resourceName, "records.0.preference", fmt.Sprintf("%d", preference)),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, records),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force", "records"},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "MX", host),
	})
}
