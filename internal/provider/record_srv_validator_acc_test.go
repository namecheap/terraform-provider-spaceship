package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDNSRecords_srvRecord(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}

	resourceName := "spaceship_dns_records.test"
	host := fmt.Sprintf("%s-srv", prefix)

	records := []testAccDNSRecord{
		{
			Type: "SRV",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"service":  "_sip",
				"protocol": "_tcp",
				"target":   fmt.Sprintf("sipserver.%s", domain),
			},
			IntAttrs: map[string]int{
				"priority":    10,
				"weight":      60,
				"port_number": 5060,
			},
		},
	}

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "records.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "records.0.type", "SRV"),
		resource.TestCheckResourceAttr(resourceName, "records.0.name", host),
		resource.TestCheckResourceAttr(resourceName, "records.0.service", "_sip"),
		resource.TestCheckResourceAttr(resourceName, "records.0.protocol", "_tcp"),
		resource.TestCheckResourceAttr(resourceName, "records.0.priority", "10"),
		resource.TestCheckResourceAttr(resourceName, "records.0.weight", "60"),
		resource.TestCheckResourceAttr(resourceName, "records.0.port_number", "5060"),
		resource.TestCheckResourceAttr(resourceName, "records.0.target", fmt.Sprintf("sipserver.%s", domain)),
	}

	destroyChecks := []resource.TestCheckFunc{
		testAccCheckDNSRecordAbsent(domain, "SRV", host),
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

func TestAccDNSRecords_srvMissingRequiredFieldsFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()

	// SRV record missing service, protocol, priority, weight, port_number, target
	records := []testAccDNSRecord{
		{
			Type: "SRV",
			Name: "test-srv-invalid",
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

// Verifies that target="@" is rejected at plan time. The API returns 422
// "target is not a valid domain name" at apply time (empirically confirmed),
// so the client-layer validator short-circuits with a plan-time error.
func TestAccDNSRecords_srvRecordApexTargetFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	host := fmt.Sprintf("%s-srv-apextarget", prefix)

	records := []testAccDNSRecord{
		{
			Type: "SRV",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"service":  "_sip",
				"protocol": "_tcp",
				"target":   "@",
			},
			IntAttrs: map[string]int{
				"priority":    10,
				"weight":      60,
				"port_number": 5060,
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Target Value"),
			},
		},
	})
}

// Verifies that target="*" is rejected at plan time. Same rationale as the
// apex test: API returns 422 at apply time, plan-time rejection is the
// correct UX.
func TestAccDNSRecords_srvRecordWildcardTargetFailsPlan(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	host := fmt.Sprintf("%s-srv-wildtarget", prefix)

	records := []testAccDNSRecord{
		{
			Type: "SRV",
			Name: host,
			TTL:  intPointer(3600),
			StringAttrs: map[string]string{
				"service":  "_sip",
				"protocol": "_tcp",
				"target":   "*",
			},
			IntAttrs: map[string]int{
				"priority":    10,
				"weight":      60,
				"port_number": 5060,
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordsConfig(domain, records),
				ExpectError: regexp.MustCompile("Invalid Target Value"),
			},
		},
	})
}
