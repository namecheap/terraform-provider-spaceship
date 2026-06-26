package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/namecheap/go-spaceship-sdk/client"
)

// txtHost returns a unique record name per test so parallel/repeat runs
// against the same SPACESHIP_TEST_DOMAIN don't collide.
func txtHost(suffix string) string {
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	return fmt.Sprintf("%s-txt-%s", prefix, suffix)
}

// Verifies that a standard TXT record round-trips through create, import,
// and destroy.
func TestAccDNSRecords_txtRecord(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := txtHost("basic")
	value := "v=spf1 a mx -all"

	records := []testAccDNSRecord{
		{
			Type: "TXT",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": value,
			},
		},
	}

	resourceName := "spaceship_dns_records.test"
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "records.0.type", "TXT"),
		resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.0.value", value),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "TXT", host),
	})
}

// Verifies that a whitespace-only value is rejected at plan time.
// Live-API probing showed the API itself returns "Value field is required"
// for whitespace-only values despite the spec listing minLength=1; the
// validator pre-empts that 422 with a plan-time error.
func TestAccDNSRecords_txtRecordWhitespaceOnlyValueFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "TXT",
			Name: txtHost("ws"),
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": " \t\n",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Value"),
			},
		},
	})
}

// Verifies that a value with leading/trailing whitespace round-trips intact
// — the API accepts these as long as the value is not entirely whitespace,
// so the validator must not strip or reject them.
func TestAccDNSRecords_txtRecordLeadingTrailingWhitespace(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := txtHost("trim")
	value := " v=spf1 -all\n"

	records := []testAccDNSRecord{
		{
			Type: "TXT",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": value,
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
					resource.TestCheckResourceAttr(resourceName, "records.0.value", value),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "TXT", host),
	})
}

// Verifies that an empty value is rejected at plan time.
func TestAccDNSRecords_txtRecordEmptyValueFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "TXT",
			Name: txtHost("empty"),
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": "",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Value"),
			},
		},
	})
}

// Verifies that a missing value field is rejected at plan time.
func TestAccDNSRecords_txtRecordMissingValueFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "TXT",
			Name: txtHost("noval"),
			TTL:  intPtr(3600),
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

// Verifies that changing the value of an existing TXT record works. The diff
// signature keys on type+name+value, so a value change is a delete+upsert,
// not an in-place update — this exercises both paths.
func TestAccDNSRecords_txtRecordUpdateValue(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := txtHost("update")

	initialRecords := []testAccDNSRecord{
		{
			Type: "TXT",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": "v=spf1 a mx -all",
			},
		},
	}

	updatedRecords := []testAccDNSRecord{
		{
			Type: "TXT",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": "v=spf1 include:_spf.example.com -all",
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
					resource.TestCheckResourceAttr(resourceName, "records.0.value", "v=spf1 a mx -all"),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, updatedRecords),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.value", "v=spf1 include:_spf.example.com -all"),
				),
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "TXT", host),
	})
}

// Verifies that TXT values are matched case-sensitively. Two records on the
// same host whose values differ only in case must coexist as distinct
// records — for any other record type these would collide as duplicates.
func TestAccDNSRecords_txtRecordValueCaseSensitive(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := txtHost("case")

	records := []testAccDNSRecord{
		{
			Type: "TXT",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": "MixedCaseToken",
			},
		},
		{
			Type: "TXT",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": "mixedcasetoken",
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
				// Re-apply the same config; plan must be empty. This proves
				// the upsert+read+diff cycle preserves case end-to-end.
				Config: testAccDNSRecordsConfig(domain, records),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "TXT", host),
	})
}

// Verifies that multiple TXT records with distinct values coexist on the
// same host — typical for SPF + domain verification + DKIM combinations.
func TestAccDNSRecords_txtRecordMultipleValuesSameHost(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := txtHost("multi")

	records := []testAccDNSRecord{
		{
			Type: "TXT",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": "v=spf1 a mx -all",
			},
		},
		{
			Type: "TXT",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": "google-site-verification=abc123def456",
			},
		},
		{
			Type: "TXT",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": "facebook-domain-verification=xyz789",
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
					resource.TestCheckResourceAttr(resourceName, "records.#", "3"),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "TXT", host),
	})
}

// Verifies that a value longer than 255 bytes — beyond a single DNS
// character-string and into the multi-string territory at the wire level —
// round-trips intact through the provider.
func TestAccDNSRecords_txtRecordLongValue(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := txtHost("long")

	// ~600-byte DKIM-like value: well past 255, well under 65535.
	value := "v=DKIM1; k=rsa; p=" + strings.Repeat("A", 600)

	records := []testAccDNSRecord{
		{
			Type: "TXT",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": value,
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
					resource.TestCheckResourceAttr(resourceName, "records.0.value", value),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "TXT", host),
	})
}

// Verifies that a DKIM-style value containing semicolons, equals, and
// embedded spaces round-trips without escaping or normalization changes.
func TestAccDNSRecords_txtRecordSpecialChars(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := txtHost("special")
	value := "v=DKIM1; k=rsa; t=s; p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQ"

	records := []testAccDNSRecord{
		{
			Type: "TXT",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": value,
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
					resource.TestCheckResourceAttr(resourceName, "records.0.value", value),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "TXT", host),
	})
}

// Verifies that a TXT record created directly via the API can be imported
// into Terraform and that a subsequent plan is empty.
func TestAccDNSRecords_txtRecordImportPreExisting(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := txtHost("import")
	value := "v=spf1 -all"

	preExistingRecord := client.DNSRecord{
		Type:  "TXT",
		Name:  host,
		TTL:   3600,
		Value: value,
	}

	preConfig := func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to create test client: %v", err)
		}
		if err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Fatalf("failed to pre-seed TXT record: %v", err)
		}
	}

	t.Cleanup(func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Logf("cleanup: failed to create test client: %v", err)
			return
		}
		if err := testClient.DeleteDNSRecords(context.Background(), domain, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Logf("cleanup: failed to delete pre-seeded TXT record: %v", err)
		}
	})

	records := []testAccDNSRecord{
		{
			Type: "TXT",
			Name: host,
			TTL:  intPtr(3600),
			StringAttrs: map[string]string{
				"value": value,
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
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "TXT"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.value", value),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "TXT", host),
	})
}
