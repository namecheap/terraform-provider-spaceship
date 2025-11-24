package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func NewDomainInfoDataSource() datasource.DataSource {
	return &domainInfoDataSource{}
}

type domainInfoDataSource struct {
	client *Client
}

func (d *domainInfoDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_info"
}

func (d *domainInfoDataSource) Read(_ context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
}

func (d *domainInfoDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {

}
