package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNameserversValidator_MarkdownDescription(t *testing.T) {
	v := &nameserversValidator{}
	desc := v.MarkdownDescription(context.Background())

	if desc == "" {
		t.Error("expected non-empty MarkdownDescription")
	}
}

func TestNameserversValidator_ValidateObject(t *testing.T) {
	attrTypes := map[string]attr.Type{
		"provider": types.StringType,
		"hosts":    types.SetType{ElemType: types.StringType},
	}

	tests := []struct {
		name        string
		configValue types.Object
		expectError string
	}{
		{
			name:        "null config produces no errors",
			configValue: types.ObjectNull(attrTypes),
			expectError: "",
		},
		{
			name:        "unknown config produces no errors",
			configValue: types.ObjectUnknown(attrTypes),
			expectError: "",
		},
		{
			name: "missing provider attr (null) produces no errors",
			configValue: types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"provider": types.StringNull(),
				"hosts":    types.SetNull(types.StringType),
			}),
			expectError: "",
		},
		{
			name: "provider custom with valid hosts produces no errors",
			configValue: types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"provider": types.StringValue("custom"),
				"hosts": types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1.example.com"),
					types.StringValue("ns2.example.com"),
				}),
			}),
			expectError: "",
		},
		{
			name: "provider custom with null hosts produces error",
			configValue: types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"provider": types.StringValue("custom"),
				"hosts":    types.SetNull(types.StringType),
			}),
			expectError: "Missing Required Hosts",
		},
		{
			name: "provider custom with empty hosts set produces error",
			configValue: types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"provider": types.StringValue("custom"),
				"hosts":    types.SetValueMust(types.StringType, []attr.Value{}),
			}),
			expectError: "Missing Required Hosts",
		},
		{
			name: "provider custom with default spaceship hosts produces error",
			configValue: types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"provider": types.StringValue("custom"),
				"hosts": types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("launch1.spaceship.net"),
					types.StringValue("launch2.spaceship.net"),
				}),
			}),
			expectError: "Invalid Hosts Configuration",
		},
		{
			name: "provider basic with no hosts produces no errors",
			configValue: types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"provider": types.StringValue("basic"),
				"hosts":    types.SetNull(types.StringType),
			}),
			expectError: "",
		},
		{
			name: "provider basic with hosts provided produces error",
			configValue: types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"provider": types.StringValue("basic"),
				"hosts": types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1.example.com"),
				}),
			}),
			expectError: "Invalid Hosts Configuration",
		},
		{
			name: "unknown hosts set produces no errors",
			configValue: types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"provider": types.StringValue("custom"),
				"hosts":    types.SetUnknown(types.StringType),
			}),
			expectError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &nameserversValidator{}
			req := validator.ObjectRequest{
				Path:        path.Root("nameservers"),
				ConfigValue: tt.configValue,
			}
			resp := &validator.ObjectResponse{}

			v.ValidateObject(context.Background(), req, resp)

			if tt.expectError == "" {
				if resp.Diagnostics.HasError() {
					t.Errorf("expected no errors, got: %s", resp.Diagnostics.Errors())
				}
			} else {
				if !resp.Diagnostics.HasError() {
					t.Fatalf("expected error containing %q, got none", tt.expectError)
				}

				found := false
				for _, d := range resp.Diagnostics.Errors() {
					if d.Summary() == tt.expectError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error with summary %q, got: %s", tt.expectError, resp.Diagnostics.Errors())
				}
			}
		})
	}
}
