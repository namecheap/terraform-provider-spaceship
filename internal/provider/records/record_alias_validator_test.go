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

// Spaceship silently stores an apex ALIAS (name "@") as a root CNAME, so
// Terraform can never reconcile it as an ALIAS and would recreate it on every
// apply. Reject it at plan time and point the user at the CNAME form instead.
func TestALIASValidator_RejectsApexName(t *testing.T) {
	obj := buildRecordObject(t, map[string]attr.Value{
		"type":       types.StringValue("ALIAS"),
		"name":       types.StringValue("@"),
		"alias_name": types.StringValue("target.example.com"),
	})

	resp := &validator.ObjectResponse{}
	ALIASValidator().ValidateObject(context.Background(),
		validator.ObjectRequest{Path: path.Empty(), ConfigValue: obj}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected apex ALIAS (name \"@\") to be rejected, got no error")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Summary(), "Invalid Apex ALIAS") {
		t.Errorf("unexpected diagnostic: %s", resp.Diagnostics.Errors()[0])
	}
}

// A non-apex ALIAS with a valid target round-trips fine and must pass.
func TestALIASValidator_AllowsNonApexName(t *testing.T) {
	obj := buildRecordObject(t, map[string]attr.Value{
		"type":       types.StringValue("ALIAS"),
		"name":       types.StringValue("www"),
		"alias_name": types.StringValue("target.example.com"),
	})

	resp := &validator.ObjectResponse{}
	ALIASValidator().ValidateObject(context.Background(),
		validator.ObjectRequest{Path: path.Empty(), ConfigValue: obj}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected non-apex ALIAS to pass, got: %s", resp.Diagnostics)
	}
}

// The apex check is scoped to ALIAS only — other types legitimately use "@".
func TestALIASValidator_IgnoresApexNameForOtherTypes(t *testing.T) {
	obj := buildRecordObject(t, map[string]attr.Value{
		"type":    types.StringValue("A"),
		"name":    types.StringValue("@"),
		"address": types.StringValue("192.0.2.1"),
	})

	resp := &validator.ObjectResponse{}
	ALIASValidator().ValidateObject(context.Background(),
		validator.ObjectRequest{Path: path.Empty(), ConfigValue: obj}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected non-ALIAS type to be a no-op, got: %s", resp.Diagnostics)
	}
}
