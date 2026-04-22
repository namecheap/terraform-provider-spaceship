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

// Verifies plan-time rejections for HTTPS: port without scheme, non-_https
// scheme, target_name="@" (apex), and target_name="*" (wildcard).
func TestAccDNSRecords_httpsValidation(t *testing.T) {
	testAccPreCheck(t)
	domain := testAccDomainValue()

	svcTarget := fmt.Sprintf("svc.%s", domain)

	cases := []struct {
		suffix      string
		stringAttrs map[string]string
		errRegex    string
	}{
		{
			suffix: "portnoscheme",
			stringAttrs: map[string]string{
				"target_name": svcTarget,
				"port":        "_443",
			},
			errRegex: "Invalid Scheme Value",
		},
		{
			suffix: "badscheme",
			stringAttrs: map[string]string{
				"target_name": svcTarget,
				"port":        "_443",
				"scheme":      "_http",
			},
			errRegex: "Invalid Scheme Value",
		},
		{
			suffix:      "apextarget",
			stringAttrs: map[string]string{"target_name": "@"},
			errRegex:    "Invalid TargetName Value",
		},
		{
			suffix:      "wildtarget",
			stringAttrs: map[string]string{"target_name": "*"},
			errRegex:    "Invalid TargetName Value",
		},
	}

	steps := make([]resource.TestStep, 0, len(cases))
	for _, tc := range cases {
		records := []testAccDNSRecord{
			{
				Type:        "HTTPS",
				Name:        httpsHost(tc.suffix),
				TTL:         intPointer(3600),
				IntAttrs:    map[string]int{"svc_priority": 1},
				StringAttrs: tc.stringAttrs,
			},
		}
		steps = append(steps, resource.TestStep{
			Config:      testAccDNSRecordsConfig(domain, records),
			ExpectError: regexp.MustCompile(tc.errRegex),
		})
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    steps,
	})
}

// Verifies HTTPS ServiceMode lifecycle: create, update target_name
// (delete+upsert path — target is in the diff signature), update TTL (upsert
// only — TTL is not in the signature), empty re-plan, and import.
func TestAccDNSRecords_httpsCreateAndUpdate(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("lifecycle")
	resourceName := "spaceship_dns_records.test"

	mkRecord := func(targetHost string, ttl int) []testAccDNSRecord {
		return []testAccDNSRecord{{
			Type:     "HTTPS",
			Name:     host,
			TTL:      intPointer(ttl),
			IntAttrs: map[string]int{"svc_priority": 1},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("%s.%s", targetHost, domain),
				"svc_params":  "alpn=h2",
				"port":        "_443",
				"scheme":      "_https",
			},
		}}
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord("svc1", 3600)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "HTTPS"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.svc_priority", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.target_name", fmt.Sprintf("svc1.%s", domain)),
					resource.TestCheckResourceAttr(resourceName, "records.0.port", "_443"),
					resource.TestCheckResourceAttr(resourceName, "records.0.scheme", "_https"),
					resource.TestCheckResourceAttr(resourceName, "records.0.ttl", "3600"),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord("svc2", 3600)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.target_name", fmt.Sprintf("svc2.%s", domain)),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord("svc2", 600)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.ttl", "600"),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord("svc2", 600)),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
	})
}

// Verifies AliasMode (svc_priority=0, target_name=".") stability and the
// transition AliasMode → ServiceMode, which is a delete+upsert because both
// svc_priority and target_name are in the diff signature.
func TestAccDNSRecords_httpsAliasModeLifecycle(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("alias")
	resourceName := "spaceship_dns_records.test"

	aliasMode := []testAccDNSRecord{{
		Type:        "HTTPS",
		Name:        host,
		TTL:         intPointer(3600),
		IntAttrs:    map[string]int{"svc_priority": 0},
		StringAttrs: map[string]string{"target_name": "."},
	}}

	serviceMode := []testAccDNSRecord{{
		Type:     "HTTPS",
		Name:     host,
		TTL:      intPointer(3600),
		IntAttrs: map[string]int{"svc_priority": 1},
		StringAttrs: map[string]string{
			"target_name": fmt.Sprintf("svc.%s", domain),
			"svc_params":  "alpn=h2",
		},
	}}

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
				Config: testAccDNSRecordsConfig(domain, aliasMode),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: testAccDNSRecordsConfig(domain, serviceMode),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.svc_priority", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.target_name", fmt.Sprintf("svc.%s", domain)),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, serviceMode),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
	})
}

// Verifies six independent edge scenarios coexisting in one config: apex
// name, boundary svc_priority=65535, wildcard port "*", high-range port
// _64999, mixed-case svc_params (case-insensitive matching), and multiple
// priorities on the same host.
func TestAccDNSRecords_httpsEdgeValues(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	maxPrioHost := httpsHost("maxprio")
	wildPortHost := httpsHost("wildport")
	highPortHost := httpsHost("highport")
	multiHost := httpsHost("multi")
	resourceName := "spaceship_dns_records.test"

	records := []testAccDNSRecord{
		{
			Type:     "HTTPS",
			Name:     "@",
			TTL:      intPointer(3600),
			IntAttrs: map[string]int{"svc_priority": 1},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2,h3",
			},
		},
		{
			Type:     "HTTPS",
			Name:     maxPrioHost,
			TTL:      intPointer(3600),
			IntAttrs: map[string]int{"svc_priority": 65535},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "ALPN=H2", // mixed-case, verifies case-insensitive matching on re-plan
			},
		},
		{
			Type:     "HTTPS",
			Name:     wildPortHost,
			TTL:      intPointer(3600),
			IntAttrs: map[string]int{"svc_priority": 1},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2",
				"port":        "*",
				"scheme":      "_https",
			},
		},
		{
			Type:     "HTTPS",
			Name:     highPortHost,
			TTL:      intPointer(3600),
			IntAttrs: map[string]int{"svc_priority": 1},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2",
				"port":        "_64999",
				"scheme":      "_https",
			},
		},
		{
			Type:     "HTTPS",
			Name:     multiHost,
			TTL:      intPointer(3600),
			IntAttrs: map[string]int{"svc_priority": 1},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc1.%s", domain),
				"svc_params":  "alpn=h2",
			},
		},
		{
			Type:     "HTTPS",
			Name:     multiHost,
			TTL:      intPointer(3600),
			IntAttrs: map[string]int{"svc_priority": 2},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc2.%s", domain),
				"svc_params":  "alpn=h3",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, records),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "6"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", "@"),
					resource.TestCheckResourceAttr(resourceName, "records.1.svc_priority", "65535"),
					resource.TestCheckResourceAttr(resourceName, "records.2.port", "*"),
					resource.TestCheckResourceAttr(resourceName, "records.3.port", "_64999"),
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
			testAccCheckDNSRecordAbsent(domain, "HTTPS", "@"),
			testAccCheckDNSRecordAbsent(domain, "HTTPS", maxPrioHost),
			testAccCheckDNSRecordAbsent(domain, "HTTPS", wildPortHost),
			testAccCheckDNSRecordAbsent(domain, "HTTPS", highPortHost),
			testAccCheckDNSRecordAbsent(domain, "HTTPS", multiHost),
		),
	})
}

// Verifies HTTPS coexists with an A record on the same host (type is part
// of recordKey so same-name cross-type records must not collide), and that
// removing HTTPS from config while keeping A exercises the Update delete
// path distinct from resource destroy.
func TestAccDNSRecords_httpsRecordComposition(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := httpsHost("compose")
	resourceName := "spaceship_dns_records.test"

	httpsRecord := testAccDNSRecord{
		Type:     "HTTPS",
		Name:     host,
		TTL:      intPointer(3600),
		IntAttrs: map[string]int{"svc_priority": 1},
		StringAttrs: map[string]string{
			"target_name": fmt.Sprintf("svc.%s", domain),
			"svc_params":  "alpn=h2",
		},
	}
	aRecord := testAccDNSRecord{
		Type:        "A",
		Name:        host,
		TTL:         intPointer(3600),
		StringAttrs: map[string]string{"address": "1.2.3.4"},
	}

	withBoth := []testAccDNSRecord{httpsRecord, aRecord}
	onlyA := []testAccDNSRecord{aRecord}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, withBoth),
				Check:  resource.TestCheckResourceAttr(resourceName, "records.#", "2"),
			},
			{
				Config: testAccDNSRecordsConfig(domain, withBoth),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: testAccDNSRecordsConfig(domain, onlyA),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "A"),
					testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
				),
			},
		},
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
			testAccCheckDNSRecordAbsent(domain, "A", host),
		),
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

	preConfig := func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to create test client: %v", err)
		}
		if err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Fatalf("failed to pre-seed HTTPS record: %v", err)
		}
	}

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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "HTTPS", host),
	})
}
