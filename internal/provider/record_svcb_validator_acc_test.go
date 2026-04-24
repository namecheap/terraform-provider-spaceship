package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// svcbHost returns a unique record name per test so parallel/repeat runs
// against the same SPACESHIP_TEST_DOMAIN don't collide.
func svcbHost(suffix string) string {
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	return fmt.Sprintf("%s-svcb-%s", prefix, suffix)
}

// Verifies plan-time rejections for SVCB: target_name="@" (apex),
// target_name="*" (wildcard), port "_0" (schema-pattern passes but range
// fails), svc_priority out of uint16 range, and port set without scheme
// (the API returns 422 for this, so we reject at plan time).
func TestAccDNSRecords_svcbValidation(t *testing.T) {
	testAccPreCheck(t)
	domain := testAccDomainValue()

	svcTarget := fmt.Sprintf("svc.%s", domain)

	cases := []struct {
		suffix      string
		stringAttrs map[string]string
		intAttrs    map[string]int
		errRegex    string
	}{
		{
			suffix:      "apextarget",
			stringAttrs: map[string]string{"target_name": "@"},
			intAttrs:    map[string]int{"svc_priority": 1},
			errRegex:    "Invalid TargetName Value",
		},
		{
			suffix:      "wildtarget",
			stringAttrs: map[string]string{"target_name": "*"},
			intAttrs:    map[string]int{"svc_priority": 1},
			errRegex:    "Invalid TargetName Value",
		},
		{
			suffix: "zeroport",
			stringAttrs: map[string]string{
				"target_name": svcTarget,
				"port":        "_0",
				"scheme":      "_tcp",
			},
			intAttrs: map[string]int{"svc_priority": 1},
			errRegex: "Invalid Port Value",
		},
		{
			suffix:      "priorityoutofrange",
			stringAttrs: map[string]string{"target_name": svcTarget},
			intAttrs:    map[string]int{"svc_priority": 70000},
			errRegex:    "Invalid SvcPriority Value",
		},
		{
			suffix: "portnoscheme",
			stringAttrs: map[string]string{
				"target_name": svcTarget,
				"port":        "_443",
			},
			intAttrs: map[string]int{"svc_priority": 1},
			errRegex: "Invalid Scheme Value",
		},
	}

	steps := make([]resource.TestStep, 0, len(cases))
	for _, tc := range cases {
		records := []testAccDNSRecord{
			{
				Type:        "SVCB",
				Name:        svcbHost(tc.suffix),
				TTL:         intPointer(3600),
				IntAttrs:    tc.intAttrs,
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

// Verifies SVCB ServiceMode lifecycle with a non-HTTPS scheme (_tcp): create,
// update target_name (delete+upsert — target is in the diff signature),
// update TTL (upsert only — TTL is not in the signature), empty re-plan,
// and import.
func TestAccDNSRecords_svcbCreateAndUpdate(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := svcbHost("lifecycle")
	resourceName := "spaceship_dns_records.test"

	mkRecord := func(targetHost string, ttl int) []testAccDNSRecord {
		return []testAccDNSRecord{{
			Type:     "SVCB",
			Name:     host,
			TTL:      intPointer(ttl),
			IntAttrs: map[string]int{"svc_priority": 1},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("%s.%s", targetHost, domain),
				"svc_params":  "alpn=h2",
				"port":        "_443",
				"scheme":      "_tcp",
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
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "SVCB"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.svc_priority", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.target_name", fmt.Sprintf("svc1.%s", domain)),
					resource.TestCheckResourceAttr(resourceName, "records.0.port", "_443"),
					resource.TestCheckResourceAttr(resourceName, "records.0.scheme", "_tcp"),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "SVCB", host),
	})
}

// Verifies AliasMode (svc_priority=0, target_name=".") stability and the
// transition AliasMode → ServiceMode, which is a delete+upsert because both
// svc_priority and target_name are in the diff signature.
func TestAccDNSRecords_svcbAliasModeLifecycle(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := svcbHost("alias")
	resourceName := "spaceship_dns_records.test"

	aliasMode := []testAccDNSRecord{{
		Type:        "SVCB",
		Name:        host,
		TTL:         intPointer(3600),
		IntAttrs:    map[string]int{"svc_priority": 0},
		StringAttrs: map[string]string{"target_name": "."},
	}}

	serviceMode := []testAccDNSRecord{{
		Type:     "SVCB",
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "SVCB", host),
	})
}

// Verifies SVCB edge scenarios coexisting in one config: apex name, boundary
// svc_priority=65535, wildcard port "*", non-HTTPS scheme "_udp", mixed-case
// svc_params (case-insensitive matching), and multiple priorities on the
// same host.
func TestAccDNSRecords_svcbEdgeValues(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	maxPrioHost := svcbHost("maxprio")
	wildPortHost := svcbHost("wildport")
	udpSchemeHost := svcbHost("udpscheme")
	multiHost := svcbHost("multi")
	resourceName := "spaceship_dns_records.test"

	records := []testAccDNSRecord{
		{
			Type:     "SVCB",
			Name:     "@",
			TTL:      intPointer(3600),
			IntAttrs: map[string]int{"svc_priority": 1},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2,h3",
			},
		},
		{
			Type:     "SVCB",
			Name:     maxPrioHost,
			TTL:      intPointer(3600),
			IntAttrs: map[string]int{"svc_priority": 65535},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "ALPN=H2", // mixed-case, verifies case-insensitive matching on re-plan
			},
		},
		{
			Type:     "SVCB",
			Name:     wildPortHost,
			TTL:      intPointer(3600),
			IntAttrs: map[string]int{"svc_priority": 1},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2",
				"port":        "*",
				"scheme":      "_tcp",
			},
		},
		{
			Type:     "SVCB",
			Name:     udpSchemeHost,
			TTL:      intPointer(3600),
			IntAttrs: map[string]int{"svc_priority": 1},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc.%s", domain),
				"svc_params":  "alpn=h2",
				"port":        "_8443",
				"scheme":      "_udp",
			},
		},
		{
			Type:     "SVCB",
			Name:     multiHost,
			TTL:      intPointer(3600),
			IntAttrs: map[string]int{"svc_priority": 1},
			StringAttrs: map[string]string{
				"target_name": fmt.Sprintf("svc1.%s", domain),
				"svc_params":  "alpn=h2",
			},
		},
		{
			Type:     "SVCB",
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
					resource.TestCheckResourceAttr(resourceName, "records.3.scheme", "_udp"),
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
			testAccCheckDNSRecordAbsent(domain, "SVCB", "@"),
			testAccCheckDNSRecordAbsent(domain, "SVCB", maxPrioHost),
			testAccCheckDNSRecordAbsent(domain, "SVCB", wildPortHost),
			testAccCheckDNSRecordAbsent(domain, "SVCB", udpSchemeHost),
			testAccCheckDNSRecordAbsent(domain, "SVCB", multiHost),
		),
	})
}
