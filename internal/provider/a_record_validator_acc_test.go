package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDNSRecords_aRecord(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}

	resourceName := "spaceship_dns_records.test"
	host := fmt.Sprintf("%s-a", prefix)

	records := []testAccDNSRecord{
		{
			Type: "A",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "192.0.2.1",
			},
		},
	}

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "records.0.type", "A"),
		resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.0.address", "192.0.2.1"),
	}

	destroyChecks := []resource.TestCheckFunc{
		testAccCheckDNSRecordAbsent(domain, "A", host),
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

func TestAccDNSRecords_aRecordInvalidAddressFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	records := []testAccDNSRecord{
		{
			Type: "A",
			Name: "test-a-invalid",
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"address": "not-an-ip",
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

func TestAccDNSRecords_aRecordMissingAddressFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	// A record without address field
	records := []testAccDNSRecord{
		{
			Type: "A",
			Name: "test-a-noaddr",
			TTL:  intPointer(3600),
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
