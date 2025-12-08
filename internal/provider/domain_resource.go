package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func NewDomainResource() resource.Resource {
	return &domainResource{}
}

type domainResource struct {
	client *Client
}

func (d *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *domainResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages domain settings for Spaceship domain",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "indicate domain which you want to manage",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1), // TODO could be improved by valid regex
				},
			},
			"auto_renew": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},

			"name":              schema.StringAttribute{Computed: true},
			"unicode_name":      schema.StringAttribute{Computed: true},
			"is_premium":        schema.BoolAttribute{Computed: true},
			"registration_date": schema.StringAttribute{Computed: true},
			"expiration_date":   schema.StringAttribute{Computed: true},
			"lifecycle_status": schema.StringAttribute{
				Computed:    true,
				Description: "Lifecycle phase. One of creating, registered, grace1, grace2, redemption.",
			},

			"verification_status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the RAA verification process. One of verification, success, failed. Null when not applicable.",
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

			/*
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
			*/
		},
	}
}

func (d *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainInfo, err := d.client.GetDomainInfo(ctx, state.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain info", err.Error())
		return
	}

	state.AutoRenew = types.BoolValue(domainInfo.AutoRenew)
	state.Name = types.StringValue(domainInfo.Name)
	state.UnicodeName = types.StringValue(domainInfo.UnicodeName)
	state.IsPremium = types.BoolValue(domainInfo.IsPremium)
	state.RegistrationDate = types.StringValue(domainInfo.RegistrationDate)
	state.ExpirationDate = types.StringValue(domainInfo.ExpirationDate)
	state.LifecycleStatus = types.StringValue(domainInfo.LifecycleStatus)
	state.VerificationStatus = types.StringValue(domainInfo.VerificationStatus)

	tflog.Debug(ctx, "API response", map[string]any{
		"epp_statuses":      domainInfo.EPPStatuses,
		"epp_statuses_type": fmt.Sprintf("%T", domainInfo.EPPStatuses),
		"epp_statuses_len":  len(domainInfo.EPPStatuses),
	})

	eppStatuses, _ := types.ListValueFrom(ctx, types.StringType, domainInfo.EPPStatuses)
	state.EppStatuses = eppStatuses

	tflog.Debug(ctx, "API response", map[string]any{
		"suspensions":      domainInfo.Suspensions,
		"suspensions_type": fmt.Sprintf("%T", domainInfo.Suspensions),
		"suspensions_len":  len(domainInfo.Suspensions),
	})

	suspensions, diags := SuspensionsToTerraformList(ctx, domainInfo.Suspensions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Suspensions = suspensions

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (d *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (d *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var plan domainResourceModel
	var domainInfo DomainInfo
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainInfo, err := d.client.GetDomainInfo(ctx, plan.Domain.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain info", err.Error())
		return
	}

	var state domainResourceModel

	state.AutoRenew = types.BoolValue(domainInfo.AutoRenew)
	state.Domain = types.StringValue(domainInfo.Name)
	state.Name = types.StringValue(domainInfo.Name)
	state.UnicodeName = types.StringValue(domainInfo.UnicodeName)
	state.IsPremium = types.BoolValue(domainInfo.IsPremium)
	state.RegistrationDate = types.StringValue(domainInfo.RegistrationDate)
	state.ExpirationDate = types.StringValue(domainInfo.ExpirationDate)
	state.LifecycleStatus = types.StringValue(domainInfo.LifecycleStatus)
	state.VerificationStatus = types.StringValue(domainInfo.VerificationStatus)

	tflog.Debug(ctx, "API response", map[string]any{
		"epp_statuses":      domainInfo.EPPStatuses,
		"epp_statuses_type": fmt.Sprintf("%T", domainInfo.EPPStatuses),
		"epp_statuses_len":  len(domainInfo.EPPStatuses),
	})

	eppStatuses, _ := types.ListValueFrom(ctx, types.StringType, domainInfo.EPPStatuses)
	state.EppStatuses = eppStatuses

	suspensions, diags := SuspensionsToTerraformList(ctx, domainInfo.Suspensions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Suspensions = suspensions

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (d *domainResource) Delete(_ context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// todo should be done as removal from state only
	// no external call
	// leaving infra in the same state

}

func (d *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "calling domain autorenewal update with domain %s", map[string]any{
		"domain": plan.Domain.String(),
	},
	)

	_, err := d.client.UpdateAutoRenew(ctx, plan.Domain.ValueString(), plan.AutoRenew.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating domain auto renew",
			fmt.Sprintf("Could not update auto_renew for domain %s: %s", plan.Domain.String(), err),
		)
		return
	}

	domainInfo, err := d.client.GetDomainInfo(ctx, plan.Domain.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain info", err.Error())
	}

	var state domainResourceModel

	state.AutoRenew = types.BoolValue(domainInfo.AutoRenew)
	state.Domain = types.StringValue(domainInfo.Name)
	state.Name = types.StringValue(domainInfo.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type domainResourceModel struct {
	Domain types.String `tfsdk:"domain"`

	//configurable
	AutoRenew types.Bool `tfsdk:"auto_renew"`

	// Nameservers       basetypes.ObjectValue `tfsdk:"nameservers"`
	// PrivacyProtection basetypes.ObjectValue `tfsdk:"privacy_protection"`

	//read only
	Name               types.String `tfsdk:"name"`
	UnicodeName        types.String `tfsdk:"unicode_name"`
	IsPremium          types.Bool   `tfsdk:"is_premium"`
	RegistrationDate   types.String `tfsdk:"registration_date"`
	ExpirationDate     types.String `tfsdk:"expiration_date"`
	LifecycleStatus    types.String `tfsdk:"lifecycle_status"`
	VerificationStatus types.String `tfsdk:"verification_status"`
	EppStatuses        types.List   `tfsdk:"epp_statuses"`
	Suspensions        types.List   `tfsdk:"suspensions"`
	// Contacts           basetypes.ObjectValue `tfsdk:"contacts"`
}

func SuspensionsToTerraformList(ctx context.Context, suspensions []ReasonCode) (types.List, diag.Diagnostics) {
	suspensionAttrTypes := map[string]attr.Type{
		"reason_code": types.StringType,
	}

	suspensionObjectType := types.ObjectType{AttrTypes: suspensionAttrTypes}

	if suspensions == nil {
		return types.ListNull(suspensionObjectType), nil
	}

	if len(suspensions) == 0 {
		return types.ListNull(suspensionObjectType), nil
	}

	suspensionValues := make([]attr.Value, len(suspensions))

	for i, s := range suspensions {
		objValue, diags := types.ObjectValue(suspensionAttrTypes, map[string]attr.Value{
			"reason_code": types.StringValue(s.ReasonCode),
		})
		if diags.HasError() {
			return types.ListNull(suspensionObjectType), diags
		}
		suspensionValues[i] = objValue
	}
	return types.ListValue(suspensionObjectType, suspensionValues)

}
