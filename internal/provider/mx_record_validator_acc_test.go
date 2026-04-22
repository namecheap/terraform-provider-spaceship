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

// Verifies that a standard MX record round-trips through create, import,
// and destroy.
func TestAccDNSRecords_mxRecord(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := mxHost("basic")
	exchange := fmt.Sprintf("mail.%s", domain)

	records := []testAccDNSRecord{
		{
			Type: "MX",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": exchange,
			},
			IntAttrs: map[string]int{
				"preference": 10,
			},
		},
	}

	resourceName := "spaceship_dns_records.test"
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "records.0.type", "MX"),
		resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.0.exchange", exchange),
		resource.TestCheckResourceAttr(resourceName, "records.0.preference", "10"),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "MX", host),
	})
}

// Verifies that a missing exchange field is rejected at plan time.
func TestAccDNSRecords_mxRecordMissingExchangeFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "MX",
			Name: mxHost("noexchange"),
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"preference": 10,
			},
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

// Verifies that a missing preference field is rejected at plan time.
func TestAccDNSRecords_mxRecordMissingPreferenceFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "MX",
			Name: mxHost("nopref"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": fmt.Sprintf("mail.%s", testAccDomainValue()),
			},
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

// Verifies that an invalid exchange (starts with dot) is rejected at plan time
// by the client-layer ValidateExchange check.
func TestAccDNSRecords_mxRecordInvalidExchangeFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "MX",
			Name: mxHost("badexchange"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": ".mail.example.com",
			},
			IntAttrs: map[string]int{
				"preference": 10,
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Exchange Value"),
			},
		},
	})
}

// Verifies that a preference above uint16 range is rejected at plan time.
// The schema-level int64validator.Between(0, 65535) catches this before the
// Object validator runs, so the error shape differs from other fields.
func TestAccDNSRecords_mxRecordInvalidPreferenceFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "MX",
			Name: mxHost("badpref"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": fmt.Sprintf("mail.%s", testAccDomainValue()),
			},
			IntAttrs: map[string]int{
				"preference": 70000,
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Attribute Value"),
			},
		},
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

// Verifies that changing preference on an existing MX record converges. The
// diff signature keys on exchange+preference, so a preference change is a
// delete+upsert rather than an in-place update — this exercises both paths.
func TestAccDNSRecords_mxRecordUpdatePreference(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := mxHost("update")
	exchange := fmt.Sprintf("mail.%s", domain)

	initialRecords := []testAccDNSRecord{
		{
			Type: "MX",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": exchange,
			},
			IntAttrs: map[string]int{
				"preference": 10,
			},
		},
	}

	updatedRecords := []testAccDNSRecord{
		{
			Type: "MX",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": exchange,
			},
			IntAttrs: map[string]int{
				"preference": 20,
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
					resource.TestCheckResourceAttr(resourceName, "records.0.preference", "10"),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, updatedRecords),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.preference", "20"),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, updatedRecords),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "MX", host),
	})
}

// Verifies that two MX records on the same host with different exchanges
// coexist. Real-world MX config typically pairs a primary and secondary
// mail server on the zone apex with different preferences.
func TestAccDNSRecords_mxRecordMultipleExchangesSameHost(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := mxHost("multi")
	primary := fmt.Sprintf("mail1.%s", domain)
	secondary := fmt.Sprintf("mail2.%s", domain)

	records := []testAccDNSRecord{
		{
			Type: "MX",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": primary,
			},
			IntAttrs: map[string]int{
				"preference": 10,
			},
		},
		{
			Type: "MX",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": secondary,
			},
			IntAttrs: map[string]int{
				"preference": 20,
			},
		},
	}

	resourceName := "spaceship_dns_records.test"

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
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "MX", host),
	})
}

// Verifies that exchange matching is case-insensitive end-to-end: applying a
// mixed-case exchange and re-applying produces an empty plan regardless of
// what casing the API persists.
func TestAccDNSRecords_mxRecordExchangeCaseInsensitive(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := mxHost("case")

	records := []testAccDNSRecord{
		{
			Type: "MX",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": fmt.Sprintf("Mail.%s", domain),
			},
			IntAttrs: map[string]int{
				"preference": 10,
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
				Config: testAccDNSRecordsConfig(domain, records),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "MX", host),
	})
}

// Verifies that exchange="@" is rejected at plan time. The API returns 422
// for this value at apply time (empirically confirmed), so the client-layer
// validator short-circuits with a plan-time error to avoid a failed apply.
func TestAccDNSRecords_mxRecordApexExchangeFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "MX",
			Name: mxHost("apexexch"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": "@",
			},
			IntAttrs: map[string]int{
				"preference": 10,
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Exchange Value"),
			},
		},
	})
}

// Verifies that exchange="*" is rejected at plan time. Same rationale as
// the apex test: API returns 422 at apply time, so plan-time rejection is
// the correct UX.
func TestAccDNSRecords_mxRecordWildcardExchangeFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "MX",
			Name: mxHost("wildexch"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": "*",
			},
			IntAttrs: map[string]int{
				"preference": 10,
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Exchange Value"),
			},
		},
	})
}

// Verifies that TTL is actually updatable on an MX record. The API docs
// omit ttl from MxResourceRecord, but the UI console sets it — this test
// confirms empirically whether per-record TTL is honored. Creates with
// TTL=3600, updates to TTL=300, checks state reflects 300, then re-applies
// the same config with ExpectEmptyPlan to catch any server-side normalization.
//
// If this test fails with drift on the final step, the API is silently
// coercing MX TTL to a default and per-record TTL is effectively read-only.
func TestAccDNSRecords_mxRecordUpdateTTL(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := mxHost("ttlupdate")
	exchange := fmt.Sprintf("mail.%s", domain)

	initialRecords := []testAccDNSRecord{
		{
			Type: "MX",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": exchange,
			},
			IntAttrs: map[string]int{
				"preference": 10,
			},
		},
	}

	updatedRecords := []testAccDNSRecord{
		{
			Type: "MX",
			Name: host,
			TTL:  intPointer(300),
			StringAttrs: map[string]string{
				"exchange": exchange,
			},
			IntAttrs: map[string]int{
				"preference": 10,
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
					resource.TestCheckResourceAttr(resourceName, "records.0.ttl", "3600"),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, updatedRecords),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.ttl", "300"),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, updatedRecords),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "MX", host),
	})
}

// Verifies that preference=0 (highest priority per RFC 5321) round-trips
// without being silently coerced or treated as a missing value.
func TestAccDNSRecords_mxRecordZeroPreference(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := mxHost("zeropref")
	exchange := fmt.Sprintf("mail.%s", domain)

	records := []testAccDNSRecord{
		{
			Type: "MX",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": exchange,
			},
			IntAttrs: map[string]int{
				"preference": 0,
			},
		},
	}

	resourceName := "spaceship_dns_records.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, records),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.0.preference", "0"),
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
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "MX", host),
	})
}
