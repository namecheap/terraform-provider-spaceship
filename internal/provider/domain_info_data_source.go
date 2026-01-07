package provider

import (
	"context"
	"fmt"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewDomainInfoDataSource() datasource.DataSource {
	return &domainInfoDataSource{}
}

type domainInfoDataSource struct {
	client *client.Client
}

func (d *domainInfoDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_info"
}

func (d *domainInfoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var domain types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("domain"), &domain)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := d.client.GetDomainInfo(ctx, domain.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain info", err.Error())
		return
	}

	domainDetails, domainDiags := buildDomainModel(ctx, response)
	resp.Diagnostics.Append(domainDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data := domainInfoModel{Domain: domain}
	data.domainModel = domainDetails

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (d *domainInfoDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := domainAttributes()
	attrs["domain"] = schema.StringAttribute{
		Required: true,
		Validators: []validator.String{
			stringvalidator.LengthBetween(4, 255),
		},
	}

	resp.Schema = schema.Schema{
		Description: "Get all info about single domain",
		Attributes:  attrs,
	}
}

func (d *domainInfoDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
		return
	}

	d.client = client
}
