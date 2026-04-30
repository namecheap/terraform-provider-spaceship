package provider

import (
	"context"
	"fmt"
	"maps"
	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewDNSRecordResource() resource.Resource {
	return &dnsRecordResource{}
}

type dnsRecordResource struct {
	client *client.Client
}

type dnsRecordResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Domain types.String `tfsdk:"domain"`

	//todo
	// learn how does it work in go
	// try more example
	dnsRecordModel
}

func (r *dnsRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *dnsRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Composite identifier (`domain/type/name/data`) for this record.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"domain": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The domain name the record belongs to (for example `example.com`). The domain must already exist in the Spaceship account.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
	maps.Copy(attrs, recordAttributes())

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single DNS record for a Spaceship-managed domain. Only records in the `custom` DNS group are managed — records owned by Spaceship features (e.g. URL redirect, personal nameservers) are left untouched.",
		Attributes:          attrs,
	}
}

func (r *dnsRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *dnsRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured provider", "The Spaceship provider was not configured. Please ensure the provider block is present")
		return
	}

	var plan dnsRecordResourceModel
	// this line is super important
	// values would not be read from terraform without it
	// how it even works?
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	domain := plan.Domain.ValueString()
	recordType := plan.Type.ValueString()
	name := plan.Name.ValueString()

	// how to create any other record field?
	address := plan.Address.ValueString()

	//todo why it is needed
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	//adopt on creation?
	//existingRecords, err := r.client.GetDNSRecords(ctx, plan.Domain.ValueString())

	//TODO
	//how recordKey() func could be reused here?

	// TODO fetch before creation
	// record, err := r.client.GetSpecificDNSRecord(ctx, domain, recordType, name)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("failed to read existing DNS records: %s", err))
	// 	return
	// }

	record := client.DNSRecord{
		Type:    recordType,
		Name:    name,
		TTL:     3600,
		Address: address,
	}

	if err := r.client.CreateDNSRecord(ctx, domain, record); err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to create DNS record: %s", err))
		return
	}

	//todo change to composite key
	plan.ID = types.StringValue(domain)
	plan.Domain = types.StringValue(domain)
	plan.Type = types.StringValue(recordType)
	plan.Address = types.StringValue(address)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}

func (r *dnsRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured provider", "The Spaceship provider was not configured. Please ensure the provider block is present.")
		return
	}

	var state dnsRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := state.Domain.ValueString()
	recordType := state.Type.ValueString()
	name := state.Name.ValueString()
	// how to create any other record field?

	address := state.Address.ValueString()
	record := client.DNSRecord{
		Type:    recordType,
		Name:    name,
		TTL:     3600,
		Address: address,
	}

	if err := r.client.DeleteDNSRecord(ctx, domain, record); err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to delete DNS record: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)

}

func (r *dnsRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

}

func (r *dnsRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

}

func (r *dnsRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
}
