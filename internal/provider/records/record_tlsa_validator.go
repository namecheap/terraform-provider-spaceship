package records

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientrecords "github.com/namecheap/go-spaceship-sdk/client/records"
)

type tlsaRecordValidator struct{}

var _ validator.Object = &tlsaRecordValidator{}

func TLSAValidator() validator.Object {
	return &tlsaRecordValidator{}
}

func (v *tlsaRecordValidator) Description(_ context.Context) string {
	return "validates that TLSA records contain all required fields: port, protocol, usage, selector, matching, and association_data"
}

func (v *tlsaRecordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateObject checks TLSA record-specific fields only.
// Name and TTL are validated by schema-level attribute validators in the resource schema.
func (v *tlsaRecordValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	typeAttr, ok := attrs["type"].(types.String)
	if !ok || typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}

	if typeAttr.ValueString() != "TLSA" {
		return
	}

	rec := &clientrecords.TLSARecord{}

	portAttr, ok := attrs["port"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("port"),
			"Invalid Field Type",
			"The 'port' field must be a string for TLSA records.",
		)
	} else if portAttr.IsNull() || portAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("port"),
			"Missing Required Field",
			"The 'port' field is required for TLSA records.",
		)
	} else {
		rec.Port = portAttr.ValueString()
		if err := rec.ValidatePort(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("port"),
				"Invalid Port Value",
				err.Error(),
			)
		}
	}

	protocolAttr, ok := attrs["protocol"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("protocol"),
			"Invalid Field Type",
			"The 'protocol' field must be a string for TLSA records.",
		)
	} else if protocolAttr.IsNull() || protocolAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("protocol"),
			"Missing Required Field",
			"The 'protocol' field is required for TLSA records.",
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

	usageAttr, ok := attrs["usage"].(types.Int64)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("usage"),
			"Invalid Field Type",
			"The 'usage' field must be an integer for TLSA records.",
		)
	} else if usageAttr.IsNull() || usageAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("usage"),
			"Missing Required Field",
			"The 'usage' field is required for TLSA records.",
		)
	} else {
		rec.Usage = int(usageAttr.ValueInt64())
		if err := rec.ValidateUsage(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("usage"),
				"Invalid Usage Value",
				err.Error(),
			)
		}
	}

	selectorAttr, ok := attrs["selector"].(types.Int64)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("selector"),
			"Invalid Field Type",
			"The 'selector' field must be an integer for TLSA records.",
		)
	} else if selectorAttr.IsNull() || selectorAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("selector"),
			"Missing Required Field",
			"The 'selector' field is required for TLSA records.",
		)
	} else {
		rec.Selector = int(selectorAttr.ValueInt64())
		if err := rec.ValidateSelector(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("selector"),
				"Invalid Selector Value",
				err.Error(),
			)
		}
	}

	matchingAttr, ok := attrs["matching"].(types.Int64)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("matching"),
			"Invalid Field Type",
			"The 'matching' field must be an integer for TLSA records.",
		)
	} else if matchingAttr.IsNull() || matchingAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("matching"),
			"Missing Required Field",
			"The 'matching' field is required for TLSA records.",
		)
	} else {
		rec.Matching = int(matchingAttr.ValueInt64())
		if err := rec.ValidateMatching(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("matching"),
				"Invalid Matching Value",
				err.Error(),
			)
		}
	}

	assocAttr, ok := attrs["association_data"].(types.String)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("association_data"),
			"Invalid Field Type",
			"The 'association_data' field must be a string for TLSA records.",
		)
	} else if assocAttr.IsNull() || assocAttr.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("association_data"),
			"Missing Required Field",
			"The 'association_data' field is required for TLSA records.",
		)
	} else {
		rec.AssociationData = assocAttr.ValueString()
		if err := rec.ValidateAssociationData(); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("association_data"),
				"Invalid AssociationData Value",
				err.Error(),
			)
		}
	}
}
