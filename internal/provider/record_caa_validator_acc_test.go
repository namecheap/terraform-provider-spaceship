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

// caaHost returns a unique record name per test so parallel/repeat runs
// against the same SPACESHIP_TEST_DOMAIN don't collide.
func caaHost(suffix string) string {
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	return fmt.Sprintf("%s-caa-%s", prefix, suffix)
}

// Verifies that a standard CAA record round-trips through create, import,
// and destroy.
func TestAccDNSRecords_caaRecord(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := caaHost("basic")

	records := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"tag":   "issue",
				"value": "letsencrypt.org",
			},
		},
	}

	resourceName := "spaceship_dns_records.test"
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "records.0.type", "CAA"),
		resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.0.flag", "0"),
		resource.TestCheckResourceAttr(resourceName, "records.0.tag", "issue"),
		resource.TestCheckResourceAttr(resourceName, "records.0.value", "letsencrypt.org"),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CAA", host),
	})
}

// Verifies that an unknown tag value is rejected at plan time.
func TestAccDNSRecords_caaRecordInvalidTagFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: caaHost("badtag"),
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"tag":   "not-a-tag",
				"value": "letsencrypt.org",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Tag Value"),
			},
		},
	})
}

// Verifies that a value containing non-printable characters is rejected at plan time.
func TestAccDNSRecords_caaRecordInvalidValueFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: caaHost("badvalue"),
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"tag":   "issue",
				"value": "letsencrypt.org\t",
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

// Verifies that a missing tag field is rejected at plan time.
func TestAccDNSRecords_caaRecordMissingTagFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: caaHost("notag"),
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"value": "letsencrypt.org",
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

// Verifies that a missing value field is rejected at plan time.
func TestAccDNSRecords_caaRecordMissingValueFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: caaHost("noval"),
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"tag": "issue",
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

// Verifies that a missing flag field is rejected at plan time.
func TestAccDNSRecords_caaRecordMissingFlagFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: caaHost("noflag"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"tag":   "issue",
				"value": "letsencrypt.org",
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

// Verifies that a CAA record created directly via the API can be imported
// into Terraform and that a subsequent plan is empty.
func TestAccDNSRecords_caaRecordImportPreExisting(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := caaHost("import")
	flag := 0

	preExistingRecord := client.DNSRecord{
		Type:  "CAA",
		Name:  host,
		TTL:   3600,
		Flag:  &flag,
		Tag:   "issue",
		Value: "letsencrypt.org",
	}

	// Seed the record via the API before Terraform runs.
	preConfig := func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to create test client: %v", err)
		}
		if err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Fatalf("failed to pre-seed CAA record: %v", err)
		}
	}

	// Clean up the pre-seeded record after the test. Log failures so
	// the next run doesn't silently collide on the deterministic host.
	t.Cleanup(func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Logf("cleanup: failed to create test client: %v", err)
			return
		}
		if err := testClient.DeleteDNSRecords(context.Background(), domain, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Logf("cleanup: failed to delete pre-seeded CAA record: %v", err)
		}
	})

	// Terraform config matches the pre-seeded record exactly.
	records := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"tag":   "issue",
				"value": "letsencrypt.org",
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
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "CAA"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.flag", "0"),
					resource.TestCheckResourceAttr(resourceName, "records.0.tag", "issue"),
					resource.TestCheckResourceAttr(resourceName, "records.0.value", "letsencrypt.org"),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CAA", host),
	})
}

// Verifies that flag=128 (the critical bit) round-trips through create and
// read without being silently coerced to 0.
func TestAccDNSRecords_caaRecordCriticalFlag(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := caaHost("critical")

	records := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 128,
			},
			StringAttrs: map[string]string{
				"tag":   "issue",
				"value": "letsencrypt.org",
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
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.flag", "128"),
					resource.TestCheckResourceAttr(resourceName, "records.0.tag", "issue"),
					resource.TestCheckResourceAttr(resourceName, "records.0.value", "letsencrypt.org"),
				),
			},
			{
				// Re-apply the same config; plan must be empty. If the API
				// stores flag differently than we sent, this will show drift.
				Config: testAccDNSRecordsConfig(domain, records),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CAA", host),
	})
}

// Verifies that changing the value of an existing CAA record works. The diff
// signature keys on flag+tag+value, so a value change is a delete+upsert, not
// an in-place update — this exercises both paths.
func TestAccDNSRecords_caaRecordUpdateValue(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := caaHost("update")

	initialRecords := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"tag":   "issue",
				"value": "letsencrypt.org",
			},
		},
	}

	updatedRecords := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"tag":   "issue",
				"value": "sectigo.com",
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
					resource.TestCheckResourceAttr(resourceName, "records.0.value", "letsencrypt.org"),
				),
			},
			{
				// Same host, new value → old record should be deleted and new
				// one upserted. Final state has only the new record.
				Config: testAccDNSRecordsConfig(domain, updatedRecords),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.value", "sectigo.com"),
				),
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CAA", host),
	})
}

// Verifies that value matching is case-insensitive end-to-end: applying a
// mixed-case value and re-applying should produce an empty plan, regardless
// of what casing the API persists.
func TestAccDNSRecords_caaRecordValueCaseInsensitive(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := caaHost("case")

	records := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"tag":   "issue",
				"value": "LetsEncrypt.ORG",
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CAA", host),
	})
}

// Verifies that three CAA records on the same host (one per tag) coexist.
// Real-world CAA config typically pairs `issue` + `issuewild` + `iodef` for
// the zone apex. The diff logic must treat them as distinct records even
// though name and type match.
func TestAccDNSRecords_caaRecordMultipleTagsSameHost(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := caaHost("multi")

	records := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"tag":   "issue",
				"value": "letsencrypt.org",
			},
		},
		{
			Type: "CAA",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"tag":   "issuewild",
				"value": "letsencrypt.org",
			},
		},
		{
			Type: "CAA",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"tag":   "iodef",
				"value": "mailto:security@example.com",
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CAA", host),
	})
}

// Verifies that an `iodef` CAA record with a `mailto:` URI (non-hostname
// value containing `:` and `@`) round-trips cleanly.
func TestAccDNSRecords_caaRecordIodefMailto(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := caaHost("iodef")
	value := "mailto:security@example.com"

	records := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"tag":   "iodef",
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
					resource.TestCheckResourceAttr(resourceName, "records.0.tag", "iodef"),
					resource.TestCheckResourceAttr(resourceName, "records.0.value", value),
				),
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CAA", host),
	})
}

// Verifies that a bare `;` value with tag `issue` — the RFC 8659 "no CA may
// issue" form — is accepted and round-trips cleanly.
func TestAccDNSRecords_caaRecordDisallowAll(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := caaHost("disallow")

	records := []testAccDNSRecord{
		{
			Type: "CAA",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"flag": 0,
			},
			StringAttrs: map[string]string{
				"tag":   "issue",
				"value": ";",
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
					resource.TestCheckResourceAttr(resourceName, "records.0.value", ";"),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "CAA", host),
	})
}
