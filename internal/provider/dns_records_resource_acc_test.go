package provider

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDNSRecords_basicTypes(t *testing.T) {
	testAccPreCheck(t)

	domain := os.Getenv("SPACESHIP_TEST_DOMAIN")
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

	domain := os.Getenv("SPACESHIP_TEST_DOMAIN")
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

	domain := os.Getenv("SPACESHIP_TEST_DOMAIN")
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

	domain := os.Getenv("SPACESHIP_TEST_DOMAIN")

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
