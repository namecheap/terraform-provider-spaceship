package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-spaceship/internal/provider/records"

	"github.com/namecheap/go-spaceship-sdk/client"
)

var (
	_ resource.Resource                = &personalNameserverResource{}
	_ resource.ResourceWithConfigure   = &personalNameserverResource{}
	_ resource.ResourceWithImportState = &personalNameserverResource{}
)

func NewPersonalNameserverResource() resource.Resource {
	return &personalNameserverResource{}
}

type personalNameserverResource struct {
	client *client.Client
}

type personalNameserverResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Domain types.String `tfsdk:"domain"`
	Host   types.String `tfsdk:"host"`
	IPs    types.Set    `tfsdk:"ips"`
}

func (r *personalNameserverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_personal_nameserver"
}

func (r *personalNameserverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single personal nameserver host (a registry glue record) for a Spaceship-managed domain. A personal nameserver is a host label (e.g. `ns1`) relative to the domain, plus the set of IP addresses the registry serves for it. Pointing the domain at these hosts is configured separately via the domain's `nameservers` block.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier with the form `domain/host`.",
				PlanModifiers: []planmodifier.String{
					// id derives from domain/host. host is mutable (rename), so id
					// can change in place — UseStateForUnknown would freeze the old
					// value and break the rename. Compute it from the plan instead.
					personalNameserverIDPlanModifier{},
				},
			},
			"domain": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain the personal nameserver host belongs to (for example `example.com`). Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"host": schema.StringAttribute{
				Required: true,
				// Intentionally no rejection of a host that ends with the domain
				// (e.g. host = "ns1.example.com" for example.com, which the API
				// accepts and turns into the glue host ns1.example.com.example.com).
				// Dotted hosts are valid input — "ns1.sub" creates the multi-level
				// glue host ns1.sub.example.com — so any such check would be a
				// heuristic stricter than the API's own rules. We warn in the
				// registry docs (see templates/resources/personal_nameserver.md.tmpl)
				// and in the description below instead.
				MarkdownDescription: "The host label of the nameserver, relative to `domain` (for example `ns1`, not `ns1.example.com`). The registry joins the label and the domain to form the full host name `ns1.example.com`; supplying a fully qualified name here is accepted by the API but produces the almost certainly unintended glue host `ns1.example.com.example.com`. Changing this renames the host in place via the API.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"ips": schema.SetAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The glue record IP addresses (IPv4 or IPv6) served for this host. Must contain between 1 and 16 addresses.",
				Validators: []validator.Set{
					setvalidator.SizeBetween(1, 16),
					setvalidator.ValueStringsAre(records.IPAddressValidator()),
				},
			},
		},
	}
}

func (r *personalNameserverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *personalNameserverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured provider", "The Spaceship provider was not configured. Please ensure the provider block is present.")
		return
	}

	var plan personalNameserverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.Domain.ValueString()
	host := plan.Host.ValueString()

	ns, diags := r.expand(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create and rename share one endpoint; on create the path host equals the
	// body host.
	result, err := r.client.UpsertPersonalNameserver(ctx, domain, host, ns)
	if err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to create personal nameserver: %s", err))
		return
	}

	resp.Diagnostics.Append(r.hydrate(ctx, &plan, domain, result)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *personalNameserverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured provider", "The Spaceship provider was not configured. Please ensure the provider block is present.")
		return
	}

	var state personalNameserverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, host, ok := parsePersonalNameserverID(state.ID.ValueString())
	if !ok {
		resp.Diagnostics.AddError("Invalid resource ID", fmt.Sprintf("Expected format domain/host, got %q", state.ID.ValueString()))
		return
	}

	// The single-host GET is under development (HTTP 501), so FindPersonalNameserver
	// reads the working list endpoint and filters by host. See the TODO(api-501)
	// note on FindPersonalNameserver for the future switch to the direct endpoint.
	ns, err := r.client.FindPersonalNameserver(ctx, domain, host)
	// Two ways this resource can be gone: the host is absent from an existing
	// domain's list (ErrPersonalNameserverNotFound), or the domain itself was
	// deleted and the list endpoint 404s (a raw SpaceshipApiError). Treat both
	// as "removed" so Terraform can reconcile instead of wedging on an error.
	if errors.Is(err, client.ErrPersonalNameserverNotFound) || client.IsNotFoundError(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to read personal nameserver for %s: %s", domain, err))
		return
	}

	resp.Diagnostics.Append(r.hydrate(ctx, &state, domain, ns)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *personalNameserverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured provider", "The Spaceship provider was not configured. Please ensure the provider block is present.")
		return
	}

	var plan, state personalNameserverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.Domain.ValueString()

	ns, diags := r.expand(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The PUT path carries the current (state) host while the body carries the
	// desired (plan) host, so a host change renames in place and an IP-only
	// change updates the same host.
	result, err := r.client.UpsertPersonalNameserver(ctx, domain, state.Host.ValueString(), ns)
	if err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to update personal nameserver: %s", err))
		return
	}

	resp.Diagnostics.Append(r.hydrate(ctx, &plan, domain, result)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *personalNameserverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured provider", "The Spaceship provider was not configured. Please ensure the provider block is present.")
		return
	}

	var state personalNameserverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeletePersonalNameserver(ctx, state.Domain.ValueString(), state.Host.ValueString()); err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to delete personal nameserver: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *personalNameserverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import string is the composite ID (domain/host). Read parses it and
	// hydrates the remaining attributes.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// expand converts the plan model into a client struct and validates it,
// surfacing constraint violations as attribute diagnostics.
func (r *personalNameserverResource) expand(ctx context.Context, model personalNameserverResourceModel) (client.PersonalNameserver, diag.Diagnostics) {
	var diags diag.Diagnostics

	var ips []string
	diags.Append(model.IPs.ElementsAs(ctx, &ips, false)...)
	if diags.HasError() {
		return client.PersonalNameserver{}, diags
	}

	ns := client.PersonalNameserver{
		Host: model.Host.ValueString(),
		IPs:  ips,
	}

	// Route each field's validation error to its attribute path so the CLI can
	// point at the offending input. Validate them individually rather than via
	// the aggregate ns.Validate(), whose flat []error drops the field origin.
	if err := ns.ValidateHost(); err != nil {
		diags.AddAttributeError(path.Root("host"), "Invalid host", err.Error())
	}
	if err := ns.ValidateIPs(); err != nil {
		diags.AddAttributeError(path.Root("ips"), "Invalid IP addresses", err.Error())
	}

	if diags.HasError() {
		return client.PersonalNameserver{}, diags
	}
	return ns, diags
}

// hydrate writes the API result back onto the model, including the composite ID.
func (r *personalNameserverResource) hydrate(ctx context.Context, model *personalNameserverResourceModel, domain string, ns client.PersonalNameserver) diag.Diagnostics {
	ips, diags := types.SetValueFrom(ctx, types.StringType, ns.IPs)
	if diags.HasError() {
		return diags
	}

	model.ID = types.StringValue(personalNameserverID(domain, ns.Host))
	model.Domain = types.StringValue(domain)
	model.Host = types.StringValue(ns.Host)
	model.IPs = ips
	return diags
}

// personalNameserverIDPlanModifier sets the planned id to domain/host computed
// from the plan. Both components are Required and known at plan time, so the
// planned id matches what Create/Update will write — avoiding the "inconsistent
// result after apply" error that UseStateForUnknown would cause on a rename.
type personalNameserverIDPlanModifier struct{}

func (personalNameserverIDPlanModifier) Description(_ context.Context) string {
	return "Computes id as domain/host from the planned values."
}

func (m personalNameserverIDPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (personalNameserverIDPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var domain, host types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("domain"), &domain)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("host"), &host)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// On destroy the plan is null, and either component may be unknown when fed
	// by an interpolated value. Leave id unknown (known after apply) in those cases.
	if domain.IsNull() || domain.IsUnknown() || host.IsNull() || host.IsUnknown() {
		return
	}

	resp.PlanValue = types.StringValue(personalNameserverID(domain.ValueString(), host.ValueString()))
}

// personalNameserverID builds the composite Terraform identifier. domain is a
// dotted name and host a label; neither contains "/", so a 2-part split recovers them.
func personalNameserverID(domain, host string) string {
	return domain + "/" + host
}

// parsePersonalNameserverID is the inverse of personalNameserverID.
func parsePersonalNameserverID(id string) (domain, host string, ok bool) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
