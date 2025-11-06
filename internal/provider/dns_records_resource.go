package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &dnsRecordsResource{}
	_ resource.ResourceWithConfigure   = &dnsRecordsResource{}
	_ resource.ResourceWithImportState = &dnsRecordsResource{}

	// todo double check from api specs
	portStringPattern      = regexp.MustCompile(`^(\*|_[0-9]{1,5})$`)
	schemeLabelPattern     = regexp.MustCompile(`^_[A-Za-z0-9-]+$`)
	tlsaAssociationPattern = regexp.MustCompile(`^[0-9a-fA-F]{2}(\s?[0-9a-fA-F]{2})*$`)
)

func NewDNSRecordsResource() resource.Resource {
	return &dnsRecordsResource{}
}

type dnsRecordsResource struct {
	client *Client
}

type dnsRecordsResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Domain types.String `tfsdk:"domain"`
	// what is force?
	// TODO
	Force   types.Bool `tfsdk:"force"`
	Records types.List `tfsdk:"records"`
}

type dnsRecordModel struct {
	Type            types.String `tfsdk:"type"`
	Name            types.String `tfsdk:"name"`
	TTL             types.Int64  `tfsdk:"ttl"`
	Address         types.String `tfsdk:"address"`
	AliasName       types.String `tfsdk:"alias_name"`
	CName           types.String `tfsdk:"cname"`
	Flag            types.Int64  `tfsdk:"flag"`
	Tag             types.String `tfsdk:"tag"`
	Value           types.String `tfsdk:"value"`
	Port            types.String `tfsdk:"port"`
	Scheme          types.String `tfsdk:"scheme"`
	SvcPriority     types.Int64  `tfsdk:"svc_priority"`
	TargetName      types.String `tfsdk:"target_name"`
	SvcParams       types.String `tfsdk:"svc_params"`
	Exchange        types.String `tfsdk:"exchange"`
	Preference      types.Int64  `tfsdk:"preference"`
	Nameserver      types.String `tfsdk:"nameserver"`
	Pointer         types.String `tfsdk:"pointer"`
	Service         types.String `tfsdk:"service"`
	Protocol        types.String `tfsdk:"protocol"`
	Priority        types.Int64  `tfsdk:"priority"`
	Weight          types.Int64  `tfsdk:"weight"`
	PortNumber      types.Int64  `tfsdk:"port_number"`
	Target          types.String `tfsdk:"target"`
	Usage           types.Int64  `tfsdk:"usage"`
	Selector        types.Int64  `tfsdk:"selector"`
	Matching        types.Int64  `tfsdk:"matching"`
	AssociationData types.String `tfsdk:"association_data"`
}

var dnsRecordObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":             types.StringType,
		"name":             types.StringType,
		"ttl":              types.Int64Type,
		"address":          types.StringType,
		"alias_name":       types.StringType,
		"cname":            types.StringType,
		"flag":             types.Int64Type,
		"tag":              types.StringType,
		"value":            types.StringType,
		"port":             types.StringType,
		"scheme":           types.StringType,
		"svc_priority":     types.Int64Type,
		"target_name":      types.StringType,
		"svc_params":       types.StringType,
		"exchange":         types.StringType,
		"preference":       types.Int64Type,
		"nameserver":       types.StringType,
		"pointer":          types.StringType,
		"service":          types.StringType,
		"protocol":         types.StringType,
		"priority":         types.Int64Type,
		"weight":           types.Int64Type,
		"port_number":      types.Int64Type,
		"target":           types.StringType,
		"usage":            types.Int64Type,
		"selector":         types.Int64Type,
		"matching":         types.Int64Type,
		"association_data": types.StringType,
	},
}

func (r *dnsRecordsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_records"
}

func (r *dnsRecordsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the full DNS record set for a Spaceship-managed domain. The Spaceship API updates records as a batch, so Terraform replaces the entire set on each apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal identifier for tf state.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain name to manage (for example `example.com`). The domain must already exist in the Spaceship account.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1), // TODO could be improved by valid regex
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"force": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Force Spaceship to apply the DNS update even if conflicts are detected. The Spaceship API requires this flag when overwriting existing records.",
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"records": schema.ListNestedAttribute{
				MarkdownDescription: "DNS records that should be configured for the domain. Each apply replaces the complete set of records.",
				Required:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "DNS record type(A, AAAA, ALIAS, CAA, CNAME, HTTPS, MX, NS, PTR, SRV, SVCB, TLSA, TXT).",
							Validators: []validator.String{
								// TODO export to one var and reuse
								stringvalidator.OneOf("A", "AAAA", "ALIAS", "CAA", "CNAME", "HTTPS", "MX", "NS", "PTR", "SRV", "SVCB", "TLSA", "TXT"),
							},
						},
						"name": schema.StringAttribute{
							Required: true,
							// TODO check description from api specs
							MarkdownDescription: "Record host. Use `@a` for the zone apex.",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						// double check all fields
						"ttl": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Record TTL in seconds. Defaults to 3600 if omitted.",
							Default:             int64default.StaticInt64(3600),
							Validators: []validator.Int64{
								int64validator.Between(60, 3600),
							},
						},
						"address": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "IPv4 or IPv6 address for A and AAAA records",
						},
						"alias_name": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Alias target for ALIAS records.",
						},
						"cname": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Canonical name for CNAME records.",
						},
						"flag": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Flag for CAA records (0 or 128).",
							Validators: []validator.Int64{
								int64validator.OneOf(0, 128),
							},
						},
						"tag": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Tag for CAA records (e.g. `issue`)",
						},
						"value": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Generic value field used by several record tyeps (CAA, TXT).",
						},
						"port": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Port for HTTPS, SVCB and TLSA records(accepts `*` or `_NNNN`).",
							Validators: []validator.String{
								stringvalidator.RegexMatches(portStringPattern, "must be '*' or an underscore followed by digits. "),
							},
						},
						"scheme": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Scheme for HTTPS/SVCB/TLSA records (for exampel `_https`, `_tcp`)",
							Validators: []validator.String{
								stringvalidator.RegexMatches(schemeLabelPattern, "must start with '_' and contain alphanumeric or '-' characters"),
							},
						},
						"svc_priority": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Service priority for HTTPS/SVCB records (0-65535).",
							Validators: []validator.Int64{
								int64validator.Between(0, 65535),
							},
						},
						"target_name": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Target name for HTTPS/SVCB records.",
						},
						"svc_params": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "SvcParams string for HTTPS/SVCB records.",
						},
						"exchange": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Mail exchange host for MX records.",
						},
						"preference": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Preference value for MX records (0-65535).",
							Validators: []validator.Int64{
								int64validator.Between(0, 65535),
							},
						},
						"nameserver": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Nameserver host for NS records.",
						},
						"pointer": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Pointer target for PTR records.",
						},
						"service": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Service label for SRV records (for example `_sip`).",
						},
						"protocol": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Protocol label for SRV/TLSA records (e.g. `_tcp`).",
							Validators: []validator.String{
								stringvalidator.RegexMatches(schemeLabelPattern, "must start with '_' and contain alphanumeric or '-' characters"),
							},
						},
						"priority": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Priority for SRV records (0-65535).",
							Validators: []validator.Int64{
								int64validator.Between(0, 65535),
							},
						},
						"weight": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Weight for SRV records (0-65535).",
							Validators: []validator.Int64{
								int64validator.Between(0, 65535),
							},
						},
						"port_number": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Port for SRV records (1-65535).",
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"target": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Target host for SRV records.",
						},
						"usage": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Usage value for TLSA records (0-255).",
							Validators: []validator.Int64{
								int64validator.Between(0, 255),
							},
						},
						"selector": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Selector value for TLSA records (0-255).",
							Validators: []validator.Int64{
								int64validator.Between(0, 255),
							},
						},
						"matching": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Matching type for TLSA records (0-255).",
							Validators: []validator.Int64{
								int64validator.Between(0, 255),
							},
						},
						"association_data": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Association data (hex) for TLSA records.",
							Validators: []validator.String{
								stringvalidator.RegexMatches(tlsaAssociationPattern, "must be a hex string (optionally spaced)"),
							},
						},
					},
				},
			},
		},
	}
}

func (r *dnsRecordsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *Client, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *dnsRecordsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured provider", "The Spaceship provider was not configured. Please ensure the provider block is present")
		return
	}

	var plan dnsRecordsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	force := boolOrDefault(plan.Force, true)
	desiredRecords, diags := expandDNSRecords(ctx, plan.Records, path.Root("records"))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	existingRecords, err := r.client.GetDNSRecords(ctx, plan.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("failed to read existing DNS records: %s", err))
		return
	}

	toDelete, toUpsert := diffDNSRecords(existingRecords, desiredRecords)
	if err := r.client.DeleteDNSRecords(ctx, plan.Domain.ValueString(), toDelete); err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to delete DNS records: %s", err))
		return
	}

	if len(toUpsert) > 0 {
		if err := r.client.UpsertDNSRecords(ctx, plan.Domain.ValueString(), force, toUpsert); err != nil {
			resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Fail;ed to apply DNS records: %s", err))
			return
		}
	}

	updatedRecords, err := r.client.GetDNSRecords(ctx, plan.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to refresh DNS records: %s", err))
		return
	}

	orderedRecords := orderDNSRecordsLike(desiredRecords, updatedRecords)

	flattened, diags := flattenDNSRecords(ctx, orderedRecords)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString())
	plan.Force = types.BoolValue(force)
	plan.Records = flattened

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsRecordsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		// TODO looks like repeated part
		resp.Diagnostics.AddError("Unconfigured provider", "The Spaceship provider was not configured. Please ensure the provider block is present")
		return
	}

	var state dnsRecordsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateRecords, diags := expandDNSRecords(ctx, state.Records, path.Root("records"))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiRecords, err := r.client.GetDNSRecords(ctx, state.Domain.ValueString())
	if err != nil {
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to read DNS records: %s", err))
		return
	}

	orderedRecords := orderDNSRecordsLike(stateRecords, apiRecords)

	flattenedRecords, diags := flattenDNSRecords(ctx, orderedRecords)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Records = flattenedRecords

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsRecordsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured provider", "The Spaceship provider was not configured. Please ensure the provider block is present.")
		return
	}

	var plan dnsRecordsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	force := boolOrDefault(plan.Force, true)
	desiredRecords, diags := expandDNSRecords(ctx, plan.Records, path.Root("records"))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	existingRecords, err := r.client.GetDNSRecords(ctx, plan.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("failed to read existing DNS Records: %s", err))
		return
	}

	toDelete, toUpsert := diffDNSRecords(existingRecords, desiredRecords)

	if err := r.client.DeleteDNSRecords(ctx, plan.Domain.ValueString(), toDelete); err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to delete DNS records: %s", err))
		return
	}

	if len(toUpsert) > 0 {
		if err := r.client.UpsertDNSRecords(ctx, plan.Domain.ValueString(), force, toUpsert); err != nil {
			resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to update DNS records: %s", err))
			return
		}
	}

	updatedRecords, err := r.client.GetDNSRecords(ctx, plan.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to refresh DNS records: %s", err))
		return
	}

	orderedRecords := orderDNSRecordsLike(desiredRecords, updatedRecords)

	flattened, diags := flattenDNSRecords(ctx, orderedRecords)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString())
	plan.Force = types.BoolValue(force)
	plan.Records = flattened
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsRecordsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// should be extracted somewhere
	// I am tired of typing it
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured provider", "The Spaceship provider was not configured. Please ensure the provider block is present.")
		return
	}

	var state dnsRecordsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	//

	force := boolOrDefault(state.Force, true)
	// why it is here?
	if err := r.client.ClearDNSRecords(ctx, state.Domain.ValueString(), force); err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to clear DNS records: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *dnsRecordsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resourceID := req.ID

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), resourceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), resourceID)...)
}

// here goes important part
func expandDNSRecords(ctx context.Context, list types.List, listPath path.Path) ([]DNSRecord, diag.Diagnostics) {
	var diags diag.Diagnostics

	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	var models []dnsRecordModel
	diags = list.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}

	records := make([]DNSRecord, 0, len(models))

	for idx, item := range models {
		recordPath := listPath.AtListIndex(idx)

		recordType := strings.ToUpper(strings.TrimSpace(item.Type.ValueString()))
		if recordType == "" {
			diags.AddAttributeError(recordPath.AtName("type"), "Missing record type", "Each DNS record must specify a type (e.g. A, MX, TXT).")
			continue
		}

		name := strings.TrimSpace(item.Name.ValueString())
		if name == "" {
			diags.AddAttributeError(recordPath.AtName("name"), "Missing record name", "Each DNS record must specify a name (use '@' for the apex).")
			continue
		}

		// why it is not coming from default value from terraform schema in one place?
		ttl := int64(3600)
		if !item.TTL.IsNull() && !item.TTL.IsUnknown() {
			ttl = item.TTL.ValueInt64()
		}

		record := DNSRecord{
			Type: recordType,
			Name: name,
			TTL:  int(ttl),
		}

		valid := true

		getString := func(value types.String, attrName string) (string, bool) {
			if value.IsNull() || value.IsUnknown() {
				return "", false
			}
			return value.ValueString(), true
		}

		requireString := func(value types.String, attrName, description string) (string, bool) {
			if value.IsNull() || value.IsUnknown() || strings.TrimSpace(value.ValueString()) == "" {
				diags.AddAttributeError(recordPath.AtName(attrName), fmt.Sprintf("Missing %s", attrName), description)
				return "", false
			}
			return value.ValueString(), true
		}

		requireInt := func(value types.Int64, attrName, description string) (int, bool) {
			if value.IsNull() || value.IsUnknown() {
				diags.AddAttributeError(recordPath.AtName(attrName), fmt.Sprintf("Missing %s", attrName), description)
				return 0, false
			}
			return int(value.ValueInt64()), true
		}

		switch recordType {
		case "A", "AAAA":
			if addr, ok := requireString(item.Address, "address", "Rerocrds of this type require the `address` attributes."); ok {
				record.Address = addr
			} else {
				valid = false
			}

		case "ALIAS":
			if alias, ok := requireString(item.AliasName, "alias_name", "ALIAS records require the `alias_name` attribute."); ok {
				record.AliasName = alias
			} else {
				valid = false
			}

		case "CAA":
			if flag, ok := requireInt(item.Flag, "flag", "CAA records require the `flag` attribute (0 or 128)"); ok {
				record.Flag = &flag
			} else {
				valid = false
			}

			if tag, ok := requireString(item.Tag, "tag", "CAA records require the `tag` attribute (e.g. `issue`)"); ok {
				record.Tag = tag
			} else {
				valid = false
			}

			if value, ok := requireString(item.Value, "value", "CAA records require the `value` attribute"); ok {
				record.Value = value
			} else {
				valid = false
			}
		case "CNAME":
			if cname, ok := requireString(item.CName, "cname", "CNAME records require the `cname` attribute"); ok {
				record.CName = cname
			} else {
				valid = false
			}

		case "HTTPS":
			if pri, ok := requireInt(item.SvcPriority, "svc_priority", "HTTPS records require the `scv_priority` attribute"); ok {
				record.SvcPriority = &pri
			} else {
				valid = false
			}
			if target, ok := requireString(item.TargetName, "target_name", "HTTPS records require the `target_name` attribute"); ok {
				record.TargetName = target
			} else {
				valid = false
			}
			if params, ok := getString(item.SvcParams, "svc_params"); ok {
				record.SvcParams = params
			} else {
				record.SvcParams = ""
			}
			if port, ok := getString(item.Port, "port"); ok {
				record.Port = NewStringPortValue(port)
				if _, hasScheme := getString(item.Scheme, "scheme"); !hasScheme {
					diags.AddAttributeError(recordPath.AtName("scheme"), "Missing scheme", "HTTPS records that specify `port` must also set `scheme` (usually `_https`)")
					valid = false
				}
			}
			if scheme, ok := getString(item.Scheme, "scheme"); ok {
				record.Scheme = scheme
			}
		case "MX":
			if exchange, ok := requireString(item.Exchange, "exchange", "MX records require the `exchange` attribute(mail server hostname)"); ok {
				record.Exchange = exchange
			} else {
				valid = false
			}
			if pref, ok := requireInt(item.Preference, "preference", "MX records require the `preference` attribute (0-65536)."); ok {
				record.Preference = &pref
			} else {
				valid = false
			}

		case "NS":
			if ns, ok := requireString(item.Nameserver, "nameserver", "NS records require the `nameserver` attribute."); ok {
				record.Nameserver = ns
			} else {
				valid = false
			}

		case "PTR":
			if pointer, ok := requireString(item.Pointer, "pointer", "PTR records require the `pointer` attribute."); ok {
				record.Pointer = pointer
			} else {
				valid = false
			}

		case "SRV":
			if service, ok := requireString(item.Service, "service", "SRV records require the `service`, attribute (for example `_sip`)."); ok {
				record.Service = service
			} else {
				valid = false
			}
			if protocol, ok := requireString(item.Protocol, "protocol", "SRV records require the `protocol` attribute (e.g. `_tcp`)"); ok {
				record.Protocol = protocol
			} else {
				valid = false
			}
			if priority, ok := requireInt(item.Priority, "priority", "SRV records require the `priority` attribute (0-65535)."); ok {
				record.Priority = &priority
			} else {
				valid = false
			}
			if weight, ok := requireInt(item.Weight, "weight", "SRV records require the `weight` attribute(0-65535)."); ok {
				record.Weight = &weight
			} else {
				valid = false
			}
			if port, ok := requireInt(item.PortNumber, "port_number", "SRV records require the `port_number` attribute(1-65535)."); ok {
				record.Port = NewIntPortValue(port)
			} else {
				valid = false
			}
			if target, ok := requireString(item.Target, "target", "SRV recrods reqiure the `target` attriabute."); ok {
				record.Target = target
			} else {
				valid = false
			}

		case "SVCB":
			if pri, ok := requireInt(item.SvcPriority, "svc_priority", "SVCB records require the `svc_priority` attributre(0-65535)."); ok {
				record.SvcPriority = &pri
			} else {
				valid = false
			}
			if target, ok := requireString(item.TargetName, "target_name", "SVCB records require the `target` attribute."); ok {
				record.TargetName = target
			} else {
				valid = false
			}
			if params, ok := getString(item.SvcParams, "svc_params"); ok {
				record.SvcParams = params
			} else {
				record.SvcParams = ""
			}
			if port, ok := getString(item.Port, "port"); ok {
				record.Port = NewStringPortValue(port)
			}
			if scheme, ok := getString(item.Scheme, "scheme"); ok {
				record.Scheme = scheme
			}

		case "TLSA":
			if port, ok := requireString(item.Port, "port", "TLSA records require the `port` attribute (for example `_443`)."); ok {
				record.Port = NewStringPortValue(port)
			} else {
				valid = false
			}
			if protocol, ok := requireString(item.Protocol, "protocol", "TLSA records require the `protocol` attribute (e.g. `_tcp`)"); ok {
				record.Protocol = protocol
			} else {
				valid = false
			}
			if usage, ok := requireInt(item.Usage, "usage", "TLSA records require the `usage` attribute (0-255)"); ok {
				record.Usage = &usage
			} else {
				valid = false
			}
			if selector, ok := requireInt(item.Selector, "selector", "TLSA records require the `selector` attribute(0-255)"); ok {
				record.Selector = &selector
			} else {
				valid = false
			}
			if matching, ok := requireInt(item.Matching, "matching", "TLSA records require the `matching` attribute(0-255)."); ok {
				record.Matching = &matching
			} else {
				valid = false
			}
			if assoc, ok := requireString(item.AssociationData, "association_data", "TLSA records require the `association_data` attribute containign the certificate associations"); ok {
				record.AssociationData = assoc
			} else {
				valid = false
			}
			if scheme, ok := getString(item.Scheme, "scheme"); ok {
				record.Scheme = scheme
			}

		case "TXT":
			if value, ok := requireString(item.Value, "value", "TXT records require the `value` attribute."); ok {
				record.Value = value
			} else {
				valid = false
			}
		default:
			diags.AddAttributeError(recordPath.AtName("type"), "Unsupported record type", fmt.Sprintf("Type %q is not supported by the provider.", recordType))
			continue
		}

		if valid {
			records = append(records, record)
		}

	}
	return records, diags
}

func flattenDNSRecords(ctx context.Context, records []DNSRecord) (types.List, diag.Diagnostics) {
	elements := make([]dnsRecordModel, 0, len(records))
	for _, record := range records {
		model := dnsRecordModel{
			Type: types.StringValue(record.Type),
			Name: types.StringValue(record.Name),
		}

		if record.TTL > 0 {
			model.TTL = types.Int64Value(int64(record.TTL))
		} else {
			model.TTL = types.Int64Null()
		}

		stringOrNull := func(value string) types.String {
			if value == "" {
				return types.StringNull()
			}
			return types.StringValue(value)
		}

		intPointerOrNull := func(value *int) types.Int64 {
			if value == nil {
				return types.Int64Null()
			}
			return types.Int64Value(int64(*value))
		}

		model.Address = stringOrNull(record.Address)
		model.AliasName = stringOrNull(record.AliasName)
		model.CName = stringOrNull(record.CName)
		model.Flag = intPointerOrNull(record.Flag)
		model.Tag = stringOrNull(record.Tag)
		model.Value = stringOrNull(record.Value)

		if record.Port != nil {
			if record.Port.String != nil {
				model.Port = stringOrNull(*record.Port.String)
			}

			if record.Port.Int != nil {
				model.PortNumber = types.Int64Value(int64(*record.Port.Int))
			}
		}

		model.Scheme = stringOrNull(record.Scheme)
		model.SvcPriority = intPointerOrNull(record.SvcPriority)
		model.TargetName = stringOrNull(record.TargetName)
		model.SvcParams = stringOrNull(record.SvcParams)
		model.Exchange = stringOrNull(record.Exchange)
		model.Preference = intPointerOrNull(record.Preference)
		model.Nameserver = stringOrNull(record.Nameserver)
		model.Pointer = stringOrNull(record.Pointer)
		model.Service = stringOrNull(record.Service)
		model.Protocol = stringOrNull(record.Protocol)
		model.Priority = intPointerOrNull(record.Priority)
		model.Weight = intPointerOrNull(record.Weight)
		model.Target = stringOrNull(record.Target)
		model.Usage = intPointerOrNull(record.Usage)
		model.Selector = intPointerOrNull(record.Selector)
		model.Matching = intPointerOrNull(record.Matching)
		model.AssociationData = stringOrNull(record.AssociationData)

		elements = append(elements, model)
	}

	return types.ListValueFrom(ctx, dnsRecordObjectType, elements)
}

func diffDNSRecords(existing, desired []DNSRecord) (toDelete, toUpsert []DNSRecord) {
	desiredMap := make(map[string]DNSRecord, len(desired))
	for _, record := range desired {
		desiredMap[recordKey(record)] = record
	}

	existingMap := make(map[string]DNSRecord, len(existing))
	for _, record := range existing {
		existingMap[recordKey(record)] = record
		if _, ok := desiredMap[recordKey(record)]; !ok {
			toDelete = append(toDelete, record)
		}
	}

	seen := make(map[string]struct{})

	for _, record := range desired {
		key := recordKey(record)
		existingRecord, ok := existingMap[key]
		if ok && existingRecord.TTL == record.TTL && recordValueSignature(existingRecord) == recordValueSignature(record) {
			continue
		}

		if _, already := seen[key]; already {
			continue
		}

		toUpsert = append(toUpsert, record)
		seen[key] = struct{}{}
	}

	return toDelete, toUpsert
}

func recordKey(record DNSRecord) string {
	return strings.ToUpper(record.Type) + "|" + strings.ToLower(record.Name) + "|" + recordValueSignature(record)
}

func orderDNSRecordsLike(reference, records []DNSRecord) []DNSRecord {
	if len(records) <= 1 || len(reference) == 0 {
		return records
	}

	type keyedRecord struct {
		key    string
		record DNSRecord
		used   bool
	}

	keyed := make([]keyedRecord, len(records))
	for i, record := range records {
		keyed[i] = keyedRecord{
			key:    recordKey(record),
			record: record,
		}
	}

	ordered := make([]DNSRecord, 0, len(records))

	for _, ref := range reference {
		key := recordKey(ref)
		for i := range keyed {
			if keyed[i].used {
				continue
			}
			if keyed[i].key == key {
				ordered = append(ordered, keyed[i].record)
				keyed[i].used = true
				break
			}
		}
	}

	for i := range keyed {
		if !keyed[i].used {
			ordered = append(ordered, keyed[i].record)
		}
	}

	return ordered
}

func recordValueSignature(record DNSRecord) string {
	var builder strings.Builder
	write := func(parts ...string) {
		for _, part := range parts {
			if builder.Len() > 0 {
				builder.WriteString("|")
			}
			builder.WriteString(part)
		}
	}

	switch strings.ToUpper(record.Type) {
	case "A", "AAAA":
		write(strings.ToLower(record.Address))
	case "ALIAS":
		write(strings.ToLower(record.AliasName))
	case "CAA":
		write(intToString(record.Flag), strings.ToLower(record.Tag), record.Value)
	case "CNAME":
		write(strings.ToLower(record.CName))
	case "HTTPS":
		// TODO looks like a bug  when svc prirotity is present in https record
		write(intToString(record.SvcPriority), strings.ToLower(record.TargetName), record.SvcParams, portValueSignature(record.Port), strings.ToLower(record.Scheme))
	case "MX":
		write(strings.ToLower(record.Exchange), intToString(record.Preference))
	case "NS":
		write(strings.ToLower(record.Nameserver))
	case "PTR":
		write(strings.ToLower(record.Pointer))
	case "SRV":
		write(strings.ToLower(record.Service), strings.ToLower(record.Protocol), intToString(record.Priority), intToString(record.Weight))
	case "SVCB":
		write(intToString(record.SvcPriority), strings.ToLower(record.TargetName), record.SvcParams, portValueSignature(record.Port), strings.ToLower(record.Scheme))
	case "TLSA":
		normalized := strings.ReplaceAll(strings.ToLower(record.AssociationData), " ", "")
		write(portValueSignature(record.Port), strings.ToLower(record.Protocol), intToString(record.Usage), intToString(record.Selector), intToString(record.Matching), normalized)
	case "TXT":
		write(record.Value)
	default:
		write(record.Address)
	}
	return builder.String()
}

func intToString(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func portValueSignature(port *PortValue) string {
	if port == nil {
		return ""
	}
	if port.Int != nil {
		return fmt.Sprintf("%d", *port.Int)
	}
	if port.String != nil {
		return strings.ToLower(*port.String)
	}
	return ""
}

func boolOrDefault(value types.Bool, fallback bool) bool {
	if value.IsNull() || value.IsUnknown() {
		return fallback
	}
	return value.ValueBool()
}
