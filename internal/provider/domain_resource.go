package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/namecheap/go-spaceship-sdk/client"
)

// Worst case a create/update makes three rate-limitable calls (domain-info
// read plus two writes), each of which may wait out a full throttling window;
// a read makes one. See internal/docs/rate-limits.md.
const (
	domainCreateTimeout = 15 * time.Minute
	domainReadTimeout   = 5 * time.Minute
	domainUpdateTimeout = 15 * time.Minute
)

func NewDomainResource() resource.Resource {
	return &domainResource{}
}

type domainResource struct {
	client *client.Client
}

type domainResourceModel struct {
	Domain types.String `tfsdk:"domain"`

	// Configurable
	AutoRenew   types.Bool     `tfsdk:"auto_renew"`
	Nameservers types.Object   `tfsdk:"nameservers"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`

	// Read only
	Name        types.String `tfsdk:"name"`
	UnicodeName types.String `tfsdk:"unicode_name"`

	IsPremium          types.Bool   `tfsdk:"is_premium"`
	RegistrationDate   types.String `tfsdk:"registration_date"`
	ExpirationDate     types.String `tfsdk:"expiration_date"`
	LifecycleStatus    types.String `tfsdk:"lifecycle_status"`
	VerificationStatus types.String `tfsdk:"verification_status"`
	EppStatuses        types.List   `tfsdk:"epp_statuses"`
	Suspensions        types.List   `tfsdk:"suspensions"`
	Contacts           types.Object `tfsdk:"contacts"`
	PrivacyProtection  types.Object `tfsdk:"privacy_protection"`
}

func (d *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *domainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages settings of a domain registered with Spaceship — auto-renew and nameserver delegation — and exposes the domain's registration details (dates, contacts, privacy protection, lifecycle and verification status).",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "The domain name to manage (for example example.com). The domain must already exist in the Spaceship account. Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(4, 255),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"unicode_name": schema.StringAttribute{
				Computed:    true,
				Description: "Domain name in UTF-8 format (U-label)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Domain name in ASCII format (A-label)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"auto_renew": schema.BoolAttribute{
				Computed:    true,
				Optional:    true,
				Description: "Indicates whether the auto-renew option is enabled",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"nameservers": schema.SingleNestedAttribute{
				Computed:    true,
				Optional:    true,
				Description: "Nameserver delegation for the domain.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"provider": schema.StringAttribute{
						Computed:    true,
						Optional:    true,
						Description: "Nameserver provider: basic (Spaceship's default nameservers) or custom (the hosts listed in hosts).",
						Validators: []validator.String{
							stringvalidator.OneOf("basic", "custom"),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"hosts": schema.SetAttribute{
						Computed:    true,
						Optional:    true,
						ElementType: types.StringType,
						Description: "Nameserver host names (2 to 12 entries). Required when provider is custom.",
						Validators: []validator.Set{
							setvalidator.SizeBetween(2, 12),
							setvalidator.ValueStringsAre(
								stringvalidator.LengthBetween(4, 255),
							),
						},
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.UseStateForUnknown(),
						},
					},
				},
				Validators: []validator.Object{&nameserversValidator{}},
			},
			"is_premium": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the domain is a premium-priced domain.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"registration_date": schema.StringAttribute{
				Computed:    true,
				Description: "Date and time when the domain was registered.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expiration_date": schema.StringAttribute{
				Computed:    true,
				Description: "Date and time when the domain registration expires.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"lifecycle_status": schema.StringAttribute{
				Computed:    true,
				Description: "Lifecycle phase. One of creating, registered, grace1, grace2, redemption.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"verification_status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the RAA verification process. One of verification, success, failed. Null when not applicable.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"epp_statuses": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Possible values clientDeleteProhibited clientHold clientRenewProhibited clientTransferProhibited clientUpdateProhibited",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"suspensions": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Information about domain suspensions. May contain up to 2 items.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"reason_code": schema.StringAttribute{
							Computed:    true,
							Description: "Suspension reason code (raaVerification, abuse, promoAbuse, fraud, pendingAccountVerification, unauthorizedAccess, tosViolation, transferDispute, restrictedSecurity, lockCourt, suspendCourt, udrpUrs, restrictedLegal, paymentPending, unpaidService, restrictedWhois, lockedWhois)",
						},
					},
				},
			},
			"contacts": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Contact handles assigned to the domain.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
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
						Description: "Optional list of contact attributes supplied by Spaceship.",
					},
				},
			},
			"privacy_protection": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "WHOIS privacy protection settings for the domain.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"contact_form": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates whether WHOIS should display the contact form link",
					},
					"level": schema.StringAttribute{
						Computed:    true,
						Description: "Privacy level: public or high",
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
			}),
		},
	}
}

func (d *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to get state", map[string]interface{}{
			"errors": resp.Diagnostics.HasError(),
		})
		return
	}

	readTimeout, timeoutDiags := state.Timeouts.Read(ctx, domainReadTimeout)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	domain := state.Domain.ValueString()

	tflog.Debug(ctx, "About to call API to read domain state", map[string]interface{}{
		"domain_value":   domain,
		"domain_is_null": state.Domain.IsNull(),
	})

	var domainInfo client.DomainInfo
	err := withRetry(ctx, "read domain info", func() error {
		var apiErr error
		domainInfo, apiErr = d.client.GetDomainInfo(ctx, domain)
		return apiErr
	})

	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain info", err.Error())
		return
	}

	resp.Diagnostics.Append(applyDomainInfo(ctx, &state, domainInfo)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (d *domainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pd, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *providerData, got %T", req.ProviderData))
		return
	}

	d.client = pd.Client
}

func (d *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, timeoutDiags := plan.Timeouts.Create(ctx, domainCreateTimeout)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	domainName := plan.Domain.ValueString()

	var domainInfo client.DomainInfo
	err := withRetry(ctx, "read domain info", func() error {
		var apiErr error
		domainInfo, apiErr = d.client.GetDomainInfo(ctx, domainName)
		return apiErr
	})

	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain info", err.Error())
		return
	}

	// Adoption must converge the domain to the configured settings. Otherwise
	// a config/infra mismatch on the first apply stores the actual values
	// while the plan promised the configured ones, and Terraform fails with
	// "Provider produced inconsistent result after apply".
	if !plan.AutoRenew.IsNull() && !plan.AutoRenew.IsUnknown() && plan.AutoRenew.ValueBool() != domainInfo.AutoRenew {
		err := withRetry(ctx, "update auto_renew", func() error {
			_, apiErr := d.client.UpdateAutoRenew(ctx, domainName, plan.AutoRenew.ValueBool())
			return apiErr
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating domain auto_renew",
				fmt.Sprintf("Could not update auto_renew for domain %s: %s", domainName, err),
			)
			return
		}
	}

	// Nested attributes can still be unknown at create (there is no prior
	// state for UseStateForUnknown to resolve them); in that case fall back
	// to adopting the API values.
	nsPlanKnown := false
	if !plan.Nameservers.IsNull() && !plan.Nameservers.IsUnknown() {
		var planNS nameservers
		resp.Diagnostics.Append(plan.Nameservers.As(ctx, &planNS, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		nsPlanKnown = !planNS.Provider.IsUnknown() && !planNS.Hosts.IsUnknown()
	}

	if nsPlanKnown {
		infraNS, nsDiags := nameserversToTerraformObject(ctx, domainInfo.Nameservers)
		resp.Diagnostics.Append(nsDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if !plan.Nameservers.Equal(infraNS) {
			resp.Diagnostics.Append(d.pushNameservers(ctx, domainName, plan.Nameservers)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	var state domainResourceModel

	state.Domain = plan.Domain
	state.Timeouts = plan.Timeouts

	resp.Diagnostics.Append(applyDomainInfo(ctx, &state, domainInfo)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The autorenew and nameserver update APIs confirm changes synchronously;
	// trust the plan values over the pre-update read (same rule as Update).
	if !plan.AutoRenew.IsNull() && !plan.AutoRenew.IsUnknown() {
		state.AutoRenew = plan.AutoRenew
	}
	if nsPlanKnown {
		state.Nameservers = plan.Nameservers
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (d *domainResource) Delete(_ context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// removing resource from state only
	// no external call
	// leaving infra in the same state
}

func (d *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state domainResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, timeoutDiags := plan.Timeouts.Update(ctx, domainUpdateTimeout)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	domainName := plan.Domain.ValueString()

	// check autorenewal change
	if !plan.AutoRenew.IsNull() && !plan.AutoRenew.IsUnknown() && !plan.AutoRenew.Equal(state.AutoRenew) {
		newValue := plan.AutoRenew.ValueBool()

		tflog.Debug(ctx, "auto_renew changed", map[string]any{
			"old": state.AutoRenew.ValueBool(),
			"new": newValue,
		})

		err := withRetry(ctx, "update auto_renew", func() error {
			_, apiErr := d.client.UpdateAutoRenew(ctx, domainName, newValue)
			return apiErr
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating domain auto_renew",
				fmt.Sprintf("Could not update auto_renew for domain %s: %s", domainName, err),
			)
			return
		}
	}

	// Check nameservers changes
	if !plan.Nameservers.IsNull() && !plan.Nameservers.IsUnknown() {
		// Use Terraform's built-in Equal() - it handles sets correctly (order-independent)
		if !plan.Nameservers.Equal(state.Nameservers) {
			resp.Diagnostics.Append(d.pushNameservers(ctx, domainName, plan.Nameservers)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	// reread domain info configuration
	var domainInfo client.DomainInfo
	err := withRetry(ctx, "read domain info", func() error {
		var apiErr error
		domainInfo, apiErr = d.client.GetDomainInfo(ctx, domainName)
		return apiErr
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain info", err.Error())
		return
	}

	state.Domain = plan.Domain
	state.Timeouts = plan.Timeouts

	resp.Diagnostics.Append(applyDomainInfo(ctx, &state, domainInfo)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The autorenew and nameserver update APIs confirm changes synchronously
	// (return the new value), but GetDomainInfo may return stale data due to
	// eventual consistency. Trust the plan values for fields we successfully updated.
	if !plan.AutoRenew.IsNull() && !plan.AutoRenew.IsUnknown() {
		state.AutoRenew = plan.AutoRenew
	}
	if !plan.Nameservers.IsNull() && !plan.Nameservers.IsUnknown() {
		state.Nameservers = plan.Nameservers
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "ImportState called", map[string]interface{}{
		"import_id": req.ID,
	})

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}

func (d *domainResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Handle destruction
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.AddWarning(
			"Resource Destruction Considerations",
			"Applying this resource destruction will only remove the resource from the Terraform state "+
				"and will not call the deletion API due to nature of domain specifics. "+
				"Your registered domain and its settings would remain intact",
		)
		return
	}

	var plan domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Nameservers.IsUnknown() || plan.Nameservers.IsNull() {
		return
	}

	var planNS nameservers
	resp.Diagnostics.Append(plan.Nameservers.As(ctx, &planNS, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine which hosts to set
	var hostsToSet []string

	// ValueString() returns "" for null/unknown, which won't match "basic"
	if planNS.Provider.ValueString() == string(client.BasicNameserverProvider) {
		hostsToSet = client.DefaultBasicNameserverHosts()
	} else if !planNS.Hosts.IsUnknown() && !planNS.Hosts.IsNull() {
		resp.Diagnostics.Append(planNS.Hosts.ElementsAs(ctx, &hostsToSet, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		return
	}

	hostsSet, diags := types.SetValueFrom(ctx, types.StringType, hostsToSet)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Plan.SetAttribute(ctx, path.Root("nameservers").AtName("hosts"), hostsSet)
}

// pushNameservers applies the planned nameservers object to the API.
func (d *domainResource) pushNameservers(ctx context.Context, domainName string, planNameservers types.Object) diag.Diagnostics {
	var diags diag.Diagnostics

	var planNS nameservers
	diags.Append(planNameservers.As(ctx, &planNS, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return diags
	}

	provider := client.NameserverProvider(planNS.Provider.ValueString())

	var hosts []string
	if !planNS.Hosts.IsNull() && !planNS.Hosts.IsUnknown() {
		diags.Append(planNS.Hosts.ElementsAs(ctx, &hosts, false)...)
		if diags.HasError() {
			return diags
		}
	}

	// API ignores hosts for basic provider
	if provider == client.BasicNameserverProvider {
		hosts = nil
	}

	err := withRetry(ctx, "update nameservers", func() error {
		return d.client.UpdateDomainNameServers(ctx, domainName, client.UpdateNameserverRequest{
			Provider: provider,
			Hosts:    hosts,
		})
	})
	if err != nil {
		diags.AddError("Failed to update domain nameservers", err.Error())
	}

	return diags
}

func applyDomainInfo(ctx context.Context, state *domainResourceModel, info client.DomainInfo) diag.Diagnostics {
	var diags diag.Diagnostics

	state.Name = types.StringValue(info.Name)
	state.UnicodeName = types.StringValue(info.UnicodeName)
	state.AutoRenew = types.BoolValue(info.AutoRenew)

	state.IsPremium = types.BoolValue(info.IsPremium)
	state.RegistrationDate = types.StringValue(info.RegistrationDate)
	state.ExpirationDate = types.StringValue(info.ExpirationDate)
	state.LifecycleStatus = types.StringValue(info.LifecycleStatus)
	state.VerificationStatus = stringValueOrNull(info.VerificationStatus)

	eppStatuses, eppDiags := types.ListValueFrom(ctx, types.StringType, info.EPPStatuses)
	diags.Append(eppDiags...)
	if diags.HasError() {
		logDiagnostics(ctx, "Failed to map epp_statuses", diags)
		return diags
	}
	state.EppStatuses = eppStatuses

	nameservers, nsDiags := nameserversToTerraformObject(ctx, info.Nameservers)
	diags.Append(nsDiags...)
	if diags.HasError() {
		logDiagnostics(ctx, "Failed to map nameservers", diags)
		return diags
	}
	state.Nameservers = nameservers

	suspensions, suspDiags := suspensionsToTerraformList(ctx, info.Suspensions)
	diags.Append(suspDiags...)
	if diags.HasError() {
		logDiagnostics(ctx, "Failed to map suspensions", diags)
		return diags
	}
	state.Suspensions = suspensions

	contactsObj, contactDiags := contactsToTerraformObject(ctx, info.Contacts)
	diags.Append(contactDiags...)
	if diags.HasError() {
		logDiagnostics(ctx, "Failed to map contacts", diags)
		return diags
	}
	state.Contacts = contactsObj

	ppObject, ppDiags := privacyProtectionToTerraformObject(ctx, info.PrivacyProtection)
	diags.Append(ppDiags...)
	if diags.HasError() {
		logDiagnostics(ctx, "Failed to map privacy_protection", diags)
		return diags
	}
	state.PrivacyProtection = ppObject

	return diags
}

func logDiagnostics(ctx context.Context, message string, diags diag.Diagnostics) {
	if !diags.HasError() {
		return
	}

	tflog.Error(ctx, message, map[string]any{
		"diagnostics": diags,
	})
}

func nameserversToTerraformObject(ctx context.Context, ns client.Nameservers) (types.Object, diag.Diagnostics) {
	nsAttributeTypes := map[string]attr.Type{
		"provider": types.StringType,
		"hosts":    types.SetType{ElemType: types.StringType},
	}

	var nsHosts types.Set
	if ns.Hosts == nil {
		nsHosts = types.SetNull(types.StringType)
	} else {
		var diags diag.Diagnostics
		// SetValueFrom handles the conversion directly - no sorting needed for Sets
		nsHosts, diags = types.SetValueFrom(ctx, types.StringType, ns.Hosts)

		if diags.HasError() {
			return types.ObjectNull(nsAttributeTypes), diags
		}
	}

	nsValues := map[string]attr.Value{
		"provider": stringValueOrNull(ns.Provider),
		"hosts":    nsHosts,
	}

	return types.ObjectValue(nsAttributeTypes, nsValues)

}

func suspensionsToTerraformList(ctx context.Context, suspensions []client.ReasonCode) (types.List, diag.Diagnostics) {
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

func contactsToTerraformObject(ctx context.Context, contacts client.Contacts) (types.Object, diag.Diagnostics) {

	contactsAttrTypes := map[string]attr.Type{
		"admin":      types.StringType,
		"billing":    types.StringType,
		"registrant": types.StringType,
		"tech":       types.StringType,
		"attributes": types.ListType{ElemType: types.StringType},
	}

	var attributesValues types.List
	if contacts.Attributes == nil {
		attributesValues = types.ListNull(types.StringType)
	} else {
		var diags diag.Diagnostics
		attributesValues, diags = types.ListValueFrom(ctx, types.StringType, contacts.Attributes)
		if diags.HasError() {
			return types.ObjectNull(contactsAttrTypes), diags
		}
	}

	contactsValues := map[string]attr.Value{
		// cant be empty or null accordingly to api
		// it is required
		// https://docs.spaceship.dev/#tag/Domains/operation/getDomainInfo
		"registrant": types.StringValue(contacts.Registrant),
		"admin":      stringValueOrNull(contacts.Admin),
		"billing":    stringValueOrNull(contacts.Billing),
		"tech":       stringValueOrNull(contacts.Tech),
		"attributes": attributesValues,
	}

	return types.ObjectValue(contactsAttrTypes, contactsValues)

}

func privacyProtectionToTerraformObject(_ context.Context, pp client.PrivacyProtection) (types.Object, diag.Diagnostics) {
	ppAttrTypes := map[string]attr.Type{
		"contact_form": types.BoolType,
		"level":        types.StringType,
	}

	ppValues := map[string]attr.Value{
		"contact_form": types.BoolValue(pp.ContactForm),
		"level":        types.StringValue(pp.Level),
	}

	return types.ObjectValue(ppAttrTypes, ppValues)
}
