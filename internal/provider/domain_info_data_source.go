package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/namecheap/go-spaceship-sdk/client"
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

	var configTimeouts timeouts.Value
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("timeouts"), &configTimeouts)...)

	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, timeoutDiags := configTimeouts.Read(ctx, domainReadTimeout)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	var response client.DomainInfo
	err := withRetry(ctx, "read domain info", func() error {
		var apiErr error
		response, apiErr = d.client.GetDomainInfo(ctx, domain.ValueString())
		return apiErr
	})

	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain info", err.Error())
		return
	}

	domainDetails, domainDiags := buildDomainModel(ctx, response)
	resp.Diagnostics.Append(domainDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data := domainInfoModel{Domain: domain, Timeouts: configTimeouts}
	data.domainModel = domainDetails

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (d *domainInfoDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := domainAttributes()
	attrs["domain"] = schema.StringAttribute{
		Required:    true,
		Description: "The domain name to look up (for example example.com).",
		Validators: []validator.String{
			stringvalidator.LengthBetween(4, 255),
		},
	}

	resp.Schema = schema.Schema{
		Description: "Reads the details of a single domain in the Spaceship account: registration and expiration dates, nameserver delegation, contacts, privacy protection, lifecycle and verification status.",
		Attributes:  attrs,
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx),
		},
	}
}

func (d *domainInfoDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
