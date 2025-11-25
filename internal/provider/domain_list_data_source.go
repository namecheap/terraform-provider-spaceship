package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func NewDomainListDataSource() datasource.DataSource {
	return &domainListDataSource{}
}

type domainListDataSource struct {
	client *Client
}

func (r *domainListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_list"
}

type domainListDataSourceModel struct {
	Items []domainModel `tfsdk:"items"`
	Total types.Int64   `tfsdk:"total"`
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
	Hosts    types.List   `tfsdk:"hosts"`
	Provider types.String `tfsdk:"provider"`
}
type contacts struct {
	Registrant types.String `tfsdk:"registrant"`
	Admin      types.String `tfsdk:"admin"`
	Tech       types.String `tfsdk:"tech"`
	Billing    types.String `tfsdk:"billing"`
	Attributes types.List   `tfsdk:"attributes"`
}

func (r *domainListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data domainListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading domain list")

	var err error

	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured provider", "The Spaceship provider was not configured. Please run terraform init or configure the provider block.")
		return
	}

	response, err := r.client.GetDomainList(ctx)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read domain list",
			err.Error(),
		)
		return
	}

	data.Items = []domainModel{}
	for _, item := range response.Items {
		eppStatuses, diags := types.ListValueFrom(ctx, types.StringType, item.EPPStatuses)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		nsModel, nsDiags := flattenNameservers(ctx, item.Nameservers)
		resp.Diagnostics.Append(nsDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		contactModel, contactsDiags := flattenContacts(ctx, item.Contacts)
		resp.Diagnostics.Append(contactsDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		data.Items = append(data.Items, domainModel{
			Name:               types.StringValue(item.Name),
			UnicodeName:        types.StringValue(item.UnicodeName),
			IsPremium:          types.BoolValue(item.IsPremium),
			AutoRenew:          types.BoolValue(item.AutoRenew),
			RegistrationDate:   types.StringValue(item.RegistrationDate),
			ExpirationDate:     types.StringValue(item.ExpirationDate),
			LifecycleStatus:    types.StringValue(item.LifecycleStatus),
			VerificationStatus: stringValueOrNull(item.VerificationStatus),
			EppStatuses:        eppStatuses,
			Suspensions:        FlattenSuspensions(item.Suspensions),
			PrivacyProtection:  flattenPrivacyProtection(item.PrivacyProtection),
			Nameservers:        nsModel,
			Contacts:           contactModel,
		},
		)
	}

	data.Total = types.Int64Value(response.Total)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *domainListDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides full list of domain in account with domain details for each domain",
		Attributes: map[string]schema.Attribute{
			"total": schema.Int64Attribute{
				Computed: true,
			},
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
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
								"hosts": schema.ListAttribute{
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
					},
				},
			},
		},
	}
}

func (r *domainListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)

	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *Client, got %T", req.ProviderData))
		return
	}

	r.client = client
}

func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func FlattenSuspensions(values []ReasonCode) []suspension {
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

func flattenPrivacyProtection(pp PrivacyProtection) privacyProtection {
	return privacyProtection{
		ContactForm: types.BoolValue(pp.ContactForm),
		Level:       types.StringValue(pp.Level),
	}
}

func flattenNameservers(ctx context.Context, ns Nameservers) (nameservers, diag.Diagnostics) {
	hosts, diags := types.ListValueFrom(ctx, types.StringType, ns.Hosts)

	return nameservers{
		Provider: types.StringValue(ns.Provider),
		Hosts:    hosts,
	}, diags
}

func flattenContacts(ctx context.Context, c Contacts) (contacts, diag.Diagnostics) {
	attributes, diags := types.ListValueFrom(ctx, types.StringType, c.Attributes)

	return contacts{
		Registrant: types.StringValue(c.Registrant),
		Admin:      stringValueOrNull(c.Admin),
		Tech:       stringValueOrNull(c.Tech),
		Billing:    stringValueOrNull(c.Billing),
		Attributes: attributes,
	}, diags
}
