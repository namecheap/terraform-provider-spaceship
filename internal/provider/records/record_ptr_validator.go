package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientrecords "terraform-provider-spaceship/internal/client/records"
)

type ptrRecordValidator struct{}

var _ validator.Object = &ptrRecordValidator{}

func PTRValidator() validator.Object {
	return &ptrRecordValidator{}
}

func (v *ptrRecordValidator) Description(_ context.Context) string {
	return "validates that PTR records contain a valid pointer target"
}

func (v *ptrRecordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateObject checks PTR record-specific fields only.
// Name and TTL are validated by schema-level attribute validators in the resource schema.
func (v *ptrRecordValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	typeAttr, ok := attrs["type"].(types.String)
	if !ok || typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}

	if typeAttr.ValueString() != "PTR" {
		return
	}

	rec := &clientrecords.PTRRecord{}

	pointerAttr, ok := attrs["pointer"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("pointer"),
			"Invalid Field Type",
			"The 'pointer' field must be a string for PTR records.",
		)
	} else if pointerAttr.IsNull() || pointerAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("pointer"),
			"Missing Required Field",
			"The 'pointer' field is required for PTR records.",
		)
	} else {
		rec.Pointer = pointerAttr.ValueString()
		if err := rec.ValidatePointer(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("pointer"),
				"Invalid Pointer Value",
				err.Error(),
			)
		}
	}
}
