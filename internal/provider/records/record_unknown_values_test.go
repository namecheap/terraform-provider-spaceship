package records

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Regression for GH #146: during the validate walk Terraform passes input
// variables (and values computed from other resources) as unknown. A required
// field that is set but not yet known must not be reported as missing — only a
// null field is. Each case lists a valid record and its required fields; every
// required field is then swapped for an unknown value and for a null value.
var requiredFieldCases = []struct {
	recordType string
	validator  validator.Object
	valid      map[string]attr.Value
	required   []string
}{
	{
		recordType: "A",
		validator:  AValidator(),
		valid:      map[string]attr.Value{"address": types.StringValue("192.0.2.1")},
		required:   []string{"address"},
	},
	{
		recordType: "AAAA",
		validator:  AAAAValidator(),
		valid:      map[string]attr.Value{"address": types.StringValue("2001:db8::1")},
		required:   []string{"address"},
	},
	{
		recordType: "ALIAS",
		validator:  ALIASValidator(),
		valid:      map[string]attr.Value{"alias_name": types.StringValue("target.example.com")},
		required:   []string{"alias_name"},
	},
	{
		recordType: "CAA",
		validator:  CAAValidator(),
		valid: map[string]attr.Value{
			"flag":  types.Int64Value(0),
			"tag":   types.StringValue("issue"),
			"value": types.StringValue("letsencrypt.org"),
		},
		required: []string{"flag", "tag", "value"},
	},
	{
		recordType: "CNAME",
		validator:  CNAMEValidator(),
		valid:      map[string]attr.Value{"cname": types.StringValue("target.example.com")},
		required:   []string{"cname"},
	},
	{
		recordType: "HTTPS",
		validator:  HTTPSValidator(),
		valid: map[string]attr.Value{
			"svc_priority": types.Int64Value(1),
			"target_name":  types.StringValue("target.example.com"),
		},
		required: []string{"svc_priority", "target_name"},
	},
	{
		recordType: "MX",
		validator:  MXValidator(),
		valid: map[string]attr.Value{
			"exchange":   types.StringValue("mail.example.com"),
			"preference": types.Int64Value(10),
		},
		required: []string{"exchange", "preference"},
	},
	{
		recordType: "NS",
		validator:  NSValidator(),
		valid:      map[string]attr.Value{"nameserver": types.StringValue("ns1.example.com")},
		required:   []string{"nameserver"},
	},
	{
		recordType: "PTR",
		validator:  PTRValidator(),
		valid:      map[string]attr.Value{"pointer": types.StringValue("host.example.com")},
		required:   []string{"pointer"},
	},
	{
		recordType: "SRV",
		validator:  SRVValidator(),
		valid: map[string]attr.Value{
			"service":     types.StringValue("_sip"),
			"protocol":    types.StringValue("_tcp"),
			"priority":    types.Int64Value(10),
			"weight":      types.Int64Value(5),
			"port_number": types.Int64Value(5060),
			"target":      types.StringValue("sip.example.com"),
		},
		required: []string{"service", "protocol", "priority", "weight", "port_number", "target"},
	},
	{
		recordType: "SVCB",
		validator:  SVCBValidator(),
		valid: map[string]attr.Value{
			"svc_priority": types.Int64Value(1),
			"target_name":  types.StringValue("target.example.com"),
		},
		required: []string{"svc_priority", "target_name"},
	},
	{
		recordType: "TLSA",
		validator:  TLSAValidator(),
		valid: map[string]attr.Value{
			"port":             types.StringValue("_443"),
			"protocol":         types.StringValue("_tcp"),
			"usage":            types.Int64Value(3),
			"selector":         types.Int64Value(1),
			"matching":         types.Int64Value(1),
			"association_data": types.StringValue(strings.Repeat("ab", 32)),
		},
		required: []string{"port", "protocol", "usage", "selector", "matching", "association_data"},
	},
	{
		recordType: "TXT",
		validator:  TXTValidator(),
		valid:      map[string]attr.Value{"value": types.StringValue("v=spf1 -all")},
		required:   []string{"value"},
	},
}

func buildCaseObject(t *testing.T, recordType string, fields map[string]attr.Value, field string, value attr.Value) types.Object {
	t.Helper()
	overrides := map[string]attr.Value{
		"type": types.StringValue(recordType),
		"name": types.StringValue("www"),
	}
	for k, v := range fields {
		overrides[k] = v
	}
	if field != "" {
		overrides[field] = value
	}
	return buildRecordObject(t, overrides)
}

func unknownOf(t *testing.T, field string) attr.Value {
	t.Helper()
	switch testRecordAttrTypes[field] {
	case types.StringType:
		return types.StringUnknown()
	case types.Int64Type:
		return types.Int64Unknown()
	}
	t.Fatalf("no unknown value for field %q", field)
	return nil
}

func nullOf(t *testing.T, field string) attr.Value {
	t.Helper()
	switch testRecordAttrTypes[field] {
	case types.StringType:
		return types.StringNull()
	case types.Int64Type:
		return types.Int64Null()
	}
	t.Fatalf("no null value for field %q", field)
	return nil
}

func validateRecordObject(v validator.Object, obj types.Object) *validator.ObjectResponse {
	resp := &validator.ObjectResponse{}
	v.ValidateObject(context.Background(), validator.ObjectRequest{Path: path.Empty(), ConfigValue: obj}, resp)
	return resp
}

// Guards the fixtures: if a base record is itself invalid, the unknown-value
// cases below would pass or fail for the wrong reason.
func TestRecordValidators_ValidFixturesPass(t *testing.T) {
	for _, tc := range requiredFieldCases {
		t.Run(tc.recordType, func(t *testing.T) {
			resp := validateRecordObject(tc.validator, buildCaseObject(t, tc.recordType, tc.valid, "", nil))
			if resp.Diagnostics.HasError() {
				t.Fatalf("expected valid %s fixture to pass, got: %s", tc.recordType, resp.Diagnostics)
			}
		})
	}
}

func TestRecordValidators_UnknownRequiredFieldIsNotMissing(t *testing.T) {
	for _, tc := range requiredFieldCases {
		for _, field := range tc.required {
			t.Run(tc.recordType+"/"+field, func(t *testing.T) {
				obj := buildCaseObject(t, tc.recordType, tc.valid, field, unknownOf(t, field))
				resp := validateRecordObject(tc.validator, obj)
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected unknown %q on %s record to be accepted, got: %s", field, tc.recordType, resp.Diagnostics)
				}
			})
		}
	}
}

func TestRecordValidators_NullRequiredFieldIsMissing(t *testing.T) {
	for _, tc := range requiredFieldCases {
		for _, field := range tc.required {
			t.Run(tc.recordType+"/"+field, func(t *testing.T) {
				obj := buildCaseObject(t, tc.recordType, tc.valid, field, nullOf(t, field))
				resp := validateRecordObject(tc.validator, obj)
				if !resp.Diagnostics.HasError() {
					t.Fatalf("expected null %q on %s record to be rejected, got no error", field, tc.recordType)
				}
				if got := resp.Diagnostics.Errors()[0].Summary(); got != "Missing Required Field" {
					t.Errorf("expected \"Missing Required Field\", got %q", got)
				}
			})
		}
	}
}
