package provider

import (
	"context"
	"fmt"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func NewDomainListDataSource() datasource.DataSource {
	return &domainListDataSource{}
}

type domainListDataSource struct {
	client *client.Client
}

func (r *domainListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_list"
}

type domainListDataSourceModel struct {
	Items []domainModel `tfsdk:"items"`
	Total types.Int64   `tfsdk:"total"`
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
		domainDetails, domainDiags := buildDomainModel(ctx, item)
		resp.Diagnostics.Append(domainDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		data.Items = append(data.Items, domainDetails)
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
					Attributes: domainAttributes(),
				},
			},
		},
	}
}

func (r *domainListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
