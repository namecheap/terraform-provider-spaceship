package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientrecords "terraform-provider-spaceship/internal/client/records"
)

type caaRecordValidator struct{}

var _ validator.Object = &caaRecordValidator{}

func CAAValidator() validator.Object {
	return &caaRecordValidator{}
}

func (v *caaRecordValidator) Description(_ context.Context) string {
	return "validates that CAA records contain all required fields: flag, tag, and value"
}

func (v *caaRecordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateObject checks CAA record-specific fields only.
// Name and TTL are validated by schema-level attribute validators in the resource schema.
func (v *caaRecordValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	typeAttr, ok := attrs["type"].(types.String)
	if !ok || typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}

	if typeAttr.ValueString() != "CAA" {
		return
	}

	rec := &clientrecords.CAARecord{}

	flagAttr, ok := attrs["flag"].(types.Int64)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("flag"),
			"Invalid Field Type",
			"The 'flag' field must be an integer for CAA records.",
		)
	} else if flagAttr.IsNull() || flagAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("flag"),
			"Missing Required Field",
			"The 'flag' field is required for CAA records.",
		)
	} else {
		rec.Flag = int(flagAttr.ValueInt64())
		if err := rec.ValidateFlag(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("flag"),
				"Invalid Flag Value",
				err.Error(),
			)
		}
	}

	tagAttr, ok := attrs["tag"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("tag"),
			"Invalid Field Type",
			"The 'tag' field must be a string for CAA records.",
		)
	} else if tagAttr.IsNull() || tagAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("tag"),
			"Missing Required Field",
			"The 'tag' field is required for CAA records.",
		)
	} else {
		rec.Tag = tagAttr.ValueString()
		if err := rec.ValidateTag(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("tag"),
				"Invalid Tag Value",
				err.Error(),
			)
		}
	}

	valueAttr, ok := attrs["value"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("value"),
			"Invalid Field Type",
			"The 'value' field must be a string for CAA records.",
		)
	} else if valueAttr.IsNull() || valueAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("value"),
			"Missing Required Field",
			"The 'value' field is required for CAA records.",
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
