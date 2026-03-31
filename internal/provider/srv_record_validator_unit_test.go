package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func srvTestObject(t *testing.T, attrs map[string]attr.Value) types.Object {
	t.Helper()

	// Start with all null values
	defaults := map[string]attr.Value{
		"type":             types.StringNull(),
		"name":             types.StringNull(),
		"ttl":              types.Int64Null(),
		"address":          types.StringNull(),
		"alias_name":       types.StringNull(),
		"cname":            types.StringNull(),
		"flag":             types.Int64Null(),
		"tag":              types.StringNull(),
		"value":            types.StringNull(),
		"port":             types.StringNull(),
		"scheme":           types.StringNull(),
		"svc_priority":     types.Int64Null(),
		"target_name":      types.StringNull(),
		"svc_params":       types.StringNull(),
		"exchange":         types.StringNull(),
		"preference":       types.Int64Null(),
		"nameserver":       types.StringNull(),
		"pointer":          types.StringNull(),
		"service":          types.StringNull(),
		"protocol":         types.StringNull(),
		"priority":         types.Int64Null(),
		"weight":           types.Int64Null(),
		"port_number":      types.Int64Null(),
		"target":           types.StringNull(),
		"usage":            types.Int64Null(),
		"selector":         types.Int64Null(),
		"matching":         types.Int64Null(),
		"association_data": types.StringNull(),
	}

	for k, v := range attrs {
		defaults[k] = v
	}

	obj, diags := types.ObjectValue(testRecordAttrTypes, defaults)
	if diags.HasError() {
		t.Fatalf("failed to create test object: %s", diags)
	}
	return obj
}

func TestSRVRecordValidator_ValidRecord(t *testing.T) {
	v := &srvRecordValidator{}
	obj := srvTestObject(t, map[string]attr.Value{
		"type":        types.StringValue("SRV"),
		"name":        types.StringValue("@"),
		"service":     types.StringValue("_sip"),
		"protocol":    types.StringValue("_tcp"),
		"priority":    types.Int64Value(10),
		"weight":      types.Int64Value(60),
		"port_number": types.Int64Value(5060),
		"target":      types.StringValue("sipserver.example.com"),
	})

	req := validator.ObjectRequest{
		Path:        path.Root("records").AtListIndex(0),
		ConfigValue: obj,
	}
	resp := &validator.ObjectResponse{}
	v.ValidateObject(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no errors for valid SRV record, got: %s", resp.Diagnostics)
	}
}

func TestSRVRecordValidator_NonSRVType_Skipped(t *testing.T) {
	v := &srvRecordValidator{}
	// A record with no SRV fields — should pass since type != SRV
	obj := srvTestObject(t, map[string]attr.Value{
		"type":    types.StringValue("A"),
		"name":    types.StringValue("@"),
		"address": types.StringValue("1.2.3.4"),
	})

	req := validator.ObjectRequest{
		Path:        path.Root("records").AtListIndex(0),
		ConfigValue: obj,
	}
	resp := &validator.ObjectResponse{}
	v.ValidateObject(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Error("expected no errors for non-SRV record type")
	}
}

func TestSRVRecordValidator_MissingAllFields(t *testing.T) {
	v := &srvRecordValidator{}
	obj := srvTestObject(t, map[string]attr.Value{
		"type": types.StringValue("SRV"),
		"name": types.StringValue("@"),
	})

	req := validator.ObjectRequest{
		Path:        path.Root("records").AtListIndex(0),
		ConfigValue: obj,
	}
	resp := &validator.ObjectResponse{}
	v.ValidateObject(context.Background(), req, resp)

	// Should report errors for service, protocol, priority, weight, port_number, target
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected errors for missing SRV fields")
	}

	errorCount := resp.Diagnostics.ErrorsCount()
	if errorCount != 6 {
		t.Errorf("expected 6 errors (service, protocol, priority, weight, port_number, target), got %d", errorCount)
	}
}

func TestSRVRecordValidator_InvalidServiceFormat(t *testing.T) {
	v := &srvRecordValidator{}
	obj := srvTestObject(t, map[string]attr.Value{
		"type":        types.StringValue("SRV"),
		"name":        types.StringValue("@"),
		"service":     types.StringValue("sip"), // missing leading underscore
		"protocol":    types.StringValue("_tcp"),
		"priority":    types.Int64Value(10),
		"weight":      types.Int64Value(60),
		"port_number": types.Int64Value(5060),
		"target":      types.StringValue("sip.example.com"),
	})

	req := validator.ObjectRequest{
		Path:        path.Root("records").AtListIndex(0),
		ConfigValue: obj,
	}
	resp := &validator.ObjectResponse{}
	v.ValidateObject(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for service without leading underscore")
	}
}

func TestSRVRecordValidator_InvalidProtocolFormat(t *testing.T) {
	v := &srvRecordValidator{}
	obj := srvTestObject(t, map[string]attr.Value{
		"type":        types.StringValue("SRV"),
		"name":        types.StringValue("@"),
		"service":     types.StringValue("_sip"),
		"protocol":    types.StringValue("tcp"), // missing leading underscore
		"priority":    types.Int64Value(10),
		"weight":      types.Int64Value(60),
		"port_number": types.Int64Value(5060),
		"target":      types.StringValue("sip.example.com"),
	})

	req := validator.ObjectRequest{
		Path:        path.Root("records").AtListIndex(0),
		ConfigValue: obj,
	}
	resp := &validator.ObjectResponse{}
	v.ValidateObject(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for protocol without leading underscore")
	}
}

func TestSRVRecordValidator_ServiceTooShort(t *testing.T) {
	v := &srvRecordValidator{}
	obj := srvTestObject(t, map[string]attr.Value{
		"type":        types.StringValue("SRV"),
		"name":        types.StringValue("@"),
		"service":     types.StringValue("_"), // 1 char, min is 2
		"protocol":    types.StringValue("_tcp"),
		"priority":    types.Int64Value(10),
		"weight":      types.Int64Value(60),
		"port_number": types.Int64Value(5060),
		"target":      types.StringValue("sip.example.com"),
	})

	req := validator.ObjectRequest{
		Path:        path.Root("records").AtListIndex(0),
		ConfigValue: obj,
	}
	resp := &validator.ObjectResponse{}
	v.ValidateObject(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for service that is too short")
	}
}

func TestSRVRecordValidator_TargetTooLong(t *testing.T) {
	v := &srvRecordValidator{}
	longTarget := make([]byte, 254)
	for i := range longTarget {
		longTarget[i] = 'a'
	}
	obj := srvTestObject(t, map[string]attr.Value{
		"type":        types.StringValue("SRV"),
		"name":        types.StringValue("@"),
		"service":     types.StringValue("_sip"),
		"protocol":    types.StringValue("_tcp"),
		"priority":    types.Int64Value(10),
		"weight":      types.Int64Value(60),
		"port_number": types.Int64Value(5060),
		"target":      types.StringValue(string(longTarget)),
	})

	req := validator.ObjectRequest{
		Path:        path.Root("records").AtListIndex(0),
		ConfigValue: obj,
	}
	resp := &validator.ObjectResponse{}
	v.ValidateObject(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for target exceeding 253 characters")
	}
}

func TestSRVRecordValidator_BoundaryValues(t *testing.T) {
	v := &srvRecordValidator{}
	// Zero priority and weight, port 1 — all valid boundary values
	obj := srvTestObject(t, map[string]attr.Value{
		"type":        types.StringValue("SRV"),
		"name":        types.StringValue("@"),
		"service":     types.StringValue("_s"),
		"protocol":    types.StringValue("_u"),
		"priority":    types.Int64Value(0),
		"weight":      types.Int64Value(0),
		"port_number": types.Int64Value(1),
		"target":      types.StringValue("a"),
	})

	req := validator.ObjectRequest{
		Path:        path.Root("records").AtListIndex(0),
		ConfigValue: obj,
	}
	resp := &validator.ObjectResponse{}
	v.ValidateObject(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no errors for boundary values, got: %s", resp.Diagnostics)
	}
}
