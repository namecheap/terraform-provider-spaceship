package provider

import (
	"context"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func flattenSuspensions(values []client.ReasonCode) []suspension {
	if len(values) == 0 {
		return []suspension{}
	}

	out := make([]suspension, len(values))

	for i, s := range values {
		out[i] = suspension{
			ReasonCode: stringValueOrNull(s.ReasonCode),
		}
	}
	return out
}

func flattenPrivacyProtection(pp client.PrivacyProtection) privacyProtection {
	return privacyProtection{
		ContactForm: types.BoolValue(pp.ContactForm),
		Level:       types.StringValue(pp.Level),
	}
}

func flattenNameservers(ctx context.Context, ns client.Nameservers) (nameservers, diag.Diagnostics) {
	hosts, diags := types.SetValueFrom(ctx, types.StringType, ns.Hosts)

	return nameservers{
		Provider: types.StringValue(ns.Provider),
		Hosts:    hosts,
	}, diags
}

func flattenContacts(ctx context.Context, c client.Contacts) (contacts, diag.Diagnostics) {
	attributes, diags := types.ListValueFrom(ctx, types.StringType, c.Attributes)

	return contacts{
		Registrant: types.StringValue(c.Registrant),
		Admin:      stringValueOrNull(c.Admin),
		Tech:       stringValueOrNull(c.Tech),
		Billing:    stringValueOrNull(c.Billing),
		Attributes: attributes,
	}, diags
}

func domainAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name":              schema.StringAttribute{Computed: true},
		"unicode_name":      schema.StringAttribute{Computed: true},
		"is_premium":        schema.BoolAttribute{Computed: true},
		"auto_renew":        schema.BoolAttribute{Computed: true},
		"registration_date": schema.StringAttribute{Computed: true},
		"expiration_date":   schema.StringAttribute{Computed: true},
		"lifecycle_status": schema.StringAttribute{
			Computed:    true,
			Description: "Lifecycle phase. One of creating, registered, grace1, grace2, redemption.",
		},
		"verification_status": schema.StringAttribute{
			Computed:    true,
			Optional:    true,
			Description: "Status of the RAA verification process. One of verification, success, failed. Null when not applicable.",
			Validators: []validator.String{
				stringvalidator.OneOf("verification", "success", "failed"),
			},
		},

		"epp_statuses": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Possible values clientDeleteProhibited clientHold clientRenewProhibited clientTransferProhibited clientUpdateProhibited",
		},

		"suspensions": schema.ListNestedAttribute{
			Computed:    true,
			Description: "Information about domain suspensions. May contain up to 2 items.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"reason_code": schema.StringAttribute{
						Computed:    true,
						Description: "Suspension reason code (raaVerification, abuse, promoAbuse, fraud, pendingAccountVerification, unauthorizedAccess, tosViolation, transferDispute, restrictedSecurity, lockCourt, suspendCourt, udrpUrs, restrictedLegal, paymentPending, unpaidService, restrictedWhois, lockedWhois)",
					},
				},
			},
		},

		"privacy_protection": schema.SingleNestedAttribute{
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"contact_form": schema.BoolAttribute{
					Computed:    true,
					Description: "Indicates whether WHOIS should display the contact form link",
				},
				"level": schema.StringAttribute{
					Computed:    true,
					Description: "Privacy level: public or high",
					Validators: []validator.String{
						stringvalidator.OneOf("public", "high"),
					},
				},
			},
		},
		"nameservers": schema.SingleNestedAttribute{
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"provider": schema.StringAttribute{
					Computed:    true,
					Description: "type: basic or custom",
					Validators: []validator.String{
						stringvalidator.OneOf("basic", "custom"),
					},
				},
				"hosts": schema.SetAttribute{
					Computed:    true,
					ElementType: types.StringType,
				},
			},
		},
		"contacts": schema.SingleNestedAttribute{
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"registrant": schema.StringAttribute{
					Computed:    true,
					Description: "Always present registrant handle.",
				},
				"admin": schema.StringAttribute{
					Computed:    true,
					Description: "Administrative contact handle when provided.",
				},
				"tech": schema.StringAttribute{
					Computed:    true,
					Description: "Technical contact handle when provided.",
				},
				"billing": schema.StringAttribute{
					Computed:    true,
					Description: "Billing contact handle when provided.",
				},
				"attributes": schema.ListAttribute{
					Computed:    true,
					ElementType: types.StringType,
					Optional:    true,
					Description: "Optional list of contact attributes supplied by Spaceship.",
				},
			},
		},
	}
}

func buildDomainModel(ctx context.Context, info client.DomainInfo) (domainModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	eppStatuses, eppDiags := types.ListValueFrom(ctx, types.StringType, info.EPPStatuses)
	diags.Append(eppDiags...)
	if diags.HasError() {
		return domainModel{}, diags
	}

	nsModel, nsDiags := flattenNameservers(ctx, info.Nameservers)
	diags.Append(nsDiags...)
	if diags.HasError() {
		return domainModel{}, diags
	}

	contactModel, contactDiags := flattenContacts(ctx, info.Contacts)
	diags.Append(contactDiags...)
	if diags.HasError() {
		return domainModel{}, diags
	}

	return domainModel{
		Name:               types.StringValue(info.Name),
		UnicodeName:        types.StringValue(info.UnicodeName),
		IsPremium:          types.BoolValue(info.IsPremium),
		AutoRenew:          types.BoolValue(info.AutoRenew),
		RegistrationDate:   types.StringValue(info.RegistrationDate),
		ExpirationDate:     types.StringValue(info.ExpirationDate),
		LifecycleStatus:    types.StringValue(info.LifecycleStatus),
		VerificationStatus: stringValueOrNull(info.VerificationStatus),
		EppStatuses:        eppStatuses,
		Suspensions:        flattenSuspensions(info.Suspensions),
		PrivacyProtection:  flattenPrivacyProtection(info.PrivacyProtection),
		Nameservers:        nsModel,
		Contacts:           contactModel,
	}, diags
}

type domainModel struct {
	Name               types.String      `tfsdk:"name"`
	UnicodeName        types.String      `tfsdk:"unicode_name"`
	IsPremium          types.Bool        `tfsdk:"is_premium"`
	AutoRenew          types.Bool        `tfsdk:"auto_renew"`
	RegistrationDate   types.String      `tfsdk:"registration_date"`
	ExpirationDate     types.String      `tfsdk:"expiration_date"`
	LifecycleStatus    types.String      `tfsdk:"lifecycle_status"`
	VerificationStatus types.String      `tfsdk:"verification_status"`
	EppStatuses        types.List        `tfsdk:"epp_statuses"`
	Suspensions        []suspension      `tfsdk:"suspensions"`
	PrivacyProtection  privacyProtection `tfsdk:"privacy_protection"`
	Nameservers        nameservers       `tfsdk:"nameservers"`
	Contacts           contacts          `tfsdk:"contacts"`
}

type suspension struct {
	ReasonCode types.String `tfsdk:"reason_code"`
}
type privacyProtection struct {
	ContactForm types.Bool   `tfsdk:"contact_form"`
	Level       types.String `tfsdk:"level"`
}
type nameservers struct {
	Hosts    types.Set    `tfsdk:"hosts"`
	Provider types.String `tfsdk:"provider"`
}
type contacts struct {
	Registrant types.String `tfsdk:"registrant"`
	Admin      types.String `tfsdk:"admin"`
	Tech       types.String `tfsdk:"tech"`
	Billing    types.String `tfsdk:"billing"`
	Attributes types.List   `tfsdk:"attributes"`
}

type domainInfoModel struct {
	Domain types.String `tfsdk:"domain"`
	domainModel
}
