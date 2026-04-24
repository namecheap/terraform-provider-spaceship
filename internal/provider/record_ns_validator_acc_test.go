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

// nsHost returns a unique record name per test so parallel/repeat runs
// against the same SPACESHIP_TEST_DOMAIN don't collide.
func nsHost(suffix string) string {
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	return fmt.Sprintf("%s-ns-%s", prefix, suffix)
}

// Verifies that a standard NS record round-trips through create, import,
// and destroy.
func TestAccDNSRecords_nsRecord(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := nsHost("basic")
	nameserver := "ns1.example.com"

	records := []testAccDNSRecord{
		{
			Type: "NS",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"nameserver": nameserver,
			},
		},
	}

	resourceName := "spaceship_dns_records.test"
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "records.0.type", "NS"),
		resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.0.nameserver", nameserver),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "NS", host),
	})
}

// Verifies NS lifecycle: create, update the nameserver to a mixed-case
// hostname (exercises delete+upsert), empty re-plan (verifies case-insensitive
// nameserver matching), and import.
func TestAccDNSRecords_nsRecordCreateAndUpdate(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := nsHost("lifecycle")
	initialTarget := "ns1.example.com"
	updatedTarget := "NS2.Example.COM"
	resourceName := "spaceship_dns_records.test"

	mkRecord := func(target string) []testAccDNSRecord {
		return []testAccDNSRecord{{
			Type:        "NS",
			Name:        host,
			TTL:         intPointer(3600),
			StringAttrs: map[string]string{"nameserver": target},
		}}
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord(initialTarget)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "NS"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.nameserver", initialTarget),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "NS", host),
	})
}

// Verifies that two NS records at the same host (the canonical delegation
// setup) round-trip cleanly: both get written, state lists both, and
// re-apply is a no-op.
func TestAccDNSRecords_nsRecordMultipleAtSameHost(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := nsHost("multi")
	resourceName := "spaceship_dns_records.test"

	records := []testAccDNSRecord{
		{
			Type:        "NS",
			Name:        host,
			TTL:         intPointer(3600),
			StringAttrs: map[string]string{"nameserver": "ns1.example.com"},
		},
		{
			Type:        "NS",
			Name:        host,
			TTL:         intPointer(3600),
			StringAttrs: map[string]string{"nameserver": "ns2.example.com"},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, records),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "2"),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, records),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "NS", host),
	})
}

// Verifies that "@" as nameserver is rejected at plan time. Pointing a
// delegation at the zone itself is circular.
func TestAccDNSRecords_nsRecordApexTargetFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "NS",
			Name: nsHost("apex"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"nameserver": "@",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Nameserver Value"),
			},
		},
	})
}

// Verifies that "*" as nameserver is rejected at plan time.
func TestAccDNSRecords_nsRecordWildcardTargetFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "NS",
			Name: nsHost("wildcard"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"nameserver": "*",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Nameserver Value"),
			},
		},
	})
}

// Verifies that an invalid nameserver (starts with a dot) is rejected at plan time.
func TestAccDNSRecords_nsRecordInvalidNameserverFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "NS",
			Name: nsHost("badname"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"nameserver": ".invalid",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Nameserver Value"),
			},
		},
	})
}

// Verifies that a missing nameserver field is rejected at plan time.
func TestAccDNSRecords_nsRecordMissingNameserverFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "NS",
			Name: nsHost("nons"),
			TTL:  intPointer(3600),
			// no nameserver
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

// Verifies that an NS record created directly via the API can be imported
// into Terraform and that a subsequent plan is empty.
func TestAccDNSRecords_nsRecordImportPreExisting(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := nsHost("import")
	nameserver := "ns1.example.com"

	preExistingRecord := client.DNSRecord{
		Type:       "NS",
		Name:       host,
		TTL:        3600,
		Nameserver: nameserver,
	}

	preConfig := func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to create test client: %v", err)
		}
		if err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Fatalf("failed to pre-seed NS record: %v", err)
		}
	}

	t.Cleanup(func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Logf("cleanup: failed to create test client: %v", err)
			return
		}
		if err := testClient.DeleteDNSRecords(context.Background(), domain, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Logf("cleanup: failed to delete pre-seeded NS record: %v", err)
		}
	})

	records := []testAccDNSRecord{
		{
			Type: "NS",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"nameserver": nameserver,
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
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "NS"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.nameserver", nameserver),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "NS", host),
	})
}

// Probes the API directly (bypassing the provider) to empirically verify
// that the Spaceship API rejects "@" and "*" as NS nameserver values.
// This is the load-bearing test for the ValidateNameserver pre-check:
// if the API ever starts accepting these values, we should drop the
// pre-check to keep "mirror API, don't invent stricter rules" honest.
func TestAccDNSRecords_nsRecordAPIRejectsApexAndWildcard(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	testClient, err := testAccClient()
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	cases := []struct {
		name       string
		nameserver string
	}{
		{"apex", "@"},
		{"wildcard", "*"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host := nsHost("api-" + tc.name)
			rec := client.DNSRecord{
				Type:       "NS",
				Name:       host,
				TTL:        3600,
				Nameserver: tc.nameserver,
			}
			err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{rec})
			if err == nil {
				// Belt-and-suspenders cleanup in case the API accepted the record.
				_ = testClient.DeleteDNSRecords(context.Background(), domain, []client.DNSRecord{rec})
				t.Fatalf("expected API to reject nameserver=%q, but it was accepted", tc.nameserver)
			}
		})
	}
}
