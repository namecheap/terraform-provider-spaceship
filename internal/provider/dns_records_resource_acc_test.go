package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccDNSRecords_basicTypes(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}

	resourceName := "spaceship_dns_records.test"

	records := []testAccDNSRecord{
		{
			Type: "A",
			Name: fmt.Sprintf("%s-a", prefix),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "198.51.100.10",
			},
		},
		{
			Type: "AAAA",
			Name: fmt.Sprintf("%s-aaaa", prefix),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "2001:db8::1",
			},
		},
		{
			Type: "CNAME",
			Name: fmt.Sprintf("%s-cname", prefix),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"cname": fmt.Sprintf("origin.%s", domain),
			},
		},
		{
			Type: "MX",
			Name: fmt.Sprintf("%s-mx", prefix),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"exchange": fmt.Sprintf("mail.%s", domain),
			},
			IntAttrs: map[string]int{
				"preference": 10,
			},
		},
		{
			Type: "TXT",
			Name: fmt.Sprintf("%s-txt", prefix),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"value": "v=spf1 a mx -all",
			},
		},
	}

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "records.#", fmt.Sprintf("%d", len(records))),
		resource.TestCheckResourceAttr(resourceName, "records.0.type", "A"),
		resource.TestCheckResourceAttr(resourceName, "records.0.name", fmt.Sprintf("%s-a", prefix)),
		resource.TestCheckResourceAttr(resourceName, "records.0.address", "198.51.100.10"),
		resource.TestCheckResourceAttr(resourceName, "records.1.type", "AAAA"),
		resource.TestCheckResourceAttr(resourceName, "records.1.address", "2001:db8::1"),
		resource.TestCheckResourceAttr(resourceName, "records.2.type", "CNAME"),
		resource.TestCheckResourceAttr(resourceName, "records.2.cname", fmt.Sprintf("origin.%s", domain)),
		resource.TestCheckResourceAttr(resourceName, "records.3.type", "MX"),
		resource.TestCheckResourceAttr(resourceName, "records.3.exchange", fmt.Sprintf("mail.%s", domain)),
		resource.TestCheckResourceAttr(resourceName, "records.3.preference", "10"),
		resource.TestCheckResourceAttr(resourceName, "records.4.type", "TXT"),
		resource.TestCheckResourceAttr(resourceName, "records.4.value", "v=spf1 a mx -all"),
	}

	destroyChecks := []resource.TestCheckFunc{
		testAccCheckDNSRecordAbsent(domain, "A", fmt.Sprintf("%s-a", prefix)),
		testAccCheckDNSRecordAbsent(domain, "AAAA", fmt.Sprintf("%s-aaaa", prefix)),
		testAccCheckDNSRecordAbsent(domain, "CNAME", fmt.Sprintf("%s-cname", prefix)),
		testAccCheckDNSRecordAbsent(domain, "MX", fmt.Sprintf("%s-mx", prefix)),
		testAccCheckDNSRecordAbsent(domain, "TXT", fmt.Sprintf("%s-txt", prefix)),
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
		CheckDestroy: resource.ComposeTestCheckFunc(destroyChecks...),
	})
}

func TestAccDNSRecords_preservesRecordOrder(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}

	resourceName := "spaceship_dns_records.test"
	host := fmt.Sprintf("%s-order", prefix)

	records := []testAccDNSRecord{
		{
			Type: "A",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "198.51.100.20",
			},
		},
		{
			Type: "TXT",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"value": "order-check",
			},
		},
		{
			Type: "AAAA",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "2001:db8::2",
			},
		},
	}

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "records.#", "3"),
		resource.TestCheckResourceAttr(resourceName, "records.0.type", "A"),
		resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.1.type", "TXT"),
		resource.TestCheckResourceAttr(resourceName, "records.1.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.1.value", "order-check"),
		resource.TestCheckResourceAttr(resourceName, "records.2.type", "AAAA"),
		resource.TestCheckResourceAttr(resourceName, "records.2.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.2.address", "2001:db8::2"),
	}

	destroyChecks := []resource.TestCheckFunc{
		testAccCheckDNSRecordAbsent(domain, "A", host),
		testAccCheckDNSRecordAbsent(domain, "TXT", host),
		testAccCheckDNSRecordAbsent(domain, "AAAA", host),
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
		CheckDestroy: resource.ComposeTestCheckFunc(destroyChecks...),
	})
}

func TestAccDNSRecords_removedFromConfiguration(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}

	resourceName := "spaceship_dns_records.test"
	host := fmt.Sprintf("%s-remove", prefix)

	records := []testAccDNSRecord{
		{
			Type: "A",
			Name: host,
			TTL:  intPointer(300),
			StringAttrs: map[string]string{
				"address": "198.51.100.30",
			},
		},
		{
			Type: "TXT",
			Name: host,
			TTL:  intPointer(300),
			StringAttrs: map[string]string{
				"value": "removal-check",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, records),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "A"),
					resource.TestCheckResourceAttr(resourceName, "records.1.type", "TXT"),
				),
			},
			{
				Config: testAccProviderOnlyConfig(),
			},
		},
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckDNSRecordAbsent(domain, "A", host),
			testAccCheckDNSRecordAbsent(domain, "TXT", host),
		),
	})
}

func TestAccDNSRecords_invalidRecordNameFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "A",
			Name: ".example.",
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "198.51.100.20",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Record Name"),
			},
		},
	})
}

// TestAccDNSRecords_numericPortInExistingZone reproduces the issue #19 scenario:
// a zone already contains an SRV record with a numeric port (e.g. Spacemail
// autodiscovery), and the provider must not crash when reading the zone while
// managing only CNAME records.
func TestAccDNSRecords_numericPortInExistingZone(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}

	srvName := fmt.Sprintf("%s-preseed-srv", prefix)
	cnameName := fmt.Sprintf("%s-cname-only", prefix)
	resourceName := "spaceship_dns_records.test"

	// SRV record with a numeric port, simulating Spacemail autodiscovery.
	srvRecord := client.DNSRecord{
		Type:     "SRV",
		Name:     srvName,
		TTL:      300,
		Port:     client.NewIntPortValue(443),
		Service:  "_autodiscover",
		Protocol: "_tcp",
		Priority: intPointerClient(0),
		Weight:   intPointerClient(0),
		Target:   fmt.Sprintf("autoconfig.%s", domain),
	}

	// Pre-seed the SRV record before Terraform runs.
	preConfig := func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to create test client: %v", err)
		}
		if err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{srvRecord}); err != nil {
			t.Fatalf("failed to pre-seed SRV record: %v", err)
		}
	}

	// Clean up the pre-seeded SRV record after the test.
	cleanup := func() {
		testClient, err := testAccClient()
		if err != nil {
			return
		}
		_ = testClient.DeleteDNSRecords(context.Background(), domain, []client.DNSRecord{srvRecord})
	}
	t.Cleanup(cleanup)

	// Terraform config manages only a CNAME record.
	records := []testAccDNSRecord{
		{
			Type: "CNAME",
			Name: cnameName,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"cname": fmt.Sprintf("origin.%s", domain),
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: preConfig,
				Config:    testAccDNSRecordsConfig(domain, records),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "CNAME"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", cnameName),
				),
			},
		},
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckDNSRecordAbsent(domain, "CNAME", cnameName),
		),
	})
}

// TestAccDNSRecords_existingRecordsDeletedOnCreate verifies that when DNS
// records already exist on the API before the Terraform resource is created,
// the Create operation deletes the pre-existing records that are not in the
// configuration. After the first apply, only the configured records remain
// and the pre-existing records are gone from the API.
func TestAccDNSRecords_existingRecordsDeletedOnCreate(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}

	resourceName := "spaceship_dns_records.test"
	managedName := fmt.Sprintf("%s-managed", prefix)
	preExistingName := fmt.Sprintf("%s-preexist", prefix)

	preExistingRecord := client.DNSRecord{
		Type:    "A",
		Name:    preExistingName,
		TTL:     3600,
		Address: "198.51.100.99",
	}

	// Pre-seed a record before Terraform runs.
	preConfig := func() {
		testClient, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to create test client: %v", err)
		}
		if err := testClient.UpsertDNSRecords(context.Background(), domain, true, []client.DNSRecord{preExistingRecord}); err != nil {
			t.Fatalf("failed to pre-seed record: %v", err)
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

	// Terraform config only manages a different record.
	managedRecords := []testAccDNSRecord{
		{
			Type: "A",
			Name: managedName,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "198.51.100.50",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Step 1: First apply with pre-existing record on the API.
				// Create fetches existing records, diffs against config,
				// deletes the pre-existing record, and upserts the managed one.
				// After apply, only the managed record should be in state.
				PreConfig: preConfig,
				Config:    testAccDNSRecordsConfig(domain, managedRecords),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "records.0.type", "A"),
					resource.TestCheckResourceAttr(resourceName, "records.0.name", managedName),
					resource.TestCheckResourceAttr(resourceName, "records.0.address", "198.51.100.50"),
					testAccCheckDNSRecordAbsent(domain, "A", preExistingName),
				),
			},
			{
				// Step 2: Re-apply the same config. Plan should be empty
				// because state already matches config exactly.
				Config: testAccDNSRecordsConfig(domain, managedRecords),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckDNSRecordAbsent(domain, "A", managedName),
			testAccCheckDNSRecordAbsent(domain, "A", preExistingName),
		),
	})
}

// TestAccDNSRecords_matchingRecordsNoChanges verifies that when the Terraform
// configuration exactly matches the real DNS state, a second plan produces no
// changes (empty plan).
func TestAccDNSRecords_matchingRecordsNoChanges(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}

	resourceName := "spaceship_dns_records.test"
	host := fmt.Sprintf("%s-stable", prefix)

	records := []testAccDNSRecord{
		{
			Type: "A",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "198.51.100.77",
			},
		},
		{
			Type: "TXT",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"value": "stable-check",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Step 1: Create the records.
				Config: testAccDNSRecordsConfig(domain, records),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "records.#", "2"),
				),
			},
			{
				// Step 2: Re-apply the identical config. Plan should be empty
				// (no changes) because state matches config exactly.
				Config: testAccDNSRecordsConfig(domain, records),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckDNSRecordAbsent(domain, "A", host),
			testAccCheckDNSRecordAbsent(domain, "TXT", host),
		),
	})
}

func intPointerClient(v int) *int {
	return &v
}

type testAccDNSRecord struct {
	Type        string
	Name        string
	TTL         *int
	StringAttrs map[string]string
	IntAttrs    map[string]int
}

func testAccDNSRecordsConfig(domain string, records []testAccDNSRecord) string {
	var b strings.Builder

	for _, record := range records {
		fmt.Fprintf(&b, `
    {
      type = %q
      name = %q`, record.Type, record.Name)

		if record.TTL != nil {
			fmt.Fprintf(&b, `
      ttl  = %d`, *record.TTL)
		}

		stringKeys := make([]string, 0, len(record.StringAttrs))
		for k := range record.StringAttrs {
			stringKeys = append(stringKeys, k)
		}
		sort.Strings(stringKeys)
		for _, k := range stringKeys {
			fmt.Fprintf(&b, `
      %s = %q`, k, record.StringAttrs[k])
		}

		intKeys := make([]string, 0, len(record.IntAttrs))
		for k := range record.IntAttrs {
			intKeys = append(intKeys, k)
		}
		sort.Strings(intKeys)
		for _, k := range intKeys {
			fmt.Fprintf(&b, `
      %s = %d`, k, record.IntAttrs[k])
		}

		fmt.Fprintf(&b, `
    },`)
	}

	return fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_dns_records" "test" {
  domain = %q

  records = [%s
  ]
}
`, domain, b.String())
}

func testAccProviderOnlyConfig() string {
	return `
provider "spaceship" {}
`
}

func intPointer(v int) *int {
	return &v
}
