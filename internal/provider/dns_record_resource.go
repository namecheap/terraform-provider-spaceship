package provider

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
			MarkdownDescription: "Composite identifier with the form `domain/TYPE/name/<data-signature>`. The data signature is a normalized representation of the record's type-specific fields (lowercased, pipe-separated) and is the same key used internally for record matching. Stable across updates that don't change identity (e.g. TTL changes).",
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

	// The Spaceship API has no "update record data" operation — records are
	// matched by (type, name, data), so changing any of those produces a new
	// record. Every attribute except `ttl` triggers Replace; `ttl` is the sole
	// in-place mutable field and is handled by Update via the upsert endpoint.
	for attrName, attr := range attrs {
		if attrName == "id" || attrName == "ttl" {
			continue
		}
		attrs[attrName] = withRequiresReplace(attr)
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single DNS record for a Spaceship-managed domain. Only records in the `custom` DNS group are managed — records owned by Spaceship features (e.g. URL redirect, personal nameservers) are left untouched.",
		Attributes:          attrs,
	}
}

// withRequiresReplace appends a RequiresReplace plan modifier to a schema
// attribute. Used during Schema() construction to mark identity-bearing
// attributes — any change forces destroy+create because the Spaceship API
// cannot mutate record data in place.
func withRequiresReplace(attr schema.Attribute) schema.Attribute {
	switch a := attr.(type) {
	case schema.StringAttribute:
		a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.RequiresReplace())
		return a
	case schema.Int64Attribute:
		a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.RequiresReplace())
		return a
	default:
		return attr
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

	//todo why it is needed
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No "fetch existing record before creating" / adopt-on-create logic here
	// by design: the API's upsert endpoint (used by CreateDNSRecord below) is
	// idempotent for records with matching (type, name, data) — see the
	// docstring on client.CreateDNSRecord. A matching pre-existing record is
	// transparently adopted; only conflict cases (e.g. CNAME with a different
	// target at the same hostname) error, and `terraform import` is the right
	// path for those. Verified by TestAccDNSRecord_createWhenIdenticalExists.

	// modelToDNSRecord handles every supported record type: A, AAAA, ALIAS,
	// CAA, CNAME, HTTPS, MX, NS, PTR, SRV, SVCB, TLSA, TXT. It emits attribute
	// diagnostics for any missing per-type required field.
	record, recordDiags := modelToDNSRecord(plan.dnsRecordModel, path.Empty())
	resp.Diagnostics.Append(recordDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.CreateDNSRecord(ctx, domain, record); err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to create DNS record: %s", err))
		return
	}

	// Only `id` is computed — every other attribute came from the user's plan
	// and is already populated on `plan`. Setting other fields here would
	// corrupt state for record types whose data field isn't `address` (e.g.
	// a CNAME record would get plan.Address overwritten to "").
	plan.ID = types.StringValue(recordID(domain, record))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}

// recordID builds the Terraform identifier for a single DNS record.
// Format: domain/TYPE/name/<recordValueSignature>
// The signature is the same one used by recordKey() for in-memory diffing,
// so the ID is unique for any (domain, type, name, data) tuple the API
// treats as distinct. SplitN(id, "/", 4) recovers the segments — the data
// signature is last so it may safely contain "/" (e.g. inside a TXT value).
func recordID(domain string, record client.DNSRecord) string {
	return strings.Join([]string{
		domain,
		strings.ToUpper(record.Type),
		strings.ToLower(record.Name),
		client.RecordValueSignature(record),
	}, "/")
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

	// Build the API record from full state (which Read has hydrated from the
	// API). Every record type is supported via the shared model→record helper.
	// Note: the API matches records by (type, name, data) for delete — TTL is
	// not part of the match key, so its value here is irrelevant for matching.
	record, recordDiags := modelToDNSRecord(state.dnsRecordModel, path.Empty())
	resp.Diagnostics.Append(recordDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteDNSRecord(ctx, domain, record); err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to delete DNS record: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)

}

func (r *dnsRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured provider", "The Spaceship provider was not configured. Please ensure the provider block is present.")
		return
	}

	var state dnsRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, recordType, name, signature, ok := parseRecordID(state.ID.ValueString())
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid resource ID",
			fmt.Sprintf("Expected format domain/TYPE/name/<signature>, got %q", state.ID.ValueString()),
		)
		return
	}

	record, err := r.client.FindDNSRecord(ctx, domain, recordType, name, signature)
	if errors.Is(err, client.ErrRecordNotFound) {
		// Record no longer exists in the custom group — drop it from state so
		// Terraform will plan a recreate on the next apply.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to find DNS record for %s: %s", domain, err))
		return
	}

	state.Domain = types.StringValue(domain)
	hydrateRecordModel(&state.dnsRecordModel, record)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured provider", "The Spaceship provider was not configured. Please ensure the provider block is present.")
		return
	}

	// Schema marks every non-ttl attribute RequiresReplace, so Update only
	// runs when ttl is the only field that changed. Re-fetch the record by
	// identity to recover its full data, mutate the ttl, and re-upsert.
	var plan, state dnsRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, recordType, name, signature, ok := parseRecordID(state.ID.ValueString())
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid resource ID",
			fmt.Sprintf("Expected format domain/TYPE/name/<signature>, got %q", state.ID.ValueString()),
		)
		return
	}

	record, err := r.client.FindDNSRecord(ctx, domain, recordType, name, signature)
	if errors.Is(err, client.ErrRecordNotFound) {
		resp.Diagnostics.AddError(
			"DNS record not found",
			"The DNS record being updated no longer exists in the custom group. Run `terraform refresh` to reconcile state.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to look up DNS record for %s: %s", domain, err))
		return
	}

	record.TTL = int(plan.TTL.ValueInt64())

	if err := r.client.CreateDNSRecord(ctx, domain, record); err != nil {
		resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to update DNS record TTL: %s", err))
		return
	}

	// Data signature unchanged, so the composite ID remains stable.
	plan.ID = state.ID
	hydrateRecordModel(&plan.dnsRecordModel, record)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import string is the full composite ID (domain/TYPE/name/<signature>).
	// Passthrough writes it to state.ID; Terraform then calls Read which parses
	// the ID and hydrates the rest of the attributes.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// parseRecordID is the inverse of recordID. It returns the components needed
// to locate a record via the Spaceship API. The signature segment is the last
// part of the ID and may itself contain "/" (e.g. inside a TXT value), so the
// split uses a limit of 4 to keep the data segment intact.
func parseRecordID(id string) (domain, recordType, name, signature string, ok bool) {
	parts := strings.SplitN(id, "/", 4)
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", "", false
	}
	return parts[0], parts[1], parts[2], parts[3], true
}

// ConfigValidators wires the per-type record validators from
// recordTypeObjectValidators() into this singular resource. Each Object
// validator (designed for the nested-block usage in spaceship_dns_records)
// is wrapped by singularRecordValidator, which adapts the flat resource
// config into a synthetic Object value the validator can consume. This is
// the "config-validator adapter" the comment on recordTypeObjectValidators
// promises — both resources now share the same per-type validation logic.
func (r *dnsRecordResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	objectValidators := recordTypeObjectValidators()
	adapters := make([]resource.ConfigValidator, 0, len(objectValidators))
	for _, v := range objectValidators {
		adapters = append(adapters, singularRecordValidator{inner: v})
	}
	return adapters
}

// singularRecordValidator adapts a validator.Object — designed to run against
// each element of a nested record list — to operate on the flat singular
// dns_record resource. It reads the resource config into a dnsRecordModel,
// synthesizes a types.Object whose attributes match what the inner validator
// expects, and dispatches to the inner ValidateObject. Diagnostic paths use
// path.Empty() as the base, so an error on `exchange` produces a path of
// `exchange` (not `records[0].exchange`).
type singularRecordValidator struct {
	inner validator.Object
}

func (a singularRecordValidator) Description(ctx context.Context) string {
	return a.inner.Description(ctx)
}

func (a singularRecordValidator) MarkdownDescription(ctx context.Context) string {
	return a.inner.MarkdownDescription(ctx)
}

func (a singularRecordValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model dnsRecordResourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	obj, objDiags := types.ObjectValueFrom(ctx, dnsRecordObjectType.AttrTypes, model.dnsRecordModel)
	resp.Diagnostics.Append(objDiags...)
	if objDiags.HasError() {
		return
	}

	objReq := validator.ObjectRequest{
		Path:        path.Empty(),
		Config:      req.Config,
		ConfigValue: obj,
	}
	objResp := &validator.ObjectResponse{}
	a.inner.ValidateObject(ctx, objReq, objResp)
	resp.Diagnostics.Append(objResp.Diagnostics...)
}
