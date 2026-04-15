package records

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSRVValidator_ReturnsNonNil(t *testing.T) {
	v := SRVValidator()
	if v == nil {
		t.Fatal("expected non-nil validator")
	}
}

func TestSRVValidator_Description(t *testing.T) {
	v := SRVValidator()
	ctx := context.Background()

	desc := v.Description(ctx)
	if desc == "" {
		t.Fatal("expected non-empty Description")
	}

	mdDesc := v.MarkdownDescription(ctx)
	if mdDesc == "" {
		t.Fatal("expected non-empty MarkdownDescription")
	}
}

// srvAttrTypes returns the full attribute type map used for constructing SRV record objects.
func srvAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":        types.StringType,
		"service":     types.StringType,
		"protocol":    types.StringType,
		"priority":    types.Int64Type,
		"weight":      types.Int64Type,
		"port_number": types.Int64Type,
		"target":      types.StringType,
	}
}

// validSRVAttrs returns a valid set of SRV record attribute values.
func validSRVAttrs() map[string]attr.Value {
	return map[string]attr.Value{
		"type":        types.StringValue("SRV"),
		"service":     types.StringValue("_sip"),
		"protocol":    types.StringValue("_tcp"),
		"priority":    types.Int64Value(10),
		"weight":      types.Int64Value(60),
		"port_number": types.Int64Value(5060),
		"target":      types.StringValue("sipserver.example.com"),
	}
}

func TestSRVValidator_ValidateObject(t *testing.T) {
	ctx := context.Background()
	attrTypes := srvAttrTypes()

	t.Run("null config value produces no errors", func(t *testing.T) {
		req := validator.ObjectRequest{
			Path:        path.Root("test"),
			ConfigValue: types.ObjectNull(attrTypes),
		}
		resp := &validator.ObjectResponse{}
		SRVValidator().ValidateObject(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors, got: %s", resp.Diagnostics)
		}
	})

	t.Run("unknown config value produces no errors", func(t *testing.T) {
		req := validator.ObjectRequest{
			Path:        path.Root("test"),
			ConfigValue: types.ObjectUnknown(attrTypes),
		}
		resp := &validator.ObjectResponse{}
		SRVValidator().ValidateObject(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors, got: %s", resp.Diagnostics)
		}
	})

	t.Run("non-SRV record type is skipped", func(t *testing.T) {
		vals := validSRVAttrs()
		vals["type"] = types.StringValue("A")
		obj := types.ObjectValueMust(attrTypes, vals)

		req := validator.ObjectRequest{
			Path:        path.Root("test"),
			ConfigValue: obj,
		}
		resp := &validator.ObjectResponse{}
		SRVValidator().ValidateObject(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors for non-SRV type, got: %s", resp.Diagnostics)
		}
	})

	t.Run("missing type attribute is skipped", func(t *testing.T) {
		// Build an object without the "type" key at all. Use a minimal attr type set.
		minTypes := map[string]attr.Type{
			"service": types.StringType,
		}
		minVals := map[string]attr.Value{
			"service": types.StringValue("_sip"),
		}
		obj := types.ObjectValueMust(minTypes, minVals)

		req := validator.ObjectRequest{
			Path:        path.Root("test"),
			ConfigValue: obj,
		}
		resp := &validator.ObjectResponse{}
		SRVValidator().ValidateObject(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors when type is missing, got: %s", resp.Diagnostics)
		}
	})

	t.Run("valid SRV record produces no errors", func(t *testing.T) {
		obj := types.ObjectValueMust(attrTypes, validSRVAttrs())

		req := validator.ObjectRequest{
			Path:        path.Root("test"),
			ConfigValue: obj,
		}
		resp := &validator.ObjectResponse{}
		SRVValidator().ValidateObject(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors for valid SRV record, got: %s", resp.Diagnostics)
		}
	})
}

func TestSRVValidator_MissingRequiredFields(t *testing.T) {
	ctx := context.Background()
	attrTypes := srvAttrTypes()

	// Each case removes one field by setting it to null and expects a "Missing Required Field" error.
	cases := []struct {
		name  string
		field string
	}{
		{name: "missing service", field: "service"},
		{name: "missing protocol", field: "protocol"},
		{name: "missing priority", field: "priority"},
		{name: "missing weight", field: "weight"},
		{name: "missing port_number", field: "port_number"},
		{name: "missing target", field: "target"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vals := validSRVAttrs()

			// Set the target field to null based on its type.
			switch attrTypes[tc.field] {
			case types.StringType:
				vals[tc.field] = types.StringNull()
			case types.Int64Type:
				vals[tc.field] = types.Int64Null()
			}

			obj := types.ObjectValueMust(attrTypes, vals)
			req := validator.ObjectRequest{
				Path:        path.Root("test"),
				ConfigValue: obj,
			}
			resp := &validator.ObjectResponse{}
			SRVValidator().ValidateObject(ctx, req, resp)

			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected error for %s, got none", tc.field)
			}

			found := false
			for _, d := range resp.Diagnostics {
				if d.Summary() == "Missing Required Field" {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected 'Missing Required Field' for %s, got: %s", tc.field, resp.Diagnostics)
			}
		})
	}
}

func TestSRVValidator_InvalidFieldValues(t *testing.T) {
	ctx := context.Background()
	attrTypes := srvAttrTypes()

	cases := []struct {
		name            string
		field           string
		value           attr.Value
		expectedSummary string
	}{
		{
			name:            "service without leading underscore",
			field:           "service",
			value:           types.StringValue("sip"),
			expectedSummary: "Invalid Service Value",
		},
		{
			name:            "protocol without leading underscore",
			field:           "protocol",
			value:           types.StringValue("tcp"),
			expectedSummary: "Invalid Protocol Value",
		},
		{
			name:            "service too short",
			field:           "service",
			value:           types.StringValue("_"),
			expectedSummary: "Invalid Service Value",
		},
		{
			name:            "protocol too short",
			field:           "protocol",
			value:           types.StringValue("_"),
			expectedSummary: "Invalid Protocol Value",
		},
		{
			name:            "priority out of range",
			field:           "priority",
			value:           types.Int64Value(-1),
			expectedSummary: "Invalid Priority Value",
		},
		{
			name:            "weight out of range",
			field:           "weight",
			value:           types.Int64Value(-1),
			expectedSummary: "Invalid Weight Value",
		},
		{
			name:            "port_number zero",
			field:           "port_number",
			value:           types.Int64Value(0),
			expectedSummary: "Invalid Port Value",
		},
		{
			name:            "port_number out of range",
			field:           "port_number",
			value:           types.Int64Value(70000),
			expectedSummary: "Invalid Port Value",
		},
		{
			name:            "target with invalid hostname",
			field:           "target",
			value:           types.StringValue("not a valid hostname!@#"),
			expectedSummary: "Invalid Target Value",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vals := validSRVAttrs()
			vals[tc.field] = tc.value

			obj := types.ObjectValueMust(attrTypes, vals)
			req := validator.ObjectRequest{
				Path:        path.Root("test"),
				ConfigValue: obj,
			}
			resp := &validator.ObjectResponse{}
			SRVValidator().ValidateObject(ctx, req, resp)

			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected validation error for %s, got none", tc.name)
			}

			found := false
			for _, d := range resp.Diagnostics {
				if d.Summary() == tc.expectedSummary {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected %q error for %s, got: %s", tc.expectedSummary, tc.name, resp.Diagnostics)
			}
		})
	}
}

func TestSRVValidator_WrongTypeAssertions(t *testing.T) {
	ctx := context.Background()

	// Build an object where fields that should be strings are int64 and vice versa,
	// causing type assertion failures in the validator.
	cases := []struct {
		name      string
		field     string
		attrType  attr.Type
		attrValue attr.Value
	}{
		{
			name:      "service as int64 instead of string",
			field:     "service",
			attrType:  types.Int64Type,
			attrValue: types.Int64Value(42),
		},
		{
			name:      "protocol as int64 instead of string",
			field:     "protocol",
			attrType:  types.Int64Type,
			attrValue: types.Int64Value(42),
		},
		{
			name:      "priority as string instead of int64",
			field:     "priority",
			attrType:  types.StringType,
			attrValue: types.StringValue("ten"),
		},
		{
			name:      "weight as string instead of int64",
			field:     "weight",
			attrType:  types.StringType,
			attrValue: types.StringValue("sixty"),
		},
		{
			name:      "port_number as string instead of int64",
			field:     "port_number",
			attrType:  types.StringType,
			attrValue: types.StringValue("5060"),
		},
		{
			name:      "target as int64 instead of string",
			field:     "target",
			attrType:  types.Int64Type,
			attrValue: types.Int64Value(99),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Start with valid types/values then swap the one under test.
			at := srvAttrTypes()
			vals := validSRVAttrs()

			at[tc.field] = tc.attrType
			vals[tc.field] = tc.attrValue

			obj := types.ObjectValueMust(at, vals)
			req := validator.ObjectRequest{
				Path:        path.Root("test"),
				ConfigValue: obj,
			}
			resp := &validator.ObjectResponse{}
			SRVValidator().ValidateObject(ctx, req, resp)

			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected error for wrong type on %s, got none", tc.field)
			}

			found := false
			for _, d := range resp.Diagnostics {
				if d.Summary() == "Invalid Field Type" {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected 'Invalid Field Type' for %s, got: %s", tc.field, resp.Diagnostics)
			}
		})
	}
}
