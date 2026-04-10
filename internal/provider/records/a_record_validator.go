package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientrecords "terraform-provider-spaceship/internal/client/records"
)

type aRecordValidator struct{}

var _ validator.Object = &aRecordValidator{}

func AValidator() validator.Object {
	return &aRecordValidator{}
}

func (v *aRecordValidator) Description(_ context.Context) string {
	return "validates that A records contain a valid IPv4 address"
}

func (v *aRecordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *aRecordValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	typeAttr, ok := attrs["type"].(types.String)
	if !ok || typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}

	if typeAttr.ValueString() != "A" {
		return
	}

	rec := &clientrecords.ARecord{}

	addressAttr, _ := attrs["address"].(types.String)
	if addressAttr.IsNull() || addressAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("address"),
			"Missing Required Field",
			"The 'address' field is required for A records.",
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
