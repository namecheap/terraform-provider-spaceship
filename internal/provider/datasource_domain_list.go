package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewDomainListDataSource() datasource.DataSource {
	return &domainListDataSource{}
}

type domainListDataSource struct {
	client *Client
}

func (r *domainListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_list"
}

func (r *domainListDataSource) Read(_ context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

}

func (r *domainListDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides full list of domain in account with domain details for each domain",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":              schema.StringAttribute{Computed: true},
						"unicode_name":      schema.StringAttribute{Computed: true},
						"is_premium":        schema.BoolAttribute{Computed: true},
						"auto_renew":        schema.BoolAttribute{Computed: true},
						"registration_date": schema.StringAttribute{Computed: true},
						"expiration_date":   schema.StringAttribute{Computed: true},
						"lifecycle_status":  schema.StringAttribute{Computed: true, Description: "One of creating registered grace1 grace2 redemption"},

						"epp_statuses": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "Possible values clientDeleteProhibited clientHold clientRenewProhibited clientTransferProhibited clientUpdateProhibited",
						},

						"suspensions": schema.ListNestedAttribute{
							Computed:    true,
							Description: "Information about domain suspensions. May contain up to 2 items.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"reason_code": schema.StringAttribute{
										Computed:    true,
										Description: "Suspension reason code (raaVerification, abuse, promoAbuse, fraud, pendingAccountVerification, unauthorizedAccess, tosViolation, transferDispute, restrictedSecurity, lockCourt, suspendCourt, udrpUrs, restrictedLegal, paymentPending, unpaidService, restrictedWhois, lockedWhois)",
									},
								},
							},
						},

						"privacy_protection": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"contact_form": schema.BoolAttribute{
									Computed: true,
								},
								"level": schema.StringAttribute{
									Computed:    true,
									Description: "Privacy level: public or high",
								},
							},
						},
						"nameservers": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"provider": schema.StringAttribute{
									Computed:    true,
									Description: "type: basic or custom",
								},
								"hosts": schema.ListAttribute{
									Computed:    true,
									ElementType: types.StringType,
								},
							},
						},
						"contact": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"registrant": schema.StringAttribute{Computed: true},
								"admin":      schema.StringAttribute{Computed: true},
								"tech":       schema.StringAttribute{Computed: true},
								"billing":    schema.StringAttribute{Computed: true},
								"attributes": schema.ListAttribute{
									Computed:    true,
									ElementType: types.StringType,
								},
							},
						},
					},
				},
			},
		},
	}
}
