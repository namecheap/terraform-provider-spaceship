package records

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type ipAddressValidator struct{}

var _ validator.String = &ipAddressValidator{}

// IPAddressValidator validates that a string is a valid IPv4 or IPv6 address.
// Compose it with setvalidator.ValueStringsAre to validate each element of an
// IP-address set at plan time, mirroring the client's net.ParseIP check.
func IPAddressValidator() validator.String {
	return &ipAddressValidator{}
}

func (v *ipAddressValidator) Description(_ context.Context) string {
	return "validates that the value is a valid IPv4 or IPv6 address"
}

func (v *ipAddressValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString reports a diagnostic when the value does not parse as an IP.
// Null/unknown values pass: an address fed by a not-yet-known output must not
// fail at plan time when the eventual value may be valid.
func (v *ipAddressValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if net.ParseIP(value) == nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid IP Address",
			fmt.Sprintf("must be a valid IP address, got %q", value),
		)
	}
}
