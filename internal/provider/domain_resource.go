package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
						Description: "Optional list of contact attributes supplied by Spaceship.",
					},
				},
			},
			"privacy_protection": schema.SingleNestedAttribute{
				Computed: true,
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"contact_form": schema.BoolAttribute{
						Computed:    true,
						Optional:    true,
						Description: "Indicates whether WHOIS should display the contact form link",
					},
					"level": schema.StringAttribute{
						Computed:    true,
						Optional:    true,
						Description: "Privacy level: public or high",
						Validators: []validator.String{
							stringvalidator.OneOf("public", "high"),
						},
					},
				},
			},
			"nameservers": schema.SingleNestedAttribute{
				Computed: true,
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"provider": schema.StringAttribute{
						Computed:    true,
						Optional:    true,
						Description: "type: basic or custom",
						Validators: []validator.String{
							stringvalidator.OneOf("basic", "custom"),
						},
					},
					"hosts": schema.SetAttribute{
						Computed:    true,
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.Set{
							setvalidator.SizeBetween(2, 12),
							setvalidator.ValueStringsAre(
								stringvalidator.LengthBetween(4, 255),
								// Optionally add FQDN validation
							),
						},
					},
				},
			},
		},
	}
}

func (d *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to get state", map[string]interface{}{
			"errors": resp.Diagnostics.Errors(),
		})
		return
	}

	tflog.Debug(ctx, "About to call API", map[string]interface{}{
		"domain_value":      state.Domain.ValueString(),
		"domain_is_null":    state.Domain.IsNull(),
		"domain_is_unknown": state.Domain.IsUnknown(),
	})

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
		"epp_statuses":          domainInfo.EPPStatuses,
		"epp_statuses_type":     fmt.Sprintf("%T", domainInfo.EPPStatuses),
		"epp_statuses_len":      len(domainInfo.EPPStatuses),
		"nameservers.provider":  domainInfo.Nameservers.Provider,
		"nameservers.hosts.len": len(domainInfo.Nameservers.Hosts),
		"nameservers.hosts":     domainInfo.Nameservers,
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

	contactObj, diags := contactsToTerraformObject(ctx, domainInfo.Contacts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Contacts = contactObj

	ppObject, diags := privacyProtectionToTerraformObject(ctx, domainInfo.PrivacyProtection)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.PrivacyProtection = ppObject

	stateNsObject, nsDiag := nameseversToTerraformObject(ctx, domainInfo.Nameservers)
	resp.Diagnostics.Append(nsDiag...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Nameservers = stateNsObject

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

func (d *domainResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
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

	if planNS.Hosts.IsUnknown() || planNS.Hosts.IsNull() {
		return
	}

	var hosts []string
	resp.Diagnostics.Append(planNS.Hosts.ElementsAs(ctx, &hosts, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sort.Strings(hosts)

	sortedHosts, diags := types.SetValueFrom(ctx, types.StringType, hosts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Plan.SetAttribute(ctx, path.Root("nameservers").AtName("hosts"), sortedHosts)
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
		"epp_statuses":          domainInfo.EPPStatuses,
		"epp_statuses_type":     fmt.Sprintf("%T", domainInfo.EPPStatuses),
		"epp_statuses_len":      len(domainInfo.EPPStatuses),
		"nameservers.provider":  domainInfo.Nameservers.Provider,
		"nameservers.hosts.len": len(domainInfo.Nameservers.Hosts),
		"nameservers.hosts":     domainInfo.Nameservers,
	})

	eppStatuses, _ := types.ListValueFrom(ctx, types.StringType, domainInfo.EPPStatuses)
	state.EppStatuses = eppStatuses

	suspensions, diags := SuspensionsToTerraformList(ctx, domainInfo.Suspensions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Suspensions = suspensions

	contactObj, diags := contactsToTerraformObject(ctx, domainInfo.Contacts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Contacts = contactObj

	ppObject, diags := privacyProtectionToTerraformObject(ctx, domainInfo.PrivacyProtection)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.PrivacyProtection = ppObject

	stateNsObject, nsDiag := nameseversToTerraformObject(ctx, domainInfo.Nameservers)
	resp.Diagnostics.Append(nsDiag...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Nameservers = stateNsObject

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (d *domainResource) Delete(_ context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// should be done as removal from state only
	// done automatically by empty delete method
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

	var (
		planPrivacy  privacyProtection
		statePrivacy privacyProtection
		planNS       nameservers
		stateNS      nameservers
	)

	planHasPrivacy := !plan.PrivacyProtection.IsUnknown() && !plan.PrivacyProtection.IsNull()
	stateHasPrivacy := !state.PrivacyProtection.IsUnknown() && !state.PrivacyProtection.IsNull()
	planHasNameservers := !plan.Nameservers.IsUnknown() && !plan.Nameservers.IsNull()
	stateHasNameservers := !state.Nameservers.IsUnknown() && !state.Nameservers.IsNull()

	domainName := plan.Domain.ValueString()

	tflog.Info(ctx, "calling domain autorenewal update with domain %s", map[string]any{
		"domain": domainName,
	},
	)

	// check auto_renew updates
	if !plan.AutoRenew.Equal(state.AutoRenew) {
		tflog.Debug(ctx, "auto_renew changed", map[string]any{
			"old": state.AutoRenew.ValueBool(),
			"new": plan.AutoRenew.ValueBool(),
		})

		_, err := d.client.UpdateAutoRenew(ctx, domainName, plan.AutoRenew.ValueBool())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating domain auto renew",
				fmt.Sprintf("Could not update auto_renew for domain %s: %s", domainName, err),
			)
			return
		}
	}

	// check privacy protection changes
	if planHasPrivacy {
		// check each field separately
		resp.Diagnostics.Append(plan.PrivacyProtection.As(ctx, &planPrivacy, basetypes.ObjectAsOptions{})...)
		if stateHasPrivacy {
			resp.Diagnostics.Append(state.PrivacyProtection.As(ctx, &statePrivacy, basetypes.ObjectAsOptions{})...)
		}
		if resp.Diagnostics.HasError() {
			return
		}

		// update level first as contact_form preference depends on it being supported
		if !planPrivacy.Level.IsUnknown() && !planPrivacy.Level.IsNull() &&
			(!stateHasPrivacy || !planPrivacy.Level.Equal(statePrivacy.Level)) {
			tflog.Debug(ctx, "privacy level changed")

			err := d.client.UpdateDomainPrivacyPreference(ctx, domainName, PrivacyLevel(planPrivacy.Level.ValueString()))
			if err != nil {
				resp.Diagnostics.AddError("Failed to update privacy_protection level", err.Error())
				return
			}

		}

		//check contact_form
		if !planPrivacy.ContactForm.IsUnknown() && !planPrivacy.ContactForm.IsNull() &&
			(!stateHasPrivacy || !planPrivacy.ContactForm.Equal(statePrivacy.ContactForm)) {
			tflog.Debug(ctx, "contact_form changed")

			err := d.client.UpdateDomainEmailProtectionPreference(ctx, domainName, planPrivacy.ContactForm.ValueBool())
			if err != nil {
				tflog.Debug(ctx, "Failed to update privacy_protection contact_form", map[string]any{
					"contact_form_type":      fmt.Sprintf("%T", planPrivacy.ContactForm.ValueBool()),
					"contact_form_new_value": planPrivacy.ContactForm.ValueBool(),
					"error":                  err.Error(),
				},
				)
				resp.Diagnostics.AddError("Failed to update privacy_protection contact_form", err.Error())
				return
			}
		}
	}

	//check nameservers changes
	if planHasNameservers {
		resp.Diagnostics.Append(plan.Nameservers.As(ctx, &planNS, basetypes.ObjectAsOptions{})...)
		if stateHasNameservers {
			resp.Diagnostics.Append(state.Nameservers.As(ctx, &stateNS, basetypes.ObjectAsOptions{})...)
		}
		if resp.Diagnostics.HasError() {
			return
		}

		planProviderKnown := !planNS.Provider.IsUnknown() && !planNS.Provider.IsNull()
		stateProviderKnown := stateHasNameservers && !stateNS.Provider.IsUnknown() && !stateNS.Provider.IsNull()

		var planHosts, stateHosts []string
		planHostsKnown := !planNS.Hosts.IsUnknown() && !planNS.Hosts.IsNull()
		stateHostsKnown := stateHasNameservers && !stateNS.Hosts.IsUnknown() && !stateNS.Hosts.IsNull()

		if planHostsKnown {
			resp.Diagnostics.Append(planNS.Hosts.ElementsAs(ctx, &planHosts, false)...)
		}
		if stateHostsKnown {
			resp.Diagnostics.Append(stateNS.Hosts.ElementsAs(ctx, &stateHosts, false)...)
		}
		if resp.Diagnostics.HasError() {
			return
		}

		planHostsSorted := append([]string(nil), planHosts...)
		stateHostsSorted := append([]string(nil), stateHosts...)
		if planHostsKnown {
			sort.Strings(planHostsSorted)
		}
		if stateHostsKnown {
			sort.Strings(stateHostsSorted)
		}

		var planProvider, stateProvider string
		if planProviderKnown {
			planProvider = planNS.Provider.ValueString()
		}
		if stateProviderKnown {
			stateProvider = stateNS.Provider.ValueString()
		}

		providerChanged := planProviderKnown && (!stateProviderKnown || planProvider != stateProvider)
		hostsChanged := planHostsKnown && !stringSlicesEqual(planHostsSorted, stateHostsSorted)

		if providerChanged || hostsChanged {
			nsProvider := planProvider
			if nsProvider == "" {
				nsProvider = stateProvider
			}

			requestHosts := planHostsSorted
			if nsProvider == string(BasicNameserverProvider) {
				// API ignores hosts for basic provider and expects field to be omitted
				requestHosts = nil
			}

			tflog.Debug(ctx, "nameservers changed", map[string]any{
				"provider_changed": providerChanged,
				"hosts_changed":    hostsChanged,
				"new_provider":     nsProvider,
				"new_hosts":        requestHosts,
			})

			err := d.client.UpdateDomainNameServers(ctx, domainName, UpdateNameserverRequest{
				Provider: NameserverProvider(nsProvider),
				Hosts:    requestHosts,
			})
			if err != nil {
				resp.Diagnostics.AddError("Failed to update domain nameservers", err.Error())
				return
			}
		}
	}

	domainInfo, err := d.client.GetDomainInfo(ctx, plan.Domain.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain info", err.Error())
	}

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
		"epp_statuses":                    domainInfo.EPPStatuses,
		"epp_statuses_type":               fmt.Sprintf("%T", domainInfo.EPPStatuses),
		"epp_statuses_len":                len(domainInfo.EPPStatuses),
		"privacy_protection.contact_form": domainInfo.PrivacyProtection.ContactForm,
		"privacy_protection.level":        domainInfo.PrivacyProtection.Level,
	})

	eppStatuses, _ := types.ListValueFrom(ctx, types.StringType, domainInfo.EPPStatuses)
	state.EppStatuses = eppStatuses

	suspensions, diags := SuspensionsToTerraformList(ctx, domainInfo.Suspensions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Suspensions = suspensions

	contactObj, diags := contactsToTerraformObject(ctx, domainInfo.Contacts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Contacts = contactObj

	ppObject, diags := privacyProtectionToTerraformObject(ctx, domainInfo.PrivacyProtection)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// prefer explicit plan values when provided to avoid transient backend lag
	ppAttrTypes := map[string]attr.Type{
		"contact_form": types.BoolType,
		"level":        types.StringType,
	}
	ppValues := map[string]attr.Value{
		"contact_form": ppObject.Attributes()["contact_form"],
		"level":        ppObject.Attributes()["level"],
	}
	if planHasPrivacy {
		if !planPrivacy.ContactForm.IsUnknown() && !planPrivacy.ContactForm.IsNull() {
			ppValues["contact_form"] = planPrivacy.ContactForm
		}
		if !planPrivacy.Level.IsUnknown() && !planPrivacy.Level.IsNull() {
			ppValues["level"] = planPrivacy.Level
		}
	}

	statePrivacyObject, ppDiag := types.ObjectValue(ppAttrTypes, ppValues)
	resp.Diagnostics.Append(ppDiag...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.PrivacyProtection = statePrivacyObject

	stateNsObject, nsDiag := nameseversToTerraformObject(ctx, domainInfo.Nameservers)
	resp.Diagnostics.Append(nsDiag...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Nameservers = stateNsObject

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "ImportState called", map[string]interface{}{
		"import_id": req.ID,
	})

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}

type domainResourceModel struct {
	Domain types.String `tfsdk:"domain"`

	//configurable
	AutoRenew         types.Bool   `tfsdk:"auto_renew"`
	Nameservers       types.Object `tfsdk:"nameservers"`
	PrivacyProtection types.Object `tfsdk:"privacy_protection"`

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
	Contacts           types.Object `tfsdk:"contacts"`
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
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

func contactsToTerraformObject(ctx context.Context, contacts Contacts) (types.Object, diag.Diagnostics) {

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
		"admin":      types.StringValue(contacts.Admin),
		"billing":    stringValueOrNull(contacts.Billing),
		"registrant": stringValueOrNull(contacts.Registrant),
		"tech":       stringValueOrNull(contacts.Tech),
		"attributes": attributesValues,
	}

	return types.ObjectValue(contactsAttrTypes, contactsValues)

}

func privacyProtectionToTerraformObject(_ context.Context, pp PrivacyProtection) (types.Object, diag.Diagnostics) {
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

func nameseversToTerraformObject(ctx context.Context, ns Nameservers) (types.Object, diag.Diagnostics) {

	nsAttrTypes := map[string]attr.Type{
		"provider": types.StringType,
		"hosts":    types.SetType{ElemType: types.StringType},
	}

	var nsHosts types.Set
	if ns.Hosts == nil {
		nsHosts = types.SetNull(types.StringType)
	} else {
		sortedHosts := make([]string, len(ns.Hosts))
		copy(sortedHosts, ns.Hosts)
		sort.Strings(sortedHosts)

		var diags diag.Diagnostics
		nsHosts, diags = types.SetValueFrom(ctx, types.StringType, sortedHosts)
		if diags.HasError() {
			return types.ObjectNull(nsAttrTypes), diags
		}
	}

	nsValues := map[string]attr.Value{
		"provider": types.StringValue(ns.Provider),
		"hosts":    nsHosts,
	}
	return types.ObjectValue(nsAttrTypes, nsValues)
}
