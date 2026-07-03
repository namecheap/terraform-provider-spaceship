package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientrecords "github.com/namecheap/go-spaceship-sdk/client/records"
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

	// Spaceship stores an apex ALIAS (name "@") as a root CNAME, not an ALIAS.
	// Terraform matches records by type+name+data, so the record read back never
	// matches the declared ALIAS and the provider recreates it on every apply.
	// Reject early and point the user at the CNAME form the API actually keeps.
	if nameAttr, ok := attrs["name"].(types.String); ok && !nameAttr.IsNull() && !nameAttr.IsUnknown() && nameAttr.ValueString() == "@" {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("name"),
			"Invalid Apex ALIAS Record",
			`Spaceship stores an apex ALIAS (name "@") as a root CNAME, so Terraform cannot manage it as an ALIAS record (it would be recreated on every apply). `+
				`Declare it as a CNAME instead: type = "CNAME", name = "@", cname = "<target>".`,
		)
		return
	}

	aliasNameAttr, ok := attrs["alias_name"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("alias_name"),
			"Invalid Field Type",
			"The 'alias_name' field must be a string for ALIAS records.",
		)
		return
	}
	if aliasNameAttr.IsNull() || aliasNameAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("alias_name"),
			"Missing Required Field",
			"The 'alias_name' field is required for ALIAS records.",
		)
		return
	}

	if err := (&clientrecords.ALIASRecord{AliasName: aliasNameAttr.ValueString()}).ValidateAliasName(); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("alias_name"),
			"Invalid Alias Name Value",
			err.Error(),
		)
	}
}
