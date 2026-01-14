package provider

import (
	"context"
	"strings"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type nameserversValidator struct{}

var _ validator.Object = &nameserversValidator{}

func (v *nameserversValidator) Description(ctx context.Context) string {
	return "validates nameservers configuration: 'custom' provider requires hosts, 'basic' must not have hosts, and default Spaceship hosts require 'basic'"
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
	if hostsAttrSet.IsUnknown() {
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
			return
		}

		var hosts []string
		resp.Diagnostics.Append(hostsAttrSet.ElementsAs(ctx, &hosts, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if isDefaultBasicNameservers(hosts) {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("hosts"),
				"Invalid Hosts Configuration",
				"The default Spaceship nameservers can only be used with provider \"basic\".",
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

/*
This validation is needed because atm api allows setting
custom ns provider and leaving default nameservers
Bug is reported to the team
*/
func isDefaultBasicNameservers(hosts []string) bool {
	defaultHosts := client.DefaultBasicNameserverHosts()
	if len(hosts) != len(defaultHosts) {
		return false
	}

	// this is how set is made in go
	defaultSet := make(map[string]struct{}, len(defaultHosts))
	for _, host := range defaultHosts {
		defaultSet[strings.ToLower(host)] = struct{}{}
	}

	for _, host := range hosts {
		if _, ok := defaultSet[strings.ToLower(host)]; !ok {
			return false
		}
	}
	return true
}
