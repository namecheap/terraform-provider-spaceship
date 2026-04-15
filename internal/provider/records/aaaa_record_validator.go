package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientrecords "terraform-provider-spaceship/internal/client/records"
)

type aaaaRecordValidator struct{}

var _ validator.Object = &aaaaRecordValidator{}

func AAAAValidator() validator.Object {
	return &aaaaRecordValidator{}
}

func (v *aaaaRecordValidator) Description(_ context.Context) string {
	return "validates that AAAA records contain a valid IPv6 address"
}

func (v *aaaaRecordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateObject checks AAAA record-specific fields only.
// Name and TTL are validated by schema-level attribute validators in the resource schema.
func (v *aaaaRecordValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	typeAttr, ok := attrs["type"].(types.String)
	if !ok || typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}

	if typeAttr.ValueString() != "AAAA" {
		return
	}

	rec := &clientrecords.AAAARecord{}

	addressAttr, _ := attrs["address"].(types.String)
	if addressAttr.IsNull() || addressAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("address"),
			"Missing Required Field",
			"The 'address' field is required for AAAA records.",
		)
	} else {
		rec.Address = addressAttr.ValueString()
		if err := rec.ValidateAddress(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("address"),
				"Invalid Address Value",
				err.Error(),
			)
		}
	}
}
