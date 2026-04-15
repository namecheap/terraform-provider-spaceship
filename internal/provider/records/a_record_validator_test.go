package records

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// attrTypes defines the attribute types for a DNS record object, matching the
// resource schema. Only "type" and "address" are relevant for A record
// validation, but all fields must be present for ObjectValueMust.
var attrTypes = map[string]attr.Type{
	"type":             types.StringType,
	"name":             types.StringType,
	"ttl":              types.Int64Type,
	"address":          types.StringType,
	"alias_name":       types.StringType,
	"cname":            types.StringType,
	"flag":             types.Int64Type,
	"tag":              types.StringType,
	"value":            types.StringType,
	"port":             types.StringType,
	"scheme":           types.StringType,
	"svc_priority":     types.Int64Type,
	"target_name":      types.StringType,
	"svc_params":       types.StringType,
	"exchange":         types.StringType,
	"preference":       types.Int64Type,
	"nameserver":       types.StringType,
	"pointer":          types.StringType,
	"service":          types.StringType,
	"protocol":         types.StringType,
	"priority":         types.Int64Type,
	"weight":           types.Int64Type,
	"port_number":      types.Int64Type,
	"target":           types.StringType,
	"usage":            types.Int64Type,
	"selector":         types.Int64Type,
	"matching":         types.Int64Type,
	"association_data": types.StringType,
}

// nullStringAttrs returns a map with all attrTypes keys set to null values of
// the correct type. Callers can override specific keys before passing to
// ObjectValueMust.
func nullStringAttrs() map[string]attr.Value {
	vals := make(map[string]attr.Value, len(attrTypes))
	for k, t := range attrTypes {
		switch t {
		case types.StringType:
			vals[k] = types.StringNull()
		case types.Int64Type:
			vals[k] = types.Int64Null()
		default:
			vals[k] = types.StringNull()
		}
	}
	return vals
}

func TestAValidator_ReturnsNonNil(t *testing.T) {
	v := AValidator()
	if v == nil {
		t.Fatal("AValidator() returned nil")
	}
}

func TestAValidator_Description(t *testing.T) {
	v := AValidator()
	ctx := context.Background()

	desc := v.Description(ctx)
	if desc == "" {
		t.Error("Description returned empty string")
	}

	md := v.MarkdownDescription(ctx)
	if md == "" {
		t.Error("MarkdownDescription returned empty string")
	}
}

func TestAValidator_ValidateObject(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		configValue types.Object
		expectError bool
		errorSumm   string
	}{
		{
			name:        "null config value",
			configValue: types.ObjectNull(attrTypes),
			expectError: false,
		},
		{
			name:        "unknown config value",
			configValue: types.ObjectUnknown(attrTypes),
			expectError: false,
		},
		{
			name: "non-A record type is skipped",
			configValue: func() types.Object {
				vals := nullStringAttrs()
				vals["type"] = types.StringValue("CNAME")
				return types.ObjectValueMust(attrTypes, vals)
			}(),
			expectError: false,
		},
		{
			name: "missing type attribute is skipped",
			configValue: func() types.Object {
				vals := nullStringAttrs()
				// type stays null
				return types.ObjectValueMust(attrTypes, vals)
			}(),
			expectError: false,
		},
		{
			name: "valid A record",
			configValue: func() types.Object {
				vals := nullStringAttrs()
				vals["type"] = types.StringValue("A")
				vals["address"] = types.StringValue("192.168.1.1")
				return types.ObjectValueMust(attrTypes, vals)
			}(),
			expectError: false,
		},
		{
			name: "invalid address value",
			configValue: func() types.Object {
				vals := nullStringAttrs()
				vals["type"] = types.StringValue("A")
				vals["address"] = types.StringValue("notanip")
				return types.ObjectValueMust(attrTypes, vals)
			}(),
			expectError: true,
			errorSumm:   "Invalid Address Value",
		},
		{
			name: "missing address (null)",
			configValue: func() types.Object {
				vals := nullStringAttrs()
				vals["type"] = types.StringValue("A")
				// address stays null
				return types.ObjectValueMust(attrTypes, vals)
			}(),
			expectError: true,
			errorSumm:   "Missing Required Field",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := AValidator()
			req := validator.ObjectRequest{
				Path:        path.Root("test"),
				ConfigValue: tc.configValue,
			}
			resp := &validator.ObjectResponse{}

			v.ValidateObject(ctx, req, resp)

			if tc.expectError && !resp.Diagnostics.HasError() {
				t.Fatal("expected error but got none")
			}
			if !tc.expectError && resp.Diagnostics.HasError() {
				t.Fatalf("expected no error but got: %s", resp.Diagnostics.Errors())
			}
			if tc.errorSumm != "" {
				found := false
				for _, d := range resp.Diagnostics.Errors() {
					if d.Summary() == tc.errorSumm {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error summary %q, got diagnostics: %s", tc.errorSumm, resp.Diagnostics.Errors())
				}
			}
		})
	}
}

// TestAValidator_InvalidFieldType tests the branch where the "address"
// attribute is not present in the object attributes at all (wrong type
// assertion). We build an object whose attrTypes lack "address" so the
// type assertion to types.String fails.
func TestAValidator_InvalidFieldType(t *testing.T) {
	// Build a minimal attrTypes that has "type" but replaces "address" with
	// an Int64 so the type assertion `attrs["address"].(types.String)` fails.
	customAttrTypes := make(map[string]attr.Type, len(attrTypes))
	for k, v := range attrTypes {
		customAttrTypes[k] = v
	}
	customAttrTypes["address"] = types.Int64Type

	vals := make(map[string]attr.Value, len(customAttrTypes))
	for k, t := range customAttrTypes {
		switch t {
		case types.StringType:
			vals[k] = types.StringNull()
		case types.Int64Type:
			vals[k] = types.Int64Null()
		}
	}
	vals["type"] = types.StringValue("A")
	vals["address"] = types.Int64Value(42)

	obj := types.ObjectValueMust(customAttrTypes, vals)

	ctx := context.Background()
	v := AValidator()
	req := validator.ObjectRequest{
		Path:        path.Root("test"),
		ConfigValue: obj,
	}
	resp := &validator.ObjectResponse{}

	v.ValidateObject(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong address type but got none")
	}
	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "Invalid Field Type" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error summary %q, got diagnostics: %s", "Invalid Field Type", resp.Diagnostics.Errors())
	}
}
