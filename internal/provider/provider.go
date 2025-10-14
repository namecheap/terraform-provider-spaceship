package provider

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const defaultBaseURL = "https://spaceship.dev/api/v1"

// ensure spaceship provider satisfies expected interfaces
var _ provider.Provider = &spaceshipProvider{}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &spaceshipProvider{
			version: version,
		}
	}
}

type spaceshipProvider struct {
	version string
}

type providerModel struct {
	APIKey    types.String `tfsdk:"apk_key"`
	APISecret types.String `tfsdk:"api_secret"`
}

func (p *spaceshipProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "spaceship"
	resp.Version = p.version
}

func (p *spaceshipProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Spaceship provider is used to manage DNS records for domains managed by the Spaceship registrar",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Spaceship API key. If omitted, the provider will attempt to read the value from the `SPACESHIP_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"api_secret": schema.StringAttribute{
				MarkdownDescription: "Spaceship API secret. If omitted, the provider will attempt to read the value from the `SPACESHIP_API_SECRET` environment variable.",
				Optional:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (p *spaceshipProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := resolveString(config.APIKey, "SPACESHIP_API_KEY")
	apiSecret := resolveString(config.APISecret, "SPACESHIP_API_SECRET")

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Spaceship API key",
			"The provider cannot create the Spaceship API client without an API key. "+
				"Set the `api_key` attribute or configure the SPACESHIP_API_KEY environment variable.",
		)
	}

	if apiSecret == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_secret"),
			"Missing Spaceship API secret",
			"The provider cannot create the Spaceship API client without an API secret. "+
				"Set the `api_secret` attribute or configure the SPACESHIP_API_SECRET environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client := NewClient(defaultBaseURL, apiKey, apiSecret)
	//map of any type could be improved at least in this case?
	// or it will not work for empty interface?
	tflog.Info(ctx, "Configured Spaceship provider", map[string]any{
		"base_url": defaultBaseURL,
	})

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *spaceshipProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDNSRecordsResource,
	}
}

func (p *spaceshipProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func resolveString(value types.String, envVar string) string {
	if !value.IsNull() && !value.IsUnknown() {
		return value.ValueString()
	}

	return strings.TrimSpace(os.Getenv(envVar))
}
