package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientrecords "terraform-provider-spaceship/internal/client/records"
)

type txtRecordValidator struct{}

var _ validator.Object = &txtRecordValidator{}

func TXTValidator() validator.Object {
	return &txtRecordValidator{}
}

func (v *txtRecordValidator) Description(_ context.Context) string {
	return "validates that TXT records contain a value within the allowed length"
}

func (v *txtRecordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateObject checks TXT record-specific fields only.
// Name and TTL are validated by schema-level attribute validators in the resource schema.
func (v *txtRecordValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	typeAttr, ok := attrs["type"].(types.String)
	if !ok || typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}

	if typeAttr.ValueString() != "TXT" {
		return
	}

	rec := &clientrecords.TXTRecord{}

	valueAttr, ok := attrs["value"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("value"),
			"Invalid Field Type",
			"The 'value' field must be a string for TXT records.",
		)
	} else if valueAttr.IsNull() || valueAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("value"),
			"Missing Required Field",
			"The 'value' field is required for TXT records.",
		)
	} else {
		rec.Value = valueAttr.ValueString()
		if err := rec.ValidateValue(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("value"),
				"Invalid Value",
				err.Error(),
			)
		}
	}
}
