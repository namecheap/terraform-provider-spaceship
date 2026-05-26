package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// testAccRecordLifecycleCase describes a single record-type's inputs to the
// shared lifecycle test driver. initialDataHCL/changedDataHCL are the
// type-specific attribute lines inserted into the resource block; the data
// signatures must match what client.RecordValueSignature produces for those
// exact values (otherwise the ID assertions will fail).
//
// createBeforeDestroy is opt-in because the Spaceship API enforces a
// "one CNAME per hostname" constraint (and similar for ALIAS) that fails
// during a create-before-destroy swap: the new record gets rejected because
// the old one still exists. Default destroy-then-create works for every
// record type; only opt in for types known to support multiple coexisting
// records at the same (type, name) — A, AAAA, MX, TXT, NS, SRV, etc.
type testAccRecordLifecycleCase struct {
	recordType          string
	recordName          string
	initialDataHCL      string
	changedDataHCL      string
	initialDataSig      string
	changedDataSig      string
	initialChecks       []resource.TestCheckFunc
	changedChecks       []resource.TestCheckFunc
	createBeforeDestroy bool
}

func testAccRecordPrefix() string {
	prefix := os.Getenv("SPACESHIP_TEST_RECORD_PREFIX")
	if prefix == "" {
		prefix = "tfacc"
	}
	return prefix
}

// testAccDNSRecordLifecycle drives the standard 5-step acceptance lifecycle
// (create, empty-plan re-apply, ttl in-place update, data-field Replace,
// import). Each per-type test supplies the type-specific HCL and expected
// data signature; the assertions about lifecycle behavior (in-place vs
// Replace, ID stability, import round-trip) live here.
func testAccDNSRecordLifecycle(t *testing.T, tc testAccRecordLifecycleCase) {
	t.Helper()
	testAccPreCheck(t)

	domain := testAccDomainValue()
	resourceName := "spaceship_dns_record.test"

	lifecycleBlock := ""
	if tc.createBeforeDestroy {
		lifecycleBlock = `
  lifecycle {
    create_before_destroy = true
  }`
	}

	config := func(dataHCL string, ttl int) string {
		return fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_dns_record" "test" {
  domain = %q
  type   = %q
  name   = %q
  ttl    = %d

  %s
%s
}
`, domain, tc.recordType, tc.recordName, ttl, dataHCL, lifecycleBlock)
	}

	initialID := fmt.Sprintf("%s/%s/%s/%s", domain, tc.recordType, strings.ToLower(tc.recordName), tc.initialDataSig)
	changedID := fmt.Sprintf("%s/%s/%s/%s", domain, tc.recordType, strings.ToLower(tc.recordName), tc.changedDataSig)

	baseInitialChecks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "domain", domain),
		resource.TestCheckResourceAttr(resourceName, "type", tc.recordType),
		resource.TestCheckResourceAttr(resourceName, "name", tc.recordName),
		resource.TestCheckResourceAttr(resourceName, "ttl", "3600"),
		resource.TestCheckResourceAttr(resourceName, "id", initialID),
	}
	initialChecks := append(baseInitialChecks, tc.initialChecks...)

	baseChangedChecks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "ttl", "600"),
		resource.TestCheckResourceAttr(resourceName, "id", changedID),
	}
	changedChecks := append(baseChangedChecks, tc.changedChecks...)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1. Create
			{
				Config: config(tc.initialDataHCL, 3600),
				Check:  resource.ComposeTestCheckFunc(initialChecks...),
			},
			// 2. Re-apply same config — must converge with no diff
			{
				Config: config(tc.initialDataHCL, 3600),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			// 3. TTL change — in-place; composite ID stable
			{
				Config: config(tc.initialDataHCL, 600),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "ttl", "600"),
					resource.TestCheckResourceAttr(resourceName, "id", initialID),
				),
			},
			// 4. Data-field change — Replace; composite ID changes
			{
				Config: config(tc.changedDataHCL, 600),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeTestCheckFunc(changedChecks...),
			},
			// 5. Import the replacement record by composite ID
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, tc.recordType, tc.recordName),
	})
}

func TestAccDNSRecord_AAAA_lifecycle(t *testing.T) {
	prefix := testAccRecordPrefix()
	resourceName := "spaceship_dns_record.test"
	testAccDNSRecordLifecycle(t, testAccRecordLifecycleCase{
		recordType:     "AAAA",
		recordName:     prefix + "-aaaa",
		initialDataHCL: `address = "2001:db8::1"`,
		changedDataHCL: `address = "2001:db8::2"`,
		initialDataSig: "2001:db8::1",
		changedDataSig: "2001:db8::2",
		initialChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "address", "2001:db8::1"),
		},
		changedChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "address", "2001:db8::2"),
		},
	})
}

func TestAccDNSRecord_ALIAS_lifecycle(t *testing.T) {
	prefix := testAccRecordPrefix()
	resourceName := "spaceship_dns_record.test"
	testAccDNSRecordLifecycle(t, testAccRecordLifecycleCase{
		recordType:     "ALIAS",
		recordName:     prefix + "-alias",
		initialDataHCL: `alias_name = "target1.example.com"`,
		changedDataHCL: `alias_name = "target2.example.com"`,
		initialDataSig: "target1.example.com",
		changedDataSig: "target2.example.com",
		initialChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "alias_name", "target1.example.com"),
		},
		changedChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "alias_name", "target2.example.com"),
		},
	})
}

func TestAccDNSRecord_CAA_lifecycle(t *testing.T) {
	prefix := testAccRecordPrefix()
	resourceName := "spaceship_dns_record.test"
	testAccDNSRecordLifecycle(t, testAccRecordLifecycleCase{
		recordType: "CAA",
		recordName: prefix + "-caa",
		initialDataHCL: `flag  = 0
  tag   = "issue"
  value = "letsencrypt.org"`,
		changedDataHCL: `flag  = 0
  tag   = "issue"
  value = "digicert.com"`,
		initialDataSig: "0|issue|letsencrypt.org",
		changedDataSig: "0|issue|digicert.com",
		initialChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "flag", "0"),
			resource.TestCheckResourceAttr(resourceName, "tag", "issue"),
			resource.TestCheckResourceAttr(resourceName, "value", "letsencrypt.org"),
		},
		changedChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "value", "digicert.com"),
		},
	})
}

func TestAccDNSRecord_CNAME_lifecycle(t *testing.T) {
	prefix := testAccRecordPrefix()
	resourceName := "spaceship_dns_record.test"
	testAccDNSRecordLifecycle(t, testAccRecordLifecycleCase{
		recordType:     "CNAME",
		recordName:     prefix + "-cname",
		initialDataHCL: `cname = "origin1.example.com"`,
		changedDataHCL: `cname = "origin2.example.com"`,
		initialDataSig: "origin1.example.com",
		changedDataSig: "origin2.example.com",
		initialChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "cname", "origin1.example.com"),
		},
		changedChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "cname", "origin2.example.com"),
		},
	})
}

func TestAccDNSRecord_HTTPS_lifecycle(t *testing.T) {
	prefix := testAccRecordPrefix()
	resourceName := "spaceship_dns_record.test"
	testAccDNSRecordLifecycle(t, testAccRecordLifecycleCase{
		recordType: "HTTPS",
		recordName: prefix + "-https",
		initialDataHCL: `svc_priority = 1
  target_name  = "target1.example.com"
  svc_params   = "alpn=h2"
  port         = "_443"
  scheme       = "_https"`,
		changedDataHCL: `svc_priority = 1
  target_name  = "target2.example.com"
  svc_params   = "alpn=h2"
  port         = "_443"
  scheme       = "_https"`,
		initialDataSig: "1|target1.example.com|alpn=h2|_443|_https",
		changedDataSig: "1|target2.example.com|alpn=h2|_443|_https",
		initialChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "svc_priority", "1"),
			resource.TestCheckResourceAttr(resourceName, "target_name", "target1.example.com"),
		},
		changedChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "target_name", "target2.example.com"),
		},
	})
}

func TestAccDNSRecord_MX_lifecycle(t *testing.T) {
	prefix := testAccRecordPrefix()
	resourceName := "spaceship_dns_record.test"
	testAccDNSRecordLifecycle(t, testAccRecordLifecycleCase{
		recordType: "MX",
		recordName: prefix + "-mx",
		initialDataHCL: `exchange   = "mail1.example.com"
  preference = 10`,
		changedDataHCL: `exchange   = "mail2.example.com"
  preference = 10`,
		initialDataSig: "mail1.example.com|10",
		changedDataSig: "mail2.example.com|10",
		initialChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "exchange", "mail1.example.com"),
			resource.TestCheckResourceAttr(resourceName, "preference", "10"),
		},
		changedChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "exchange", "mail2.example.com"),
		},
	})
}

func TestAccDNSRecord_NS_lifecycle(t *testing.T) {
	prefix := testAccRecordPrefix()
	resourceName := "spaceship_dns_record.test"
	testAccDNSRecordLifecycle(t, testAccRecordLifecycleCase{
		recordType:     "NS",
		recordName:     prefix + "-ns",
		initialDataHCL: `nameserver = "ns1.example.com"`,
		changedDataHCL: `nameserver = "ns2.example.com"`,
		initialDataSig: "ns1.example.com",
		changedDataSig: "ns2.example.com",
		initialChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "nameserver", "ns1.example.com"),
		},
		changedChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "nameserver", "ns2.example.com"),
		},
	})
}

func TestAccDNSRecord_PTR_lifecycle(t *testing.T) {
	prefix := testAccRecordPrefix()
	resourceName := "spaceship_dns_record.test"
	testAccDNSRecordLifecycle(t, testAccRecordLifecycleCase{
		recordType:     "PTR",
		recordName:     prefix + "-ptr",
		initialDataHCL: `pointer = "host1.example.com"`,
		changedDataHCL: `pointer = "host2.example.com"`,
		initialDataSig: "host1.example.com",
		changedDataSig: "host2.example.com",
		initialChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "pointer", "host1.example.com"),
		},
		changedChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "pointer", "host2.example.com"),
		},
	})
}

// SRV: change `priority` to trigger Replace via a field that's actually in
// recordValueSignature (`service|protocol|priority|weight`). Changing only
// `target` would not change the signature, so the ID would not change either
// — Replace would still happen because the schema marks every non-ttl field
// RequiresReplace, but the ID assertion in step 4 would fail.
func TestAccDNSRecord_SRV_lifecycle(t *testing.T) {
	prefix := testAccRecordPrefix()
	resourceName := "spaceship_dns_record.test"
	testAccDNSRecordLifecycle(t, testAccRecordLifecycleCase{
		recordType: "SRV",
		recordName: prefix + "-srv",
		initialDataHCL: `service     = "_sip"
  protocol    = "_tcp"
  priority    = 5
  weight      = 10
  port_number = 5060
  target      = "srv.example.com"`,
		changedDataHCL: `service     = "_sip"
  protocol    = "_tcp"
  priority    = 10
  weight      = 10
  port_number = 5060
  target      = "srv.example.com"`,
		initialDataSig: "_sip|_tcp|5|10",
		changedDataSig: "_sip|_tcp|10|10",
		initialChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "service", "_sip"),
			resource.TestCheckResourceAttr(resourceName, "protocol", "_tcp"),
			resource.TestCheckResourceAttr(resourceName, "priority", "5"),
			resource.TestCheckResourceAttr(resourceName, "weight", "10"),
			resource.TestCheckResourceAttr(resourceName, "port_number", "5060"),
			resource.TestCheckResourceAttr(resourceName, "target", "srv.example.com"),
		},
		changedChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "priority", "10"),
		},
	})
}

func TestAccDNSRecord_SVCB_lifecycle(t *testing.T) {
	prefix := testAccRecordPrefix()
	resourceName := "spaceship_dns_record.test"
	testAccDNSRecordLifecycle(t, testAccRecordLifecycleCase{
		recordType: "SVCB",
		recordName: prefix + "-svcb",
		initialDataHCL: `svc_priority = 1
  target_name  = "svc1.example.com"
  svc_params   = "alpn=h2"
  port         = "_853"
  scheme       = "_dot"`,
		changedDataHCL: `svc_priority = 1
  target_name  = "svc2.example.com"
  svc_params   = "alpn=h2"
  port         = "_853"
  scheme       = "_dot"`,
		initialDataSig: "1|svc1.example.com|alpn=h2|_853|_dot",
		changedDataSig: "1|svc2.example.com|alpn=h2|_853|_dot",
		initialChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "svc_priority", "1"),
			resource.TestCheckResourceAttr(resourceName, "target_name", "svc1.example.com"),
		},
		changedChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "target_name", "svc2.example.com"),
		},
	})
}

// TLSA: association_data is normalized to lowercase in the signature (see
// dns_match.go). Using lowercase hex in the HCL keeps state, ID, and signature
// in sync so ImportStateVerify in step 5 doesn't trip on case mismatches.
//
// The values are 64 hex chars apiece — the size of a SHA-256 hash, which is
// the minimum the per-type validator (records.TLSAValidator) accepts.
// Anything shorter fails at plan time with "must be between 64 and 65535
// characters" once the ConfigValidators adapter is in place.
func TestAccDNSRecord_TLSA_lifecycle(t *testing.T) {
	prefix := testAccRecordPrefix()
	resourceName := "spaceship_dns_record.test"
	initialAssoc := "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
	changedAssoc := "1122334455667788991122334455667788991122334455667788991122334455"
	testAccDNSRecordLifecycle(t, testAccRecordLifecycleCase{
		recordType: "TLSA",
		recordName: prefix + "-tlsa",
		initialDataHCL: fmt.Sprintf(`port             = "_443"
  protocol         = "_tcp"
  usage            = 2
  selector         = 1
  matching         = 1
  association_data = %q`, initialAssoc),
		changedDataHCL: fmt.Sprintf(`port             = "_443"
  protocol         = "_tcp"
  usage            = 2
  selector         = 1
  matching         = 1
  association_data = %q`, changedAssoc),
		initialDataSig: "_443|_tcp|2|1|1|" + initialAssoc,
		changedDataSig: "_443|_tcp|2|1|1|" + changedAssoc,
		initialChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "association_data", initialAssoc),
		},
		changedChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "association_data", changedAssoc),
		},
	})
}

// TXT: value is case-sensitive in the signature (TXT is the one type the API
// matches case-sensitively). Both initial and changed values stay in lowercase
// because that's how they're stored verbatim — no normalization.
func TestAccDNSRecord_TXT_lifecycle(t *testing.T) {
	prefix := testAccRecordPrefix()
	resourceName := "spaceship_dns_record.test"
	testAccDNSRecordLifecycle(t, testAccRecordLifecycleCase{
		recordType:     "TXT",
		recordName:     prefix + "-txt",
		initialDataHCL: `value = "v=spf1 a -all"`,
		changedDataHCL: `value = "v=spf1 mx -all"`,
		initialDataSig: "v=spf1 a -all",
		changedDataSig: "v=spf1 mx -all",
		initialChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "value", "v=spf1 a -all"),
		},
		changedChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "value", "v=spf1 mx -all"),
		},
	})
}

// TestAccDNSRecord_A_lifecycle walks an A record through the standard
// lifecycle and additionally opts in to the recommended create-before-destroy
// pattern (safe for A because the API allows multiple A records at the same
// name — they coexist briefly during the Replace swap).
func TestAccDNSRecord_A_lifecycle(t *testing.T) {
	prefix := testAccRecordPrefix()
	resourceName := "spaceship_dns_record.test"
	testAccDNSRecordLifecycle(t, testAccRecordLifecycleCase{
		recordType:          "A",
		recordName:          prefix + "-a",
		initialDataHCL:      `address = "198.51.100.10"`,
		changedDataHCL:      `address = "198.51.100.20"`,
		initialDataSig:      "198.51.100.10",
		changedDataSig:      "198.51.100.20",
		createBeforeDestroy: true,
		initialChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "address", "198.51.100.10"),
		},
		changedChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "address", "198.51.100.20"),
		},
	})
}

// TestAccDNSRecord_createWhenIdenticalExists is an empirical probe of the
// API's behavior when Terraform tries to "create" a record that already
// exists with identical (type, name, data). The test pre-creates the record
// directly via the client (simulating something created manually or by
// another tool) and then runs Terraform's Create against a matching config.
//
// Three possible outcomes:
//   - Step 1 passes + step 2 empty plan → upsert is idempotent for matching
//     records; no code changes needed. Standard Terraform import is still
//     the right path for non-matching pre-existing records.
//   - Step 1 fails with "already exists" → API rejects duplicates and the
//     resource needs explicit adopt-on-create or a similar opt-in.
//   - Step 1 passes but CheckDestroy fails → a duplicate was created; the
//     API tolerates duplicates somehow and we have a different problem.
func TestAccDNSRecord_createWhenIdenticalExists(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccDomainValue()
	prefix := testAccRecordPrefix()
	recordName := prefix + "-preexist"
	address := "203.0.113.50"
	resourceName := "spaceship_dns_record.test"

	config := fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_dns_record" "test" {
  domain  = %q
  type    = "A"
  name    = %q
  ttl     = 3600
  address = %q
}
`, domain, recordName, address)

	// Create the record outside Terraform's lifecycle, before any test step
	// runs. The record has the exact identity Terraform will request below.
	preCreate := func() {
		c, err := testAccClient()
		if err != nil {
			t.Fatalf("failed to construct client: %s", err)
		}
		record := client.DNSRecord{
			Type:    "A",
			Name:    recordName,
			TTL:     3600,
			Address: address,
		}
		if err := c.CreateDNSRecord(context.Background(), domain, record); err != nil {
			t.Fatalf("failed to pre-create DNS record: %s", err)
		}
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1. Terraform applies a config matching the pre-created record.
			//    Expectation: succeeds (upsert is idempotent for identical
			//    records). The record's composite ID must reflect the same
			//    identity used in pre-create.
			{
				PreConfig: preCreate,
				Config:    config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "A"),
					resource.TestCheckResourceAttr(resourceName, "name", recordName),
					resource.TestCheckResourceAttr(resourceName, "address", address),
					resource.TestCheckResourceAttr(resourceName, "ttl", "3600"),
					resource.TestCheckResourceAttr(resourceName, "id", fmt.Sprintf("%s/A/%s/%s", domain, strings.ToLower(recordName), address)),
				),
			},
			// 2. Re-apply: confirms state matches API (no drift from the
			//    pre-existing record being adopted/upserted in step 1).
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
		// Destroy should remove the record (regardless of who originally
		// created it). If a duplicate exists, this absent check fails and
		// surfaces the bug.
		CheckDestroy: testAccCheckDNSRecordAbsent(domain, "A", recordName),
	})
}
