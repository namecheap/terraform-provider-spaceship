package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientrecords "terraform-provider-spaceship/internal/client/records"
)

type aliasRecordValidator struct{}

var _ validator.Object = &aliasRecordValidator{}

func ALIASValidator() validator.Object {
	return &aliasRecordValidator{}
}

func (v *aliasRecordValidator) Description(_ context.Context) string {
	return "validates that ALIAS records contain a valid alias target hostname"
}

func (v *aliasRecordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateObject checks ALIAS record-specific fields only.
// Name and TTL are validated by schema-level attribute validators in the resource schema.
func (v *aliasRecordValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	typeAttr, ok := attrs["type"].(types.String)
	if !ok || typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}

	if typeAttr.ValueString() != "ALIAS" {
		return
	}

	rec := &clientrecords.ALIASRecord{}

	aliasNameAttr, ok := attrs["alias_name"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("alias_name"),
			"Invalid Field Type",
			"The 'alias_name' field must be a string for ALIAS records.",
		)
	} else if aliasNameAttr.IsNull() || aliasNameAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("alias_name"),
			"Missing Required Field",
			"The 'alias_name' field is required for ALIAS records.",
		)
	} else {
		rec.AliasName = aliasNameAttr.ValueString()
		if err := rec.ValidateAliasName(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("alias_name"),
				"Invalid Alias Name Value",
				err.Error(),
			)
		}
	}
}
