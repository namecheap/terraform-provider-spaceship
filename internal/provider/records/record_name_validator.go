package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	clientrecords "terraform-provider-spaceship/internal/client/records"
)

type nameValidator struct{}

var _ validator.String = &nameValidator{}

// NameValidator validates the hostname format of a record's name attribute.
// Length is enforced separately by a schema-level length validator, so this
// delegates to the client's pattern-only check to avoid duplicating the regex.
func NameValidator() validator.String {
	return &nameValidator{}
}

func (v *nameValidator) Description(_ context.Context) string {
	return "must be a valid record name (hostname format, or '@' for the zone apex)"
}

func (v *nameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *nameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if err := clientrecords.ValidateNamePattern(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Record Name",
			err.Error(),
		)
	}
}
