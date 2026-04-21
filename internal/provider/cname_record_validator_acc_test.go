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

// Verifies that a standard CNAME record round-trips through create, import,
// and destroy.
func TestAccDNSRecords_cnameRecord(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := cnameHost("basic")
	target := "target.example.com"

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
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "records.0.type", "CNAME"),
		resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.0.cname", target),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CNAME", host),
	})
}

// Verifies that "@" as cname is rejected at plan time. The API treats "@" as
// the zone-relative apex shorthand; a CNAME target must be a real hostname.
func TestAccDNSRecords_cnameRecordApexTargetFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "CNAME",
			Name: cnameHost("apex"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"cname": "@",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid CName Value"),
			},
		},
	})
}

// Verifies that "*" as cname is rejected at plan time. A CNAME target must
// be a concrete hostname, not a wildcard placeholder.
func TestAccDNSRecords_cnameRecordWildcardTargetFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "CNAME",
			Name: cnameHost("wild"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"cname": "*",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid CName Value"),
			},
		},
	})
}

// Verifies that an invalid cname (starts with a dot) is rejected at plan time.
func TestAccDNSRecords_cnameRecordInvalidCNameFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "CNAME",
			Name: cnameHost("badname"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"cname": ".invalid",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid CName Value"),
			},
		},
	})
}

// Verifies that a missing cname field is rejected at plan time.
func TestAccDNSRecords_cnameRecordMissingCNameFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "CNAME",
			Name: cnameHost("nocname"),
			TTL:  intPointer(3600),
			// no cname
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

	// Seed the record via the API before Terraform runs.
	preConfig := func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to create test client: %v", err)
		}
		if err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Fatalf("failed to pre-seed CNAME record: %v", err)
		}
	}

	// Clean up the pre-seeded record after the test.
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
				// Step 1: Apply with the pre-existing record already on
				// the API. Terraform should adopt it without changes.
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
				// Step 2: Re-apply. Plan should be empty since state
				// already matches configuration and API.
				Config: testAccDNSRecordsConfig(domain, records),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				// Step 3: Import and verify.
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force", "records"},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CNAME", host),
	})
}

// Verifies that changing the cname target of an existing record works. The
// diff signature keys on type+name+cname, so a target change is a delete+upsert,
// not an in-place update — this exercises both paths.
func TestAccDNSRecords_cnameRecordUpdateTarget(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := cnameHost("update")

	initialRecords := []testAccDNSRecord{
		{
			Type: "CNAME",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"cname": "origin-a.example.com",
			},
		},
	}

	updatedRecords := []testAccDNSRecord{
		{
			Type: "CNAME",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"cname": "origin-b.example.com",
			},
		},
	}

	resourceName := "spaceship_dns_records.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, initialRecords),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.0.cname", "origin-a.example.com"),
				),
			},
			{
				// Same host, new target → old record should be deleted and new
				// one upserted. Final state has only the new record.
				Config: testAccDNSRecordsConfig(domain, updatedRecords),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.cname", "origin-b.example.com"),
				),
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CNAME", host),
	})
}

// Verifies that cname matching is case-insensitive end-to-end: applying a
// mixed-case cname and re-applying should produce an empty plan, regardless
// of what casing the API persists.
func TestAccDNSRecords_cnameRecordCaseInsensitive(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := cnameHost("case")

	records := []testAccDNSRecord{
		{
			Type: "CNAME",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"cname": "Origin.Example.COM",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, records),
			},
			{
				// If the API normalizes casing on the server and returns
				// something different from what we sent, this will show drift
				// and fail — surfacing a mismatch between upsert matching
				// and read normalization.
				Config: testAccDNSRecordsConfig(domain, records),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CNAME", host),
	})
}
