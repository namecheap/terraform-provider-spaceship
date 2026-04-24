package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientrecords "terraform-provider-spaceship/internal/client/records"
)

type nsRecordValidator struct{}

var _ validator.Object = &nsRecordValidator{}

func NSValidator() validator.Object {
	return &nsRecordValidator{}
}

func (v *nsRecordValidator) Description(_ context.Context) string {
	return "validates that NS records contain a valid nameserver hostname"
}

func (v *nsRecordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateObject checks NS record-specific fields only.
// Name and TTL are validated by schema-level attribute validators in the resource schema.
func (v *nsRecordValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	typeAttr, ok := attrs["type"].(types.String)
	if !ok || typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}

	if typeAttr.ValueString() != "NS" {
		return
	}

	rec := &clientrecords.NSRecord{}

	nameserverAttr, ok := attrs["nameserver"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("nameserver"),
			"Invalid Field Type",
			"The 'nameserver' field must be a string for NS records.",
		)
	} else if nameserverAttr.IsNull() || nameserverAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("nameserver"),
			"Missing Required Field",
			"The 'nameserver' field is required for NS records.",
		)
	} else {
		rec.Nameserver = nameserverAttr.ValueString()
		if err := rec.ValidateNameserver(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("nameserver"),
				"Invalid Nameserver Value",
				err.Error(),
			)
		}
	}
}
