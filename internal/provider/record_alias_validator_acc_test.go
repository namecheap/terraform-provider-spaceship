package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/namecheap/go-spaceship-sdk/client"
)

// aliasHost returns a unique record name per test so parallel/repeat runs
// against the same SPACESHIP_TEST_DOMAIN don't collide.
func aliasHost(suffix string) string {
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	return fmt.Sprintf("%s-alias-%s", prefix, suffix)
}

// Verifies that a standard ALIAS record round-trips through create, import,
// and destroy.
func TestAccDNSRecords_aliasRecord(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := aliasHost("basic")
	aliasName := "other.example.com"

	records := []testAccDNSRecord{
		{
			Type: "ALIAS",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"alias_name": aliasName,
			},
		},
	}

	resourceName := "spaceship_dns_records.test"
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "records.0.type", "ALIAS"),
		resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.0.alias_name", aliasName),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "ALIAS", host),
	})
}

// Verifies that "@" as the alias_name *target* is rejected at plan time (the
// target must be a real domain name, not the apex shorthand). This is distinct
// from an apex ALIAS record — see TestAccDNSRecords_aliasRecordApexNameFailsPlan.
func TestAccDNSRecords_aliasRecordApexTargetFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "ALIAS",
			Name: aliasHost("apex"),
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"alias_name": "@",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Alias Name Value"),
			},
		},
	})
}

// Verifies that an apex ALIAS (name "@") is rejected at plan time. Spaceship
// silently stores apex aliases as a root CNAME, which Terraform cannot
// reconcile as an ALIAS, so the provider rejects it and points at the CNAME form.
func TestAccDNSRecords_aliasRecordApexNameFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "ALIAS",
			Name: "@",
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"alias_name": "target.example.com",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Apex ALIAS Record"),
			},
		},
	})
}

// Verifies that an apex ALIAS (name "@") is rejected even when it sits among
// other valid records — the per-element validator must fire on the offending
// record, not just when the ALIAS is the sole record in the list.
func TestAccDNSRecords_aliasRecordApexNameAmongMultipleFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "A",
			Name: aliasHost("multi-a"),
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"address": "192.0.2.1",
			},
		},
		{
			Type: "ALIAS",
			Name: "@",
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"alias_name": "target.example.com",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Apex ALIAS Record"),
			},
		},
	})
}

// Verifies that an invalid alias_name (starts with a dot) is rejected at plan time.
func TestAccDNSRecords_aliasRecordInvalidAliasNameFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "ALIAS",
			Name: aliasHost("badname"),
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"alias_name": ".invalid",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Alias Name Value"),
			},
		},
	})
}

// Verifies that a missing alias_name field is rejected at plan time.
func TestAccDNSRecords_aliasRecordMissingAliasNameFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "ALIAS",
			Name: aliasHost("noalias"),
			TTL:  intPtr(3600),
			// no alias_name
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

// Verifies that an ALIAS record created directly via the API can be imported
// into Terraform and that a subsequent plan is empty.
func TestAccDNSRecords_aliasRecordImportPreExisting(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := aliasHost("import")
	aliasName := "origin.example.com"

	preExistingRecord := client.DNSRecord{
		Type:      "ALIAS",
		Name:      host,
		TTL:       3600,
		AliasName: aliasName,
	}

	// Seed the record via the API before Terraform runs.
	preConfig := func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to create test client: %v", err)
		}
		if err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Fatalf("failed to pre-seed ALIAS record: %v", err)
		}
	}

	// Clean up the pre-seeded record after the test.
	t.Cleanup(func() {
		testClient, err := testAccClient()
		if err != nil {
			return
		}
		_ = testClient.DeleteDNSRecords(context.Background(), domain, []client.DNSRecord{preExistingRecord})
	})

	// Terraform config matches the pre-seeded record exactly.
	records := []testAccDNSRecord{
		{
			Type: "ALIAS",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"alias_name": aliasName,
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
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "ALIAS"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.alias_name", aliasName),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "ALIAS", host),
	})
}
