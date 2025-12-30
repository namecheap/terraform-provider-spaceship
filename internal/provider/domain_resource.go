package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	client *Client
}

type domainResourceModel struct {
	Domain types.String `tfsdk:"domain"`
	Name   types.String `tfsdk:"name"`

	UnicodeName types.String `tfsdk:"unicode_name"`
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

	state.Name = types.StringValue(domainInfo.Name)
	state.UnicodeName = types.StringValue(domainInfo.UnicodeName)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (d *domainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *Client got %T", req.ProviderData))
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

	state.Domain = plan.Domain

	state.UnicodeName = types.StringValue(domainInfo.UnicodeName)
	state.Name = types.StringValue(domainInfo.Name)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
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

	tflog.Info(ctx, "calling domain autorenewal update with domain %s", map[string]any{
		"domain": domainName,
	},
	)

	domainInfo, err := d.client.GetDomainInfo(ctx, plan.Domain.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain info", err.Error())
	}

	state.UnicodeName = types.StringValue(domainInfo.UnicodeName)
	state.Name = types.StringValue(domainInfo.Name)

	state.Domain = plan.Domain
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "ImportState called", map[string]interface{}{
		"import_id": req.ID,
	})

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}
