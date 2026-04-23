package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientrecords "terraform-provider-spaceship/internal/client/records"
)

type srvRecordValidator struct{}

var _ validator.Object = &srvRecordValidator{}

func SRVValidator() validator.Object {
	return &srvRecordValidator{}
}

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

	rec := &clientrecords.SRVRecord{}

	serviceAttr, ok := attrs["service"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("service"),
			"Invalid Field Type",
			"The 'service' field must be a string for SRV records.",
		)
	} else if serviceAttr.IsNull() || serviceAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("service"),
			"Missing Required Field",
			"The 'service' field is required for SRV records.",
		)
	} else {
		rec.Service = serviceAttr.ValueString()
		if err := rec.ValidateService(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("service"),
				"Invalid Service Value",
				err.Error(),
			)
		}
	}

	protocolAttr, ok := attrs["protocol"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("protocol"),
			"Invalid Field Type",
			"The 'protocol' field must be a string for SRV records.",
		)
	} else if protocolAttr.IsNull() || protocolAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("protocol"),
			"Missing Required Field",
			"The 'protocol' field is required for SRV records.",
		)
	} else {
		rec.Protocol = protocolAttr.ValueString()
		if err := rec.ValidateProtocol(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("protocol"),
				"Invalid Protocol Value",
				err.Error(),
			)
		}
	}

	priorityAttr, ok := attrs["priority"].(types.Int64)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("priority"),
			"Invalid Field Type",
			"The 'priority' field is required for SRV records and must be an integer.",
		)
	} else if priorityAttr.IsNull() || priorityAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("priority"),
			"Missing Required Field",
			"The 'priority' field is required for SRV records.",
		)
	} else {
		rec.Priority = int(priorityAttr.ValueInt64())
		if err := rec.ValidatePriority(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("priority"),
				"Invalid Priority Value",
				err.Error(),
			)
		}
	}

	weightAttr, ok := attrs["weight"].(types.Int64)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("weight"),
			"Invalid Field Type",
			"The 'weight' field must be an integer for SRV records.",
		)
	} else if weightAttr.IsNull() || weightAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("weight"),
			"Missing Required Field",
			"The 'weight' field is required for SRV records.",
		)
	} else {
		rec.Weight = int(weightAttr.ValueInt64())
		if err := rec.ValidateWeight(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("weight"),
				"Invalid Weight Value",
				err.Error(),
			)
		}
	}

	portNumberAttr, ok := attrs["port_number"].(types.Int64)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("port_number"),
			"Invalid Field Type",
			"The 'port_number' field must be an integer for SRV records.",
		)
	} else if portNumberAttr.IsNull() || portNumberAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("port_number"),
			"Missing Required Field",
			"The 'port_number' field is required for SRV records.",
		)
	} else {
		rec.Port = int(portNumberAttr.ValueInt64())
		if err := rec.ValidatePort(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("port_number"),
				"Invalid Port Value",
				err.Error(),
			)
		}
	}

	targetAttr, ok := attrs["target"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("target"),
			"Invalid Field Type",
			"The 'target' field must be a string for SRV records.",
		)
	} else if targetAttr.IsNull() || targetAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("target"),
			"Missing Required Field",
			"The 'target' field is required for SRV records.",
		)
	} else {
		rec.Target = targetAttr.ValueString()
		if err := rec.ValidateTarget(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("target"),
				"Invalid Target Value",
				err.Error(),
			)
		}
	}
}
