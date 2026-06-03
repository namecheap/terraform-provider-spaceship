package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// deprecatedBoolWarning is a validator that emits a warning when a deprecated
// bool attribute is explicitly configured.
type deprecatedBoolWarning struct {
	message string
}

func deprecatedBoolValidator(message string) validator.Bool {
	return deprecatedBoolWarning{message: message}
}

func (v deprecatedBoolWarning) Description(_ context.Context) string {
	return v.message
}

func (v deprecatedBoolWarning) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v deprecatedBoolWarning) ValidateBool(_ context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	resp.Diagnostics.Append(diag.NewAttributeWarningDiagnostic(
		req.Path,
		"Deprecated Attribute",
		v.message,
	))
}
