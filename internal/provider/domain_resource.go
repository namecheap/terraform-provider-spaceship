package provider

import (
	"context"
	"fmt"
	"sort"
	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func NewDomainResource() resource.Resource {
	return &domainResource{}
}

type domainResource struct {
	client *client.Client
}

type domainResourceModel struct {
	Domain types.String `tfsdk:"domain"`

	Name        types.String `tfsdk:"name"`
	UnicodeName types.String `tfsdk:"unicode_name"`
	AutoRenew   types.Bool   `tfsdk:"auto_renew"`

	Nameservers types.Object `tfsdk:"nameservers"`
}

func (d *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *domainResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages domain setting for Spaceship domain",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "Indicate domain name which you want to manage",
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
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Domain name in ASCII format (A-label)",
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
				Description: "Information about nameservers",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"provider": schema.StringAttribute{
						Computed:    true,
						Optional:    true,
						Description: "type: basic or custom",
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

	domain := state.Domain.ValueString()

	tflog.Debug(ctx, "About to call API to read domain state", map[string]interface{}{
		"domain_value":   domain,
		"domain_is_null": state.Domain.IsNull(),
	})

	domainInfo, err := d.client.GetDomainInfo(ctx, domain)

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

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *client.Client got %T", req.ProviderData))
		return
	}

	d.client = client
}

func (d *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainResourceModel
	var domainInfo client.DomainInfo

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

	state.Domain = plan.Domain

	resp.Diagnostics.Append(applyDomainInfo(ctx, &state, domainInfo)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (d *domainResource) Delete(_ context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// removing resouce from state only
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

	domainName := plan.Domain.ValueString()

	// check autorenewal change
	if !plan.AutoRenew.IsNull() && !plan.AutoRenew.IsUnknown() && !plan.AutoRenew.Equal(state.AutoRenew) {
		newValue := plan.AutoRenew.ValueBool()

		tflog.Debug(ctx, "auto_renew changed", map[string]any{
			"old": state.AutoRenew.ValueBool(),
			"new": newValue,
		})

		_, err := d.client.UpdateAutoRenew(ctx, domainName, newValue)
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
			var planNS nameservers
			resp.Diagnostics.Append(plan.Nameservers.As(ctx, &planNS, basetypes.ObjectAsOptions{})...)
			if resp.Diagnostics.HasError() {
				return
			}

			provider := client.NameserverProvider(planNS.Provider.ValueString())

			var hosts []string
			if !planNS.Hosts.IsNull() && !planNS.Hosts.IsUnknown() {
				resp.Diagnostics.Append(planNS.Hosts.ElementsAs(ctx, &hosts, false)...)
				if resp.Diagnostics.HasError() {
					return
				}
			}

			// API ignores hosts for basic provider
			if provider == client.BasicNameserverProvider {
				hosts = nil
			}

			err := d.client.UpdateDomainNameServers(ctx, domainName, client.UpdateNameserverRequest{
				Provider: provider,
				Hosts:    hosts,
			})
			if err != nil {
				resp.Diagnostics.AddError("Failed to update domain nameservers", err.Error())
				return
			}
		}
	}

	// reread domain info configuration
	domainInfo, err := d.client.GetDomainInfo(ctx, domainName)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain info", err.Error())
		return
	}

	state.Domain = plan.Domain

	resp.Diagnostics.Append(applyDomainInfo(ctx, &state, domainInfo)...)
	if resp.Diagnostics.HasError() {
		return
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

	// Sort and set hosts
	sort.Strings(hostsToSet)
	sortedHosts, diags := types.SetValueFrom(ctx, types.StringType, hostsToSet)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Plan.SetAttribute(ctx, path.Root("nameservers").AtName("hosts"), sortedHosts)
}

func applyDomainInfo(ctx context.Context, state *domainResourceModel, info client.DomainInfo) diag.Diagnostics {
	state.Name = types.StringValue(info.Name)
	state.UnicodeName = types.StringValue(info.UnicodeName)
	state.AutoRenew = types.BoolValue(info.AutoRenew)

	nameservers, diags := constructNameservers(ctx, info.Nameservers)
	if diags.HasError() {
		return diags
	}
	state.Nameservers = nameservers
	return nil

}

func constructNameservers(ctx context.Context, ns client.Nameservers) (types.Object, diag.Diagnostics) {
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
		"provider": types.StringValue(ns.Provider),
		"hosts":    nsHosts,
	}

	return types.ObjectValue(nsAttributeTypes, nsValues)

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
