package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// aaaaHost returns a unique record name per test so parallel/repeat runs
// against the same SPACESHIP_TEST_DOMAIN don't collide.
func aaaaHost(suffix string) string {
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	return fmt.Sprintf("%s-aaaa-%s", prefix, suffix)
}

// Verifies that a standard compressed IPv6 AAAA record round-trips through
// create, import, and destroy.
func TestAccDNSRecords_aaaaRecord(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := aaaaHost("basic")
	address := "2001:db8::1"

	records := []testAccDNSRecord{
		{
			Type: "AAAA",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": address,
			},
		},
	}

	resourceName := "spaceship_dns_records.test"
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "records.0.type", "AAAA"),
		resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.0.address", address),
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
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "AAAA", host),
	})
}

// Verifies that the IPv6 loopback `::1` is accepted.
func TestAccDNSRecords_aaaaRecordLoopback(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := aaaaHost("loopback")
	address := "::1"

	records := []testAccDNSRecord{
		{
			Type: "AAAA",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": address,
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, records),
				Check: resource.TestCheckResourceAttr(
					"spaceship_dns_records.test", "records.0.address", address),
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "AAAA", host),
	})
}

// Verifies that an IPv6 address at exactly 39 characters (the documented
// maximum) is accepted.
func TestAccDNSRecords_aaaaRecordMaxLength(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := aaaaHost("maxlen")
	address := "2001:0db8:85a3:0000:0000:8a2e:0370:7334" // exactly 39 chars

	records := []testAccDNSRecord{
		{
			Type: "AAAA",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": address,
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, records),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.0.type", "AAAA"),
					// Note: API may normalize the address (e.g., compress to
					// 2001:db8:85a3::8a2e:370:7334). If the assertion fails
					// with a normalized value, that's useful diagnostic info.
					resource.TestCheckResourceAttr("spaceship_dns_records.test", "records.0.address", address),
				),
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "AAAA", host),
	})
}

// Verifies that the unspecified address `::` is accepted.
func TestAccDNSRecords_aaaaRecordUnspecified(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	host := aaaaHost("unspec")
	address := "::"

	records := []testAccDNSRecord{
		{
			Type: "AAAA",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": address,
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsConfig(domain, records),
				Check: resource.TestCheckResourceAttr(
					"spaceship_dns_records.test", "records.0.address", address),
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "AAAA", host),
	})
}

// Verifies that a plain IPv4 address is rejected for an AAAA record.
func TestAccDNSRecords_aaaaRecordIPv4AsAddressFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "AAAA",
			Name: aaaaHost("ipv4leak"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "192.0.2.1", // RFC 5737 documentation IPv4
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Address Value"),
			},
		},
	})
}

// Verifies that IPv4-mapped IPv6 (`::ffff:a.b.c.d`) is rejected for an AAAA record.
func TestAccDNSRecords_aaaaRecordIPv4MappedFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "AAAA",
			Name: aaaaHost("v4mapped"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "::ffff:192.0.2.1",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Address Value"),
			},
		},
	})
}

// Verifies that an unparseable address string is rejected.
func TestAccDNSRecords_aaaaRecordInvalidAddressFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "AAAA",
			Name: aaaaHost("badaddr"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "notanip",
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Address Value"),
			},
		},
	})
}

// Verifies that an address longer than 39 characters is rejected.
func TestAccDNSRecords_aaaaRecordTooLongAddressFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	address := "2001:0db8:85a3:0000:0000:8a2e:0370:73344" // 40 chars

	records := []testAccDNSRecord{
		{
			Type: "AAAA",
			Name: aaaaHost("toolong"),
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": address,
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Address Value"),
			},
		},
	})
}

// Verifies that a missing `address` field is rejected at plan time.
func TestAccDNSRecords_aaaaRecordMissingAddressFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "AAAA",
			Name: aaaaHost("noaddr"),
			TTL:  intPointer(3600),
			// no address
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
