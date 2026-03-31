package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var srvServicePattern = regexp.MustCompile(`^_[a-zA-Z0-9-]+$`)

type srvRecordValidator struct{}

var _ validator.Object = &srvRecordValidator{}

func (v *srvRecordValidator) Description(_ context.Context) string {
	return "validates that SRV records contain all required fields: service, protocol, priority, weight, port_number, and target"
}

func (v *srvRecordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *srvRecordValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	typeAttr, ok := attrs["type"].(types.String)
	if !ok || typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}

	if typeAttr.ValueString() != "SRV" {
		return
	}

	// Validate service: required, 2-63 chars, pattern _[a-zA-Z0-9-]+
	serviceAttr, _ := attrs["service"].(types.String)
	if serviceAttr.IsNull() || serviceAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("service"),
			"Missing Required Field",
			"The 'service' field is required for SRV records.",
		)
	} else {
		val := serviceAttr.ValueString()
		if len(val) < 2 || len(val) > 63 {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("service"),
				"Invalid Service Value",
				"The 'service' field must be between 2 and 63 characters for SRV records.",
			)
		} else if !srvServicePattern.MatchString(val) {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("service"),
				"Invalid Service Format",
				"The 'service' field must start with '_' and contain only alphanumeric characters or hyphens (e.g. '_sip', '_ldap').",
			)
		}
	}

	// Validate protocol: required, 2-63 chars, pattern _[a-zA-Z0-9-]+
	protocolAttr, _ := attrs["protocol"].(types.String)
	if protocolAttr.IsNull() || protocolAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("protocol"),
			"Missing Required Field",
			"The 'protocol' field is required for SRV records.",
		)
	} else {
		val := protocolAttr.ValueString()
		if len(val) < 2 || len(val) > 63 {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("protocol"),
				"Invalid Protocol Value",
				"The 'protocol' field must be between 2 and 63 characters for SRV records.",
			)
		} else if !srvServicePattern.MatchString(val) {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("protocol"),
				"Invalid Protocol Format",
				"The 'protocol' field must start with '_' and contain only alphanumeric characters or hyphens (e.g. '_tcp', '_udp').",
			)
		}
	}

	// Validate priority: required, 0-65535
	priorityAttr, _ := attrs["priority"].(types.Int64)
	if priorityAttr.IsNull() || priorityAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("priority"),
			"Missing Required Field",
			"The 'priority' field is required for SRV records.",
		)
	}

	// Validate weight: required, 0-65535
	weightAttr, _ := attrs["weight"].(types.Int64)
	if weightAttr.IsNull() || weightAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("weight"),
			"Missing Required Field",
			"The 'weight' field is required for SRV records.",
		)
	}

	// Validate port_number: required, 1-65535
	portNumberAttr, _ := attrs["port_number"].(types.Int64)
	if portNumberAttr.IsNull() || portNumberAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("port_number"),
			"Missing Required Field",
			"The 'port_number' field is required for SRV records.",
		)
	}

	// Validate target: required, 1-253 chars
	targetAttr, _ := attrs["target"].(types.String)
	if targetAttr.IsNull() || targetAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("target"),
			"Missing Required Field",
			"The 'target' field is required for SRV records.",
		)
	} else {
		val := targetAttr.ValueString()
		if len(val) < 1 || len(val) > 253 {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("target"),
				"Invalid Target Value",
				"The 'target' field must be between 1 and 253 characters for SRV records.",
			)
		}
	}
}
