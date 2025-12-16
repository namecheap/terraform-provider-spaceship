package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type nameserversValidator struct{}

var _ validator.Object = &nameserversValidator{}

func (v *nameserversValidator) Description(ctx context.Context) string {
	return "validates nameservers configuration: 'custom' provider requires hosts, 'basic' must not have hosts"
}

func (v *nameserversValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *nameserversValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attr := req.ConfigValue.Attributes()

	providerAttr, ok := attr["provider"].(types.String)
	if !ok || providerAttr.IsNull() || providerAttr.IsUnknown() {
		return
	}

	hostsAttrSet, ok := attr["hosts"].(types.Set)
	if !ok {
		return
	}

	provider := providerAttr.ValueString()
	hostsIsEmpty := hostsAttrSet.IsNull() || len(hostsAttrSet.Elements()) == 0

	switch provider {
	case "custom":
		if hostsIsEmpty {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("hosts"),
				"Missing Required Hosts",
				"The 'hosts' field is required when provider is 'custom'.",
			)
		}
	case "basic":
		if !hostsIsEmpty {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("hosts"),
				"Invalid Hosts Configuration",
				"The 'hosts' field must be omitted when provider is 'basic'.",
			)
		}
	}
}
