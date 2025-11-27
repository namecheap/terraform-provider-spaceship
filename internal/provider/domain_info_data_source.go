package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewDomainInfoDataSource() datasource.DataSource {
	return &domainInfoDataSource{}
}

type domainInfoDataSource struct {
	client *Client
}

func (d *domainInfoDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_info"
}

func (d *domainInfoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data domainInfoModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	domain := data.Domain.ValueString()

	response, err := d.client.GetDomainInfo(ctx, domain)

	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain info", err.Error())
		return
	}

	eppStatuses, eppDiags := types.ListValueFrom(ctx, types.StringType, response.EPPStatuses)
	resp.Diagnostics.Append(eppDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ppModel := flattenPrivacyProtection(response.PrivacyProtection)
	nsModel, nsDiags := flattenNameservers(ctx, response.Nameservers)
	resp.Diagnostics.Append(nsDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	contactModel, contactDiags := flattenContacts(ctx, response.Contacts)
	resp.Diagnostics.Append(contactDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Name = types.StringValue(response.Name)
	data.UnicodeName = types.StringValue(response.UnicodeName)
	data.IsPremium = types.BoolValue(response.IsPremium)
	data.AutoRenew = types.BoolValue(response.AutoRenew)
	data.RegistrationDate = types.StringValue(response.RegistrationDate)
	data.ExpirationDate = types.StringValue(response.ExpirationDate)
	data.LifecycleStatus = types.StringValue(response.LifecycleStatus)
	data.VerificationStatus = stringValueOrNull(response.VerificationStatus)
	data.EppStatuses = eppStatuses
	data.PrivacyProtection = &ppModel
	data.Nameservers = &nsModel
	data.Contacts = &contactModel
	data.Suspensions = flattenSuspensions(response.Suspensions)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (d *domainInfoDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get all info about single domain",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(4, 255),
				},
			},
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
	}
}

func (d *domainInfoDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *Client, got %T", req.ProviderData))
		return
	}

	d.client = client
}

type domainInfoModel struct {
	Domain types.String `tfsdk:"domain"`

	Name               types.String       `tfsdk:"name"`
	UnicodeName        types.String       `tfsdk:"unicode_name"`
	IsPremium          types.Bool         `tfsdk:"is_premium"`
	AutoRenew          types.Bool         `tfsdk:"auto_renew"`
	RegistrationDate   types.String       `tfsdk:"registration_date"`
	ExpirationDate     types.String       `tfsdk:"expiration_date"`
	LifecycleStatus    types.String       `tfsdk:"lifecycle_status"`
	VerificationStatus types.String       `tfsdk:"verification_status"`
	EppStatuses        types.List         `tfsdk:"epp_statuses"`
	Suspensions        []suspension       `tfsdk:"suspensions"`
	PrivacyProtection  *privacyProtection `tfsdk:"privacy_protection"`
	Nameservers        *nameservers       `tfsdk:"nameservers"`
	Contacts           *contacts          `tfsdk:"contacts"`
}
