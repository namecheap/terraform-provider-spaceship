package provider

import (
	"context"
	"fmt"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	applyDomainInfo(&state, domainInfo)

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

	applyDomainInfo(&state, domainInfo)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (d *domainResource) Delete(_ context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// removing resouce from state only
	// no external call
	// leaving infra in the same state
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

	domainInfo, err := d.client.GetDomainInfo(ctx, domainName)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain info", err.Error())
		return
	}

	state.Domain = plan.Domain
	applyDomainInfo(&state, domainInfo)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "ImportState called", map[string]interface{}{
		"import_id": req.ID,
	})

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}

func applyDomainInfo(state *domainResourceModel, info client.DomainInfo) {
	state.Name = types.StringValue(info.Name)
	state.UnicodeName = types.StringValue(info.UnicodeName)
	state.AutoRenew = types.BoolValue(info.AutoRenew)
}
