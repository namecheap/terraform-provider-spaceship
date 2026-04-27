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

// Two distinct 64-char lowercase hex digests (SHA-256 sized) used as
// known-valid associationData payloads in update scenarios.
const (
	tlsaAssocA = "7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069"
	tlsaAssocB = "0e3f4cb09b9b3df0b71b8a8a2e1cba0a4c2b8f8e9d3c6a1b5e7f0d2c4a6b8e0f"
)

// tlsaHost returns a unique record name per test so parallel/repeat runs
// against the same SPACESHIP_TEST_DOMAIN don't collide.
func tlsaHost(suffix string) string {
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	return fmt.Sprintf("%s-tlsa-%s", prefix, suffix)
}

// Verifies plan-time rejections for TLSA: bad port format, bad protocol,
// usage out of range, uppercase association_data (rejected by the client
// validator even though the schema regex would accept it), and missing
// required fields.
func TestAccDNSRecords_tlsaValidation(t *testing.T) {
	testAccPreCheck(t)
	domain := testAccDomainValue()

	cases := []struct {
		suffix      string
		stringAttrs map[string]string
		intAttrs    map[string]int
		errRegex    string
	}{
		{
			suffix: "badport",
			stringAttrs: map[string]string{
				"port":             "443",
				"protocol":         "_tcp",
				"association_data": tlsaAssocA,
			},
			intAttrs: map[string]int{"usage": 2, "selector": 1, "matching": 1},
			errRegex: "Invalid Port Value",
		},
		{
			suffix: "badprotocol",
			stringAttrs: map[string]string{
				"port":             "_443",
				"protocol":         "tcp",
				"association_data": tlsaAssocA,
			},
			intAttrs: map[string]int{"usage": 2, "selector": 1, "matching": 1},
			errRegex: "Invalid Protocol Value",
		},
		{
			suffix: "uppercaseassoc",
			stringAttrs: map[string]string{
				"port":             "_443",
				"protocol":         "_tcp",
				"association_data": "7F83B1657FF1FC53B92DC18148A1D65DFC2D4B1FA3D677284ADDD200126D9069",
			},
			intAttrs: map[string]int{"usage": 2, "selector": 1, "matching": 1},
			errRegex: "Invalid AssociationData Value",
		},
		{
			suffix: "missingport",
			stringAttrs: map[string]string{
				"protocol":         "_tcp",
				"association_data": tlsaAssocA,
			},
			intAttrs: map[string]int{"usage": 2, "selector": 1, "matching": 1},
			errRegex: "Missing Required Field",
		},
		{
			suffix: "missingassoc",
			stringAttrs: map[string]string{
				"port":     "_443",
				"protocol": "_tcp",
			},
			intAttrs: map[string]int{"usage": 2, "selector": 1, "matching": 1},
			errRegex: "Missing Required Field",
		},
	}

	steps := make([]resource.TestStep, 0, len(cases))
	for _, tc := range cases {
		records := []testAccDNSRecord{
			{
				Type:        "TLSA",
				Name:        tlsaHost(tc.suffix),
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

// Verifies TLSA lifecycle: create, update association_data (delete+upsert
// since it is part of the diff signature), update TTL (upsert only — TTL is
// not in the signature), empty re-plan, and import.
func TestAccDNSRecords_tlsaCreateAndUpdate(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := tlsaHost("lifecycle")
	resourceName := "spaceship_dns_records.test"

	mkRecord := func(assoc string, ttl int) []testAccDNSRecord {
		return []testAccDNSRecord{{
			Type: "TLSA",
			Name: host,
			TTL:  intPointer(ttl),
			StringAttrs: map[string]string{
				"port":             "_443",
				"protocol":         "_tcp",
				"association_data": assoc,
			},
			IntAttrs: map[string]int{"usage": 2, "selector": 1, "matching": 1},
		}}
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord(tlsaAssocA, 3600)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "TLSA"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
					resource.TestCheckResourceAttr(resourceName, "records.0.port", "_443"),
					resource.TestCheckResourceAttr(resourceName, "records.0.protocol", "_tcp"),
					resource.TestCheckResourceAttr(resourceName, "records.0.usage", "2"),
					resource.TestCheckResourceAttr(resourceName, "records.0.selector", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.matching", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.association_data", tlsaAssocA),
					resource.TestCheckResourceAttr(resourceName, "records.0.ttl", "3600"),
				),
			},
			{
				// association_data is in the diff signature: this should
				// delete the old record and upsert a new one.
				Config: testAccDNSRecordsConfig(domain, mkRecord(tlsaAssocB, 3600)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.association_data", tlsaAssocB),
				),
			},
			{
				// TTL is not in the signature: upsert-only path.
				Config: testAccDNSRecordsConfig(domain, mkRecord(tlsaAssocB, 600)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.ttl", "600"),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, mkRecord(tlsaAssocB, 600)),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "TLSA", host),
	})
}

// Verifies that a TLSA record created directly via the API can be adopted by
// a matching Terraform config and that a subsequent plan is empty.
func TestAccDNSRecords_tlsaImportPreExisting(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := tlsaHost("import")
	usage, selector, matching := 2, 1, 1

	preExistingRecord := client.DNSRecord{
		Type:            "TLSA",
		Name:            host,
		TTL:             3600,
		Port:            client.NewStringPortValue("_443"),
		Protocol:        "_tcp",
		Usage:           &usage,
		Selector:        &selector,
		Matching:        &matching,
		AssociationData: tlsaAssocA,
	}

	preConfig := func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to create test client: %v", err)
		}
		if err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Fatalf("failed to pre-seed TLSA record: %v", err)
		}
	}

	t.Cleanup(func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Logf("cleanup: failed to create test client: %v", err)
			return
		}
		if err := testClient.DeleteDNSRecords(context.Background(), domain, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Logf("cleanup: failed to delete pre-seeded TLSA record: %v", err)
		}
	})

	records := []testAccDNSRecord{
		{
			Type: "TLSA",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"port":             "_443",
				"protocol":         "_tcp",
				"association_data": tlsaAssocA,
			},
			IntAttrs: map[string]int{"usage": 2, "selector": 1, "matching": 1},
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
					resource.TestCheckResourceAttr(resourceName, "records.0.association_data", tlsaAssocA),
				),
			},
			{
				Config: testAccDNSRecordsConfig(domain, records),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "TLSA", host),
	})
}

// Verifies that multiple TLSA records on the same host coexist when their
// (port, protocol, usage, selector, matching, association_data) signatures
// differ. Real-world TLSA setups commonly publish records for both `_443._tcp`
// and `_25._tcp` (HTTPS + SMTP) on the same name.
func TestAccDNSRecords_tlsaMultipleSameHost(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := tlsaHost("multi")
	resourceName := "spaceship_dns_records.test"

	records := []testAccDNSRecord{
		{
			Type: "TLSA",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"port":             "_443",
				"protocol":         "_tcp",
				"association_data": tlsaAssocA,
			},
			IntAttrs: map[string]int{"usage": 2, "selector": 1, "matching": 1},
		},
		{
			Type: "TLSA",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"port":             "_25",
				"protocol":         "_tcp",
				"association_data": tlsaAssocB,
			},
			IntAttrs: map[string]int{"usage": 3, "selector": 1, "matching": 1},
		},
		{
			Type: "TLSA",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"port":             "_443",
				"protocol":         "_tcp",
				"association_data": tlsaAssocB,
			},
			// same port+protocol as the first record but different
			// selector → must coexist.
			IntAttrs: map[string]int{"usage": 2, "selector": 0, "matching": 1},
		},
	}

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
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "TLSA", host),
	})
}

// Verifies TLSA boundary scenarios coexisting in one config: usage/selector/
// matching at the uint8 max (255), wildcard port "*", non-`_tcp` protocol
// (`_udp`), and the apex name.
func TestAccDNSRecords_tlsaBoundaryValues(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	maxValuesHost := tlsaHost("maxvals")
	wildPortHost := tlsaHost("wildport")
	udpProtoHost := tlsaHost("udpproto")
	resourceName := "spaceship_dns_records.test"

	records := []testAccDNSRecord{
		{
			Type: "TLSA",
			Name: "@",
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"port":             "_443",
				"protocol":         "_tcp",
				"association_data": tlsaAssocA,
			},
			IntAttrs: map[string]int{"usage": 2, "selector": 1, "matching": 1},
		},
		{
			Type: "TLSA",
			Name: maxValuesHost,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"port":             "_443",
				"protocol":         "_tcp",
				"association_data": tlsaAssocA,
			},
			IntAttrs: map[string]int{"usage": 255, "selector": 255, "matching": 255},
		},
		{
			Type: "TLSA",
			Name: wildPortHost,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"port":             "*",
				"protocol":         "_tcp",
				"association_data": tlsaAssocA,
			},
			IntAttrs: map[string]int{"usage": 2, "selector": 1, "matching": 1},
		},
		{
			Type: "TLSA",
			Name: udpProtoHost,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"port":             "_853",
				"protocol":         "_udp",
				"association_data": tlsaAssocA,
			},
			IntAttrs: map[string]int{"usage": 2, "selector": 1, "matching": 1},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, records),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", "@"),
					resource.TestCheckResourceAttr(resourceName, "records.1.usage", "255"),
					resource.TestCheckResourceAttr(resourceName, "records.1.selector", "255"),
					resource.TestCheckResourceAttr(resourceName, "records.1.matching", "255"),
					resource.TestCheckResourceAttr(resourceName, "records.2.port", "*"),
					resource.TestCheckResourceAttr(resourceName, "records.3.protocol", "_udp"),
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
			testAccCheckDNSRecordAbsent(domain, "TLSA", "@"),
			testAccCheckDNSRecordAbsent(domain, "TLSA", maxValuesHost),
			testAccCheckDNSRecordAbsent(domain, "TLSA", wildPortHost),
			testAccCheckDNSRecordAbsent(domain, "TLSA", udpProtoHost),
		),
	})
}
