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

// httpsHost returns a unique record name per test so parallel/repeat runs
// against the same SPACESHIP_TEST_DOMAIN don't collide.
func httpsHost(suffix string) string {
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	return fmt.Sprintf("%s-https-%s", prefix, suffix)
}

// Verifies that a ServiceMode HTTPS record with port and scheme round-trips
// through create, import, and destroy.
func TestAccDNSRecords_httpsRecord(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("basic")

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2",
				"port":        "_443",
				"scheme":      "_https",
			},
		},
	}

	resourceName := "spaceship_dns_records.test"
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "records.0.type", "HTTPS"),
		resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.0.svc_priority", "1"),
		resource.TestCheckResourceAttr(resourceName, "records.0.target_name", fmt.Sprintf("svc.%s", domain)),
		resource.TestCheckResourceAttr(resourceName, "records.0.svc_params", "alpn=h2"),
		resource.TestCheckResourceAttr(resourceName, "records.0.port", "_443"),
		resource.TestCheckResourceAttr(resourceName, "records.0.scheme", "_https"),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
	})
}

// Verifies an AliasMode HTTPS record (svc_priority=0, target_name=".")
// round-trips cleanly and a re-apply produces an empty plan.
func TestAccDNSRecords_httpsRecordAliasMode(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("alias")

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 0,
			},
			StringAttrs: map[string]string{
				"target_name": ".",
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
					resource.TestCheckResourceAttr(resourceName, "records.0.svc_priority", "0"),
					resource.TestCheckResourceAttr(resourceName, "records.0.target_name", "."),
				),
			},
			{
				// Re-apply must be a no-op — AliasMode with a dot target is
				// a common pattern and should not drift.
				Config: testAccDNSRecordsConfig(domain, records),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
	})
}

// Verifies that missing svc_priority is rejected at plan time.
func TestAccDNSRecords_httpsRecordMissingSvcPriorityFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: httpsHost("nopriority"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
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

// Verifies that missing target_name is rejected at plan time.
func TestAccDNSRecords_httpsRecordMissingTargetNameFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: httpsHost("notarget"),
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
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

// Verifies that setting port without scheme is rejected at plan time —
// HTTPS records require scheme="_https" whenever port is specified.
func TestAccDNSRecords_httpsRecordPortWithoutSchemeFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: httpsHost("noscheme"),
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"port":        "_443",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Scheme Value"),
			},
		},
	})
}

// Verifies that a non-'_https' scheme is rejected at plan time.
func TestAccDNSRecords_httpsRecordInvalidSchemeFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: httpsHost("badscheme"),
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"port":        "_443",
				"scheme":      "_http",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Scheme Value"),
			},
		},
	})
}

// Verifies that target_name="@" is rejected at plan time — the API requires
// an FQDN or the literal ".", not the apex placeholder.
func TestAccDNSRecords_httpsRecordInvalidTargetNameFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: httpsHost("badtarget"),
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": "@",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid TargetName Value"),
			},
		},
	})
}

// Verifies that an HTTPS record created directly via the API can be imported
// into Terraform and that a subsequent plan is empty.
func TestAccDNSRecords_httpsRecordImportPreExisting(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("import")
	priority := 1

	preExistingRecord := client.DNSRecord{
		Type:        "HTTPS",
		Name:        host,
		TTL:         3600,
		SvcPriority: &priority,
		TargetName:  fmt.Sprintf("svc.%s", domain),
		SvcParams:   "alpn=h2",
	}

	// Seed the record via the API before Terraform runs.
	preConfig := func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to create test client: %v", err)
		}
		if err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Fatalf("failed to pre-seed HTTPS record: %v", err)
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
			t.Logf("cleanup: failed to delete pre-seeded HTTPS record: %v", err)
		}
	})

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2",
			},
		},
	}

	resourceName := "spaceship_dns_records.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Step 1: Apply with the pre-existing record already on the
				// API. Terraform should adopt it without changes.
				PreConfig: preConfig,
				Config:    testAccDNSRecordsConfig(domain, records),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "HTTPS"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.svc_priority", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.target_name", fmt.Sprintf("svc.%s", domain)),
					resource.TestCheckResourceAttr(resourceName, "records.0.svc_params", "alpn=h2"),
				),
			},
			{
				// Step 2: Re-apply. Plan should be empty since state already
				// matches configuration and API.
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
	})
}

// Verifies that changing target_name works. The diff signature keys HTTPS
// on svc_priority+target_name+svc_params+port+scheme, so a target change
// is a delete+upsert, not an in-place update.
func TestAccDNSRecords_httpsRecordUpdateTargetName(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("update")

	initialRecords := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc1.%s", domain),
				"svc_params":  "alpn=h2",
			},
		},
	}

	updatedRecords := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc2.%s", domain),
				"svc_params":  "alpn=h2",
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
					resource.TestCheckResourceAttr(resourceName, "records.0.target_name", fmt.Sprintf("svc1.%s", domain)),
				),
			},
			{
				// Same host, new target → old record should be deleted and
				// new one upserted. Final state has only the new record.
				Config: testAccDNSRecordsConfig(domain, updatedRecords),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.target_name", fmt.Sprintf("svc2.%s", domain)),
				),
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
	})
}

// Verifies that svc_params matching is case-insensitive end-to-end: a
// mixed-case value should re-apply to an empty plan, regardless of how the
// API persists the casing.
func TestAccDNSRecords_httpsRecordSvcParamsCaseInsensitive(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("case")

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "ALPN=H2",
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
				// If the API normalizes casing server-side and returns
				// something different, this will surface as drift.
				Config: testAccDNSRecordsConfig(domain, records),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
	})
}

// Verifies that two HTTPS records on the same host with different
// svc_priority values coexist. Real-world HTTPS config often pairs a
// priority-0 AliasMode record with higher-priority ServiceMode alternatives.
func TestAccDNSRecords_httpsRecordMultiplePrioritiesSameHost(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("multi")

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc1.%s", domain),
				"svc_params":  "alpn=h2",
			},
		},
		{
			Type: "HTTPS",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 2,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc2.%s", domain),
				"svc_params":  "alpn=h3",
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
	})
}

// Verifies that svc_priority=65535 (uint16 max) round-trips without being
// silently coerced to a lower value.
func TestAccDNSRecords_httpsRecordBoundarySvcPriority(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("maxprio")

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 65535,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2",
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
					resource.TestCheckResourceAttr(resourceName, "records.0.svc_priority", "65535"),
				),
			},
			{
				// Re-apply with max priority must be a no-op.
				Config: testAccDNSRecordsConfig(domain, records),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
	})
}

// Verifies that port="*" (the wildcard port form) is accepted by the API
// and round-trips. The client spec lists "*" as a valid alternative to the
// "_NNNN" underscored port form.
func TestAccDNSRecords_httpsRecordWildcardPort(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("wildport")

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2",
				"port":        "*",
				"scheme":      "_https",
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
					resource.TestCheckResourceAttr(resourceName, "records.0.port", "*"),
					resource.TestCheckResourceAttr(resourceName, "records.0.scheme", "_https"),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
	})
}

// Verifies that an HTTPS record at the zone apex (name="@") round-trips.
// Apex HTTPS records are common in modern deployments (ECH, HTTP/3 hints).
func TestAccDNSRecords_httpsRecordApexName(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: "@",
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2,h3",
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
					resource.TestCheckResourceAttr(resourceName, "records.0.name", "@"),
					resource.TestCheckResourceAttr(resourceName, "records.0.svc_params", "alpn=h2,h3"),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", "@"),
	})
}

// Verifies transition from AliasMode (priority=0, target=".") to ServiceMode
// (priority=1, real FQDN). Since both svc_priority and target_name are in the
// diff signature, this forces a delete+upsert, not an in-place update.
func TestAccDNSRecords_httpsRecordAliasToServiceModeUpdate(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("modexchg")

	aliasMode := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 0,
			},
			StringAttrs: map[string]string{
				"target_name": ".",
			},
		},
	}

	serviceMode := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2",
			},
		},
	}

	resourceName := "spaceship_dns_records.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, aliasMode),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.0.svc_priority", "0"),
					resource.TestCheckResourceAttr(resourceName, "records.0.target_name", "."),
				),
			},
			{
				// AliasMode → ServiceMode: old record deleted, new one upserted.
				Config: testAccDNSRecordsConfig(domain, serviceMode),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.svc_priority", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.target_name", fmt.Sprintf("svc.%s", domain)),
				),
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
	})
}

// Verifies that changing only the TTL triggers an upsert without changing
// record identity. recordValueSignature excludes TTL, so the diff skip-check
// (signature match AND TTL match) must catch TTL-only differences.
func TestAccDNSRecords_httpsRecordTTLUpdate(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("ttl")

	base := testAccDNSRecord{
		Type: "HTTPS",
		Name: host,
		IntAttrs: map[string]int{
			"svc_priority": 1,
		},
		StringAttrs: map[string]string{
			"target_name": fmt.Sprintf("svc.%s", domain),
			"svc_params":  "alpn=h2",
		},
	}
	initial := base
	initial.TTL = intPointer(3600)
	updated := base
	updated.TTL = intPointer(600)

	resourceName := "spaceship_dns_records.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, []testAccDNSRecord{initial}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.0.ttl", "3600"),
				),
			},
			{
				// Same identity fields, only TTL differs → record is upserted
				// but count stays at 1. Re-applying must converge to empty.
				Config: testAccDNSRecordsConfig(domain, []testAccDNSRecord{updated}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.ttl", "600"),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, []testAccDNSRecord{updated}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
	})
}

// Verifies that an HTTPS record and an A record on the same host coexist.
// recordKey includes the record type, so same-name records of different
// types must not conflict in the diff logic.
func TestAccDNSRecords_httpsRecordCoexistsWithA(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("coexist")

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: host,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2",
			},
		},
		{
			Type: "A",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "1.2.3.4",
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
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
			testAccCheckDNSRecordAbsent(domain, "A", host),
		),
	})
}

// Verifies that removing an HTTPS record from the config (while leaving the
// resource in place) deletes it from the API. Exercises the Update delete
// path, which is distinct from resource destroy.
func TestAccDNSRecords_httpsRecordRemovedFromConfig(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	httpsName := httpsHost("removed")
	aName := httpsHost("kept")

	withHTTPS := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: httpsName,
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2",
			},
		},
		{
			Type: "A",
			Name: aName,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "1.2.3.4",
			},
		},
	}

	withoutHTTPS := []testAccDNSRecord{
		{
			Type: "A",
			Name: aName,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "1.2.3.4",
			},
		},
	}

	resourceName := "spaceship_dns_records.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, withHTTPS),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "2"),
				),
			},
			{
				// Remove HTTPS from config. The A record stays; HTTPS should
				// be deleted from the API.
				Config: testAccDNSRecordsConfig(domain, withoutHTTPS),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "A"),
					testAccCheckDNSRecordAbsent(domain, "HTTPS", httpsName),
				),
			},
		},
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckDNSRecordAbsent(domain, "HTTPS", httpsName),
			testAccCheckDNSRecordAbsent(domain, "A", aName),
		),
	})
}

// Verifies that target_name="*" is rejected at plan time. Mirrors the
// target_name="@" case but exercises the wildcard branch of the reject list.
func TestAccDNSRecords_httpsRecordWildcardTargetFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "HTTPS",
			Name: httpsHost("wildtarget"),
			TTL:  intPointer(3600),
			IntAttrs: map[string]int{
				"svc_priority": 1,
			},
			StringAttrs: map[string]string{
				"target_name": "*",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid TargetName Value"),
			},
		},
	})
}
