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

var testRecordAttrTypes = map[string]attr.Type{
	"type": types.StringType, "name": types.StringType, "ttl": types.Int64Type,
	"address": types.StringType, "alias_name": types.StringType, "cname": types.StringType,
	"flag": types.Int64Type, "tag": types.StringType, "value": types.StringType,
	"port": types.StringType, "scheme": types.StringType, "svc_priority": types.Int64Type,
	"target_name": types.StringType, "svc_params": types.StringType,
	"exchange": types.StringType, "preference": types.Int64Type,
	"nameserver": types.StringType, "pointer": types.StringType,
	"service": types.StringType, "protocol": types.StringType,
	"priority": types.Int64Type, "weight": types.Int64Type, "port_number": types.Int64Type,
	"target": types.StringType, "usage": types.Int64Type, "selector": types.Int64Type,
	"matching": types.Int64Type, "association_data": types.StringType,
}

func buildRecordObject(t *testing.T, overrides map[string]attr.Value) types.Object {
	t.Helper()
	values := make(map[string]attr.Value, len(testRecordAttrTypes))
	for name, typ := range testRecordAttrTypes {
		switch typ {
		case types.StringType:
			values[name] = types.StringNull()
		case types.Int64Type:
			values[name] = types.Int64Null()
		}
	}
	for k, v := range overrides {
		values[k] = v
	}
	obj, diags := types.ObjectValue(testRecordAttrTypes, values)
	if diags.HasError() {
		t.Fatalf("failed to build record object: %s", diags)
	}
	return obj
}

// Regression: setting `address` on a TLSA record was silently ignored by the
// API-mapper but reappeared as state drift after Read, causing Terraform to
// destroy+recreate against the upsert-idempotent API on every apply (server
// record unchanged, plan kept showing replace).
func TestIrrelevantFieldsValidator_RejectsAddressOnTLSA(t *testing.T) {
	obj := buildRecordObject(t, map[string]attr.Value{
		"type":             types.StringValue("TLSA"),
		"name":             types.StringValue("@"),
		"port":             types.StringValue("_443"),
		"protocol":         types.StringValue("_tcp"),
		"usage":            types.Int64Value(2),
		"selector":         types.Int64Value(1),
		"matching":         types.Int64Value(1),
		"association_data": types.StringValue("aabbccdd"),
		"address":          types.StringValue("192.0.2.1"),
	})

	resp := &validator.ObjectResponse{}
	IrrelevantFieldsValidator().ValidateObject(context.Background(),
		validator.ObjectRequest{Path: path.Empty(), ConfigValue: obj}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected validation error for address on TLSA, got none")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Summary(), "Invalid Field for Record Type") {
		t.Errorf("unexpected diagnostic: %s", resp.Diagnostics.Errors()[0])
	}
}

func TestIrrelevantFieldsValidator_AllowsValidTLSA(t *testing.T) {
	obj := buildRecordObject(t, map[string]attr.Value{
		"type":             types.StringValue("TLSA"),
		"name":             types.StringValue("@"),
		"port":             types.StringValue("_443"),
		"protocol":         types.StringValue("_tcp"),
		"usage":            types.Int64Value(2),
		"selector":         types.Int64Value(1),
		"matching":         types.Int64Value(1),
		"association_data": types.StringValue("aabbccdd"),
	})

	resp := &validator.ObjectResponse{}
	IrrelevantFieldsValidator().ValidateObject(context.Background(),
		validator.ObjectRequest{Path: path.Empty(), ConfigValue: obj}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for valid TLSA, got: %s", resp.Diagnostics)
	}
}

// Empty/null/unknown values for non-applicable fields are not errors — only
// actually-set values are rejected. This keeps the validator quiet for the
// expected baseline where the user simply omits the wrong-type attributes.
func TestIrrelevantFieldsValidator_IgnoresNullForeignFields(t *testing.T) {
	obj := buildRecordObject(t, map[string]attr.Value{
		"type":    types.StringValue("A"),
		"name":    types.StringValue("@"),
		"address": types.StringValue("192.0.2.1"),
	})

	resp := &validator.ObjectResponse{}
	IrrelevantFieldsValidator().ValidateObject(context.Background(),
		validator.ObjectRequest{Path: path.Empty(), ConfigValue: obj}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for A with only address set, got: %s", resp.Diagnostics)
	}
}

func TestIrrelevantFieldsValidator_RejectsMultipleForeignFields(t *testing.T) {
	obj := buildRecordObject(t, map[string]attr.Value{
		"type":  types.StringValue("CNAME"),
		"name":  types.StringValue("www"),
		"cname": types.StringValue("example.com"),
		// Both of these belong to other types.
		"address":  types.StringValue("192.0.2.1"),
		"exchange": types.StringValue("mail.example.com"),
	})

	resp := &validator.ObjectResponse{}
	IrrelevantFieldsValidator().ValidateObject(context.Background(),
		validator.ObjectRequest{Path: path.Empty(), ConfigValue: obj}, resp)

	if len(resp.Diagnostics.Errors()) != 2 {
		t.Fatalf("expected 2 errors, got %d: %s", len(resp.Diagnostics.Errors()), resp.Diagnostics)
	}
}

// Universal fields (type, name, ttl) are never type-specific and must pass.
func TestIrrelevantFieldsValidator_AllowsUniversalFields(t *testing.T) {
	obj := buildRecordObject(t, map[string]attr.Value{
		"type":    types.StringValue("A"),
		"name":    types.StringValue("foo"),
		"ttl":     types.Int64Value(600),
		"address": types.StringValue("192.0.2.1"),
	})

	resp := &validator.ObjectResponse{}
	IrrelevantFieldsValidator().ValidateObject(context.Background(),
		validator.ObjectRequest{Path: path.Empty(), ConfigValue: obj}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected universal fields to pass, got: %s", resp.Diagnostics)
	}
}

// Unknown record type — leave validation to the schema-level OneOf so this
// validator doesn't double-report.
func TestIrrelevantFieldsValidator_UnknownTypeIsNoop(t *testing.T) {
	obj := buildRecordObject(t, map[string]attr.Value{
		"type":    types.StringValue("BOGUS"),
		"name":    types.StringValue("foo"),
		"address": types.StringValue("anything"),
	})

	resp := &validator.ObjectResponse{}
	IrrelevantFieldsValidator().ValidateObject(context.Background(),
		validator.ObjectRequest{Path: path.Empty(), ConfigValue: obj}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected unknown type to be a no-op, got: %s", resp.Diagnostics)
	}
}
