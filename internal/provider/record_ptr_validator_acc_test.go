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

// ptrHost returns a unique record name per test so parallel/repeat runs
// against the same SPACESHIP_TEST_DOMAIN don't collide.
func ptrHost(suffix string) string {
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	return fmt.Sprintf("%s-ptr-%s", prefix, suffix)
}

// Verifies that a standard PTR record round-trips through create, import,
// and destroy.
func TestAccDNSRecords_ptrRecord(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := ptrHost("basic")
	pointer := "host.example.com"

	records := []testAccDNSRecord{
		{
			Type: "PTR",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"pointer": pointer,
			},
		},
	}

	resourceName := "spaceship_dns_records.test"
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "records.0.type", "PTR"),
		resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.0.pointer", pointer),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, records),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force", "records"},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "PTR", host),
	})
}

// Verifies PTR lifecycle: create, update the pointer to a mixed-case hostname
// (exercises delete+upsert), empty re-plan (verifies case-insensitive pointer
// matching), and import.
func TestAccDNSRecords_ptrRecordCreateAndUpdate(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := ptrHost("lifecycle")
	initialTarget := "host.example.com"
	updatedTarget := "Other.Example.COM"
	resourceName := "spaceship_dns_records.test"

	mkRecord := func(target string) []testAccDNSRecord {
		return []testAccDNSRecord{{
			Type:        "PTR",
			Name:        host,
			TTL:         intPointer(3600),
			StringAttrs: map[string]string{"pointer": target},
		}}
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord(initialTarget)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "PTR"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.pointer", initialTarget),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "PTR", host),
	})
}

// Verifies that "@" as pointer is rejected at plan time. Pointing a PTR
// at the zone apex is never a valid configuration.
func TestAccDNSRecords_ptrRecordApexTargetFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "PTR",
			Name: ptrHost("apex"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"pointer": "@",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Pointer Value"),
			},
		},
	})
}

// Verifies that "*" as pointer is rejected at plan time.
func TestAccDNSRecords_ptrRecordWildcardTargetFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "PTR",
			Name: ptrHost("wildcard"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"pointer": "*",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Pointer Value"),
			},
		},
	})
}

// Verifies that an invalid pointer (starts with a dot) is rejected at plan time.
func TestAccDNSRecords_ptrRecordInvalidPointerFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "PTR",
			Name: ptrHost("badname"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"pointer": ".invalid",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Pointer Value"),
			},
		},
	})
}

// Verifies that a missing pointer field is rejected at plan time.
func TestAccDNSRecords_ptrRecordMissingPointerFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "PTR",
			Name: ptrHost("noptr"),
			TTL:  intPointer(3600),
			// no pointer
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Missing Required Field"),
			},
		},
	})
}

// Verifies that a PTR record created directly via the API can be imported
// into Terraform and that a subsequent plan is empty.
func TestAccDNSRecords_ptrRecordImportPreExisting(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := ptrHost("import")
	pointer := "host.example.com"

	preExistingRecord := client.DNSRecord{
		Type:    "PTR",
		Name:    host,
		TTL:     3600,
		Pointer: pointer,
	}

	preConfig := func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to create test client: %v", err)
		}
		if err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Fatalf("failed to pre-seed PTR record: %v", err)
		}
	}

	t.Cleanup(func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Logf("cleanup: failed to create test client: %v", err)
			return
		}
		if err := testClient.DeleteDNSRecords(context.Background(), domain, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Logf("cleanup: failed to delete pre-seeded PTR record: %v", err)
		}
	})

	records := []testAccDNSRecord{
		{
			Type: "PTR",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"pointer": pointer,
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
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "PTR"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.pointer", pointer),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "PTR", host),
	})
}

// Probes the API directly (bypassing the provider) to empirically verify
// that the Spaceship API rejects "@" and "*" as PTR pointer values.
// This is the load-bearing test for the ValidatePointer pre-check:
// if the API ever starts accepting these values, we should drop the
// pre-check to keep "mirror API, don't invent stricter rules" honest.
func TestAccDNSRecords_ptrRecordAPIRejectsApexAndWildcard(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	testClient, err := testAccClient()
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	cases := []struct {
		name    string
		pointer string
	}{
		{"apex", "@"},
		{"wildcard", "*"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host := ptrHost("api-" + tc.name)
			rec := client.DNSRecord{
				Type:    "PTR",
				Name:    host,
				TTL:     3600,
				Pointer: tc.pointer,
			}
			err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{rec})
			if err == nil {
				// Belt-and-suspenders cleanup in case the API accepted the record.
				_ = testClient.DeleteDNSRecords(context.Background(), domain, []client.DNSRecord{rec})
				t.Fatalf("expected API to reject pointer=%q, but it was accepted", tc.pointer)
			}
		})
	}
}
