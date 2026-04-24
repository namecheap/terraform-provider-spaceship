package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientrecords "terraform-provider-spaceship/internal/client/records"
)

type cnameRecordValidator struct{}

var _ validator.Object = &cnameRecordValidator{}

func CNAMEValidator() validator.Object {
	return &cnameRecordValidator{}
}

func (v *cnameRecordValidator) Description(_ context.Context) string {
	return "validates that CNAME records contain a valid canonical-name target"
}

func (v *cnameRecordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateObject checks CNAME record-specific fields only.
// Name and TTL are validated by schema-level attribute validators in the resource schema.
func (v *cnameRecordValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	typeAttr, ok := attrs["type"].(types.String)
	if !ok || typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}

	if typeAttr.ValueString() != "CNAME" {
		return
	}

	rec := &clientrecords.CNAMERecord{}

	cnameAttr, ok := attrs["cname"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("cname"),
			"Invalid Field Type",
			"The 'cname' field must be a string for CNAME records.",
		)
	} else if cnameAttr.IsNull() || cnameAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("cname"),
			"Missing Required Field",
			"The 'cname' field is required for CNAME records.",
		)
	} else {
		rec.CName = cnameAttr.ValueString()
		if err := rec.ValidateCName(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("cname"),
				"Invalid CName Value",
				err.Error(),
			)
		}
	}
}
