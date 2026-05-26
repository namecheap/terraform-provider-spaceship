package provider

import (
	"context"
	"fmt"

	"terraform-provider-spaceship/internal/client"

	"github.com/dlclark/regexp2"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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

	recordNamePattern = regexp2.MustCompile(`^(?!\.)(@|\*|([_*]\.)?(?:(?!-)(?=[^\.]*[^\W_])[\w-]{1,63}(?<!-)($|\.)){1,127}(?<!\.))$`, 0)
)

// TODO move this validator in separate file?
// validators
func recordNameValidator() validator.String {
	return &recordNameValidatorImpl{}
}

type recordNameValidatorImpl struct{}

func (v recordNameValidatorImpl) Description(ctx context.Context) string {
	return "must be a valid record name"
}
func (v recordNameValidatorImpl) MarkdownDescription(ctx context.Context) string {
	return "must be a valid record name"
}
func (v recordNameValidatorImpl) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()

	match, err := recordNamePattern.MatchString(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Record Name",
			fmt.Sprintf("Error validating record name: %s", err),
		)
		return
	}

	if !match {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Record Name",
			"must be a valid record name format",
		)
	}

}

//validators ends

func NewDNSRecordsResource() resource.Resource {
	return &dnsRecordsResource{}
}

type dnsRecordsResource struct {
	client *client.Client
}

type dnsRecordsResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Domain  types.String `tfsdk:"domain"`
	Force   types.Bool   `tfsdk:"force"`
	Records types.List   `tfsdk:"records"`
}

type dnsRecordModel struct {
	//Type of the record
	Type types.String `tfsdk:"type"`
	// Name of the record
	Name types.String `tfsdk:"name"`
	TTL  types.Int64  `tfsdk:"ttl"`
	// Address
	Address types.String `tfsdk:"address"`
	// Alias name for alias record
	AliasName types.String `tfsdk:"alias_name"`
	//TODO
	//add descriptions for other records
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
		MarkdownDescription: "Manages custom DNS records for a Spaceship-managed domain. Only records in the `custom` DNS group are managed — records owned by Spaceship features (e.g. URL redirect, personal nameservers) are left untouched. On each apply, the provider computes a diff and only deletes removed records and upserts new or changed ones.\n\n> **Caution:** This resource takes ownership of the *entire* custom DNS group for the domain — any record present in the live zone but absent from the `records` list will be deleted on the next apply. Do not mix this with `spaceship_dns_record` (singular) for the same domain: records created by the singular resource will be silently destroyed when this resource next reconciles. Pick one resource per domain.",
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
				MarkdownDescription: "Deprecated: this attribute has no effect. The provider always applies DNS updates with force enabled.",
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Bool{
					deprecatedBoolValidator("The \"force\" attribute is deprecated and has no effect. The provider always applies DNS updates with force enabled. This attribute will be removed or reworked in a future version."),
				},
			},
			"records": schema.ListNestedAttribute{
				MarkdownDescription: "DNS records that should be configured for the domain. The provider diffs this list against existing custom records — only removed records are deleted and new or changed records are upserted. Records in other DNS groups (product, personalNS) are not affected.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Validators: recordTypeObjectValidators(),
					Attributes: recordAttributes(),
				},
			},
		},
	}
}

func (r *dnsRecordsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			resp.Diagnostics.AddError("Spaceship API error", fmt.Sprintf("Failed to apply DNS records: %s", err))
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
		if client.IsNotFoundError(err) {
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

	force := boolOrDefault(state.Force, true)
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

func expandDNSRecords(ctx context.Context, list types.List, listPath path.Path) ([]client.DNSRecord, diag.Diagnostics) {
	var diags diag.Diagnostics

	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	var models []dnsRecordModel
	listDiags := list.ElementsAs(ctx, &models, false)
	diags.Append(listDiags...)
	if diags.HasError() {
		return nil, diags
	}

	records := make([]client.DNSRecord, 0, len(models))
	for idx, item := range models {
		recordPath := listPath.AtListIndex(idx)
		record, recordDiags := modelToDNSRecord(item, recordPath)
		diags.Append(recordDiags...)
		if recordDiags.HasError() {
			continue
		}
		records = append(records, record)
	}
	return records, diags
}

func flattenDNSRecords(ctx context.Context, records []client.DNSRecord) (types.List, diag.Diagnostics) {
	elements := make([]dnsRecordModel, 0, len(records))
	for _, record := range records {
		var model dnsRecordModel
		hydrateRecordModel(&model, record)
		elements = append(elements, model)
	}

	return types.ListValueFrom(ctx, dnsRecordObjectType, elements)
}

func diffDNSRecords(existing, desired []client.DNSRecord) (toDelete, toUpsert []client.DNSRecord) {
	desiredMap := make(map[string]client.DNSRecord, len(desired))
	for _, record := range desired {
		desiredMap[client.RecordKey(record)] = record
	}

	existingMap := make(map[string]client.DNSRecord, len(existing))
	for _, record := range existing {
		existingMap[client.RecordKey(record)] = record
		if _, ok := desiredMap[client.RecordKey(record)]; !ok {
			toDelete = append(toDelete, record)
		}
	}

	seen := make(map[string]struct{})

	for _, record := range desired {
		key := client.RecordKey(record)
		existingRecord, ok := existingMap[key]
		if ok && existingRecord.TTL == record.TTL && client.RecordValueSignature(existingRecord) == client.RecordValueSignature(record) {
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

func orderDNSRecordsLike(reference, records []client.DNSRecord) []client.DNSRecord {
	if len(records) <= 1 || len(reference) == 0 {
		return records
	}

	type keyedRecord struct {
		key    string
		record client.DNSRecord
		used   bool
	}

	keyed := make([]keyedRecord, len(records))
	for i, record := range records {
		keyed[i] = keyedRecord{
			key:    client.RecordKey(record),
			record: record,
		}
	}

	ordered := make([]client.DNSRecord, 0, len(records))

	for _, ref := range reference {
		key := client.RecordKey(ref)
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

func boolOrDefault(value types.Bool, fallback bool) bool {
	if value.IsNull() || value.IsUnknown() {
		return fallback
	}
	return value.ValueBool()
}

// TODO
// move to some other file

// deprecatedBoolWarning is a validator that emits a warning when a deprecated
// bool attribute is explicitly configured.
type deprecatedBoolWarning struct {
	message string
}

func deprecatedBoolValidator(message string) validator.Bool {
	return deprecatedBoolWarning{message: message}
}

func (v deprecatedBoolWarning) Description(_ context.Context) string {
	return v.message
}

func (v deprecatedBoolWarning) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v deprecatedBoolWarning) ValidateBool(_ context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	resp.Diagnostics.Append(diag.NewAttributeWarningDiagnostic(
		req.Path,
		"Deprecated Attribute",
		v.message,
	))
}
