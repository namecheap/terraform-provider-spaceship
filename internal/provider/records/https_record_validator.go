package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientrecords "terraform-provider-spaceship/internal/client/records"
)

type httpsRecordValidator struct{}

var _ validator.Object = &httpsRecordValidator{}

func HTTPSValidator() validator.Object {
	return &httpsRecordValidator{}
}

func (v *httpsRecordValidator) Description(_ context.Context) string {
	return "validates that HTTPS records contain all required fields: svc_priority, target_name, and svc_params; and that scheme is '_https' when port is set"
}

func (v *httpsRecordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateObject checks HTTPS record-specific fields only.
// Name and TTL are validated by schema-level attribute validators in the resource schema.
func (v *httpsRecordValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	typeAttr, ok := attrs["type"].(types.String)
	if !ok || typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}

	if typeAttr.ValueString() != "HTTPS" {
		return
	}

	rec := &clientrecords.HTTPSRecord{}

	svcPriorityAttr, ok := attrs["svc_priority"].(types.Int64)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("svc_priority"),
			"Invalid Field Type",
			"The 'svc_priority' field must be an integer for HTTPS records.",
		)
	} else if svcPriorityAttr.IsNull() || svcPriorityAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("svc_priority"),
			"Missing Required Field",
			"The 'svc_priority' field is required for HTTPS records.",
		)
	} else {
		rec.SvcPriority = int(svcPriorityAttr.ValueInt64())
		if err := rec.ValidateSvcPriority(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("svc_priority"),
				"Invalid SvcPriority Value",
				err.Error(),
			)
		}
	}

	targetNameAttr, ok := attrs["target_name"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("target_name"),
			"Invalid Field Type",
			"The 'target_name' field must be a string for HTTPS records.",
		)
	} else if targetNameAttr.IsNull() || targetNameAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("target_name"),
			"Missing Required Field",
			"The 'target_name' field is required for HTTPS records.",
		)
	} else {
		rec.TargetName = targetNameAttr.ValueString()
		if err := rec.ValidateTargetName(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("target_name"),
				"Invalid TargetName Value",
				err.Error(),
			)
		}
	}

	svcParamsAttr, ok := attrs["svc_params"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("svc_params"),
			"Invalid Field Type",
			"The 'svc_params' field must be a string for HTTPS records.",
		)
	} else if !svcParamsAttr.IsNull() && !svcParamsAttr.IsUnknown() {
		rec.SvcParams = svcParamsAttr.ValueString()
		if err := rec.ValidateSvcParams(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("svc_params"),
				"Invalid SvcParams Value",
				err.Error(),
			)
		}
	}

	portAttr, ok := attrs["port"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("port"),
			"Invalid Field Type",
			"The 'port' field must be a string for HTTPS records.",
		)
	} else if !portAttr.IsNull() && !portAttr.IsUnknown() {
		rec.Port = portAttr.ValueString()
		if err := rec.ValidatePort(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("port"),
				"Invalid Port Value",
				err.Error(),
			)
		}
	}

	schemeAttr, ok := attrs["scheme"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("scheme"),
			"Invalid Field Type",
			"The 'scheme' field must be a string for HTTPS records.",
		)
	} else {
		if !schemeAttr.IsNull() && !schemeAttr.IsUnknown() {
			rec.Scheme = schemeAttr.ValueString()
		}
		if err := rec.ValidateScheme(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("scheme"),
				"Invalid Scheme Value",
				err.Error(),
			)
		}
	}
}
