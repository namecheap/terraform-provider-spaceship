package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/namecheap/go-spaceship-sdk/client"
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
	Items    []domainModel  `tfsdk:"items"`
	Total    types.Int64    `tfsdk:"total"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
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

	readTimeout, timeoutDiags := data.Timeouts.Read(ctx, domainReadTimeout)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	var response client.DomainList
	err = withRetry(ctx, "read domain list", func() error {
		var apiErr error
		response, apiErr = r.client.GetDomainList(ctx)
		return apiErr
	})

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

func (r *domainListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every domain in the Spaceship account, with the same details for each entry as the `spaceship_domain_info` data source.",
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx),
		},
		Attributes: map[string]schema.Attribute{
			"total": schema.Int64Attribute{
				Computed:    true,
				Description: "Total number of domains in the account.",
			},
			"items": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Details of each domain in the account.",
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

	pd, ok := req.ProviderData.(*providerData)

	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *providerData, got %T", req.ProviderData))
		return
	}

	r.client = pd.Client
}
