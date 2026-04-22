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

// cnameHost returns a unique record name per test so parallel/repeat runs
// against the same SPACESHIP_TEST_DOMAIN don't collide.
func cnameHost(suffix string) string {
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	return fmt.Sprintf("%s-cname-%s", prefix, suffix)
}

// Verifies that plan-time validation rejects cname values the API would
// 422 at apply time: leading-dot hostname, apex "@", and wildcard "*".
func TestAccDNSRecords_cnameValidation(t *testing.T) {
	testAccPreCheck(t)
	domain := testAccDomainValue()

	cases := []struct {
		suffix string
		cname  string
	}{
		{"baddot", ".invalid"},
		{"apex", "@"},
		{"wildcard", "*"},
	}

	steps := make([]resource.TestStep, 0, len(cases))
	for _, tc := range cases {
		records := []testAccDNSRecord{
			{
				Type:        "CNAME",
				Name:        cnameHost(tc.suffix),
				TTL:         intPointer(3600),
				StringAttrs: map[string]string{"cname": tc.cname},
			},
		}
		steps = append(steps, resource.TestStep{
			Config:      testAccDNSRecordsConfig(domain, records),
			ExpectError: regexp.MustCompile("Invalid CName Value"),
		})
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    steps,
	})
}

// Verifies CNAME lifecycle on a single host: create, update the target to
// a mixed-case hostname (exercises delete+upsert), empty re-plan (verifies
// case-insensitive cname matching), and import.
func TestAccDNSRecords_cnameCreateAndUpdate(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := cnameHost("lifecycle")
	initialTarget := "target.example.com"
	updatedTarget := "Origin.Example.COM"
	resourceName := "spaceship_dns_records.test"

	mkRecord := func(target string) []testAccDNSRecord {
		return []testAccDNSRecord{{
			Type:        "CNAME",
			Name:        host,
			TTL:         intPointer(3600),
			StringAttrs: map[string]string{"cname": target},
		}}
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord(initialTarget)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "CNAME"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.cname", initialTarget),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord(updatedTarget)),
				Check:  resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
			},
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord(updatedTarget)),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CNAME", host),
	})
}

// Verifies that a CNAME record created directly via the API can be imported
// into Terraform and that a subsequent plan is empty.
func TestAccDNSRecords_cnameRecordImportPreExisting(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := cnameHost("import")
	target := "origin.example.com"

	preExistingRecord := client.DNSRecord{
		Type:  "CNAME",
		Name:  host,
		TTL:   3600,
		CName: target,
	}

	preConfig := func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to create test client: %v", err)
		}
		if err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Fatalf("failed to pre-seed CNAME record: %v", err)
		}
	}

	t.Cleanup(func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Logf("cleanup: failed to create test client: %v", err)
			return
		}
		if err := testClient.DeleteDNSRecords(context.Background(), domain, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Logf("cleanup: failed to delete pre-seeded CNAME record: %v", err)
		}
	})

	records := []testAccDNSRecord{
		{
			Type: "CNAME",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"cname": target,
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
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "CNAME"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.cname", target),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CNAME", host),
	})
}
