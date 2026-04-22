package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientrecords "terraform-provider-spaceship/internal/client/records"
)

type mxRecordValidator struct{}

var _ validator.Object = &mxRecordValidator{}

func MXValidator() validator.Object {
	return &mxRecordValidator{}
}

func (v *mxRecordValidator) Description(_ context.Context) string {
	return "validates that MX records contain all required fields: exchange and preference"
}

func (v *mxRecordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateObject checks MX record-specific fields only.
// Name and TTL are validated by schema-level attribute validators in the resource schema.
func (v *mxRecordValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	typeAttr, ok := attrs["type"].(types.String)
	if !ok || typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}

	if typeAttr.ValueString() != "MX" {
		return
	}

	rec := &clientrecords.MXRecord{}

	exchangeAttr, ok := attrs["exchange"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("exchange"),
			"Invalid Field Type",
			"The 'exchange' field must be a string for MX records.",
		)
	} else if exchangeAttr.IsNull() || exchangeAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("exchange"),
			"Missing Required Field",
			"The 'exchange' field is required for MX records.",
		)
	} else {
		rec.Exchange = exchangeAttr.ValueString()
		if err := rec.ValidateExchange(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("exchange"),
				"Invalid Exchange Value",
				err.Error(),
			)
		}
	}

	preferenceAttr, ok := attrs["preference"].(types.Int64)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("preference"),
			"Invalid Field Type",
			"The 'preference' field must be an integer for MX records.",
		)
	} else if preferenceAttr.IsNull() || preferenceAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("preference"),
			"Missing Required Field",
			"The 'preference' field is required for MX records.",
		)
	} else {
		rec.Preference = int(preferenceAttr.ValueInt64())
		if err := rec.ValidatePreference(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("preference"),
				"Invalid Preference Value",
				err.Error(),
			)
		}
	}
}
