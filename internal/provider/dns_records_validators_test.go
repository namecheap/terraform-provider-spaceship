package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRecordNameValidator_ReturnsNonNil(t *testing.T) {
	v := recordNameValidator()
	if v == nil {
		t.Fatal("recordNameValidator() returned nil")
	}
}

func TestRecordNameValidator_Description(t *testing.T) {
	v := recordNameValidator()
	ctx := context.Background()

	desc := v.Description(ctx)
	if desc == "" {
		t.Error("Description() returned empty string")
	}

	md := v.MarkdownDescription(ctx)
	if md == "" {
		t.Error("MarkdownDescription() returned empty string")
	}
}

func TestRecordNameValidator_ValidateString(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		value       types.String
		expectError bool
	}{
		{name: "null value", value: types.StringNull(), expectError: false},
		{name: "unknown value", value: types.StringUnknown(), expectError: false},
		{name: "at sign", value: types.StringValue("@"), expectError: false},
		{name: "wildcard", value: types.StringValue("*"), expectError: false},
		{name: "simple host", value: types.StringValue("myhost"), expectError: false},
		{name: "subdomain", value: types.StringValue("sub.domain"), expectError: false},
		{name: "dmarc record", value: types.StringValue("_dmarc"), expectError: false},
		{name: "acme challenge subdomain", value: types.StringValue("_acme-challenge.sub"), expectError: false},
		{name: "wildcard prefix", value: types.StringValue("*.example"), expectError: false},
		{name: "underscore prefix", value: types.StringValue("_.example"), expectError: false},
		{name: "leading dot invalid", value: types.StringValue(".invalid"), expectError: true},
		{name: "empty string", value: types.StringValue(""), expectError: true},
		{name: "dot only", value: types.StringValue("."), expectError: true},
		{name: "trailing dot", value: types.StringValue("host."), expectError: true},
		{name: "hyphen start label", value: types.StringValue("-bad"), expectError: true},
		{name: "hyphen end label", value: types.StringValue("bad-"), expectError: true},
	}

	v := recordNameValidator()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: tc.value,
			}
			resp := &validator.StringResponse{}

			v.ValidateString(ctx, req, resp)

			if tc.expectError && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for value %q but got none", tc.value)
			}
			if !tc.expectError && resp.Diagnostics.HasError() {
				t.Errorf("expected no error for value %q but got: %s", tc.value, resp.Diagnostics.Errors())
			}
		})
	}
}

func TestDeprecatedBoolValidator_ReturnsNonNil(t *testing.T) {
	v := deprecatedBoolValidator("some message")
	if v == nil {
		t.Fatal("deprecatedBoolValidator() returned nil")
	}
}

func TestDeprecatedBoolValidator_Description(t *testing.T) {
	msg := "this attribute is deprecated"
	v := deprecatedBoolValidator(msg)
	ctx := context.Background()

	desc := v.Description(ctx)
	if desc != msg {
		t.Errorf("Description() = %q, want %q", desc, msg)
	}

	md := v.MarkdownDescription(ctx)
	if md != msg {
		t.Errorf("MarkdownDescription() = %q, want %q", md, msg)
	}
}

func TestDeprecatedBoolValidator_ValidateBool(t *testing.T) {
	ctx := context.Background()
	msg := "some deprecation message"

	tests := []struct {
		name            string
		value           types.Bool
		expectWarning   bool
		expectNoWarning bool
	}{
		{name: "null value", value: types.BoolNull(), expectNoWarning: true},
		{name: "unknown value", value: types.BoolUnknown(), expectNoWarning: true},
		{name: "true value", value: types.BoolValue(true), expectWarning: true},
		{name: "false value", value: types.BoolValue(false), expectWarning: true},
	}

	v := deprecatedBoolValidator(msg)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.BoolRequest{
				Path:        path.Root("test"),
				ConfigValue: tc.value,
			}
			resp := &validator.BoolResponse{}

			v.ValidateBool(ctx, req, resp)

			if resp.Diagnostics.HasError() {
				t.Errorf("expected no errors but got: %s", resp.Diagnostics.Errors())
			}

			hasWarning := resp.Diagnostics.WarningsCount() > 0

			if tc.expectWarning && !hasWarning {
				t.Error("expected a warning diagnostic but got none")
			}
			if tc.expectNoWarning && hasWarning {
				t.Error("expected no warning diagnostic but got one")
			}

			if tc.expectWarning {
				found := false
				for _, d := range resp.Diagnostics {
					if d.Severity() == diag.SeverityWarning && d.Summary() == "Deprecated Attribute" {
						found = true
						break
					}
				}
				if !found {
					t.Error("expected warning with summary 'Deprecated Attribute' not found")
				}
			}
		})
	}
}
