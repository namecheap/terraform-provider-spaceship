package records

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// A malformed address must surface a diagnostic so terraform plan catches it
// before apply.
func TestIPAddressValidator_RejectsMalformed(t *testing.T) {
	resp := &validator.StringResponse{}
	IPAddressValidator().ValidateString(t.Context(),
		validator.StringRequest{ConfigValue: types.StringValue("not-an-ip")}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected validation error for malformed IP, got none")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Summary(), "Invalid IP Address") {
		t.Errorf("unexpected diagnostic: %s", resp.Diagnostics.Errors()[0])
	}
}

// Both IPv4 and IPv6 are accepted, matching the field's documented contract.
func TestIPAddressValidator_AllowsIPv4AndIPv6(t *testing.T) {
	for _, ip := range []string{"192.0.2.1", "2001:db8::1"} {
		resp := &validator.StringResponse{}
		IPAddressValidator().ValidateString(t.Context(),
			validator.StringRequest{ConfigValue: types.StringValue(ip)}, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("expected %q to validate, got: %s", ip, resp.Diagnostics)
		}
	}
}

// Null/unknown values must not error — an IP sourced from a not-yet-known
// output would otherwise fail plan even when the eventual value is valid.
func TestIPAddressValidator_IgnoresNullAndUnknown(t *testing.T) {
	for _, v := range []types.String{types.StringNull(), types.StringUnknown()} {
		resp := &validator.StringResponse{}
		IPAddressValidator().ValidateString(t.Context(),
			validator.StringRequest{ConfigValue: v}, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("expected no error for %v, got: %s", v, resp.Diagnostics)
		}
	}
}
