package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-spaceship/internal/client"
)

// defaultRecordTTL is the TTL applied when a record omits one. It is the single
// source of truth: the schema Default (recordAttributes) and the conversion
// fallback in modelToDNSRecord both reference it.
const defaultRecordTTL int64 = 3600

// dnsRecordModel is the Terraform model for a single DNS record, shared by the
// list resource (as an element of its records list) and the singular dns_record
// resource (embedded in its model). Field descriptions mirror recordAttributes()
// in record_attributes.go, which is the canonical schema-level source.
type dnsRecordModel struct {
	// Record type (A, AAAA, ALIAS, CAA, CNAME, HTTPS, MX, NS, PTR, SRV, SVCB, TLSA, TXT).
	Type types.String `tfsdk:"type"`
	// Record host. "@" denotes the zone apex.
	Name types.String `tfsdk:"name"`
	// Record TTL in seconds. Defaults to 3600 when omitted.
	TTL types.Int64 `tfsdk:"ttl"`
	// IPv4 or IPv6 address for A and AAAA records.
	Address types.String `tfsdk:"address"`
	// Canonical domain name for ALIAS records (apex-safe CNAME-like behavior).
	AliasName types.String `tfsdk:"alias_name"`
	// Canonical name for CNAME records.
	CName types.String `tfsdk:"cname"`
	// Flag for CAA records (0 or 128).
	Flag types.Int64 `tfsdk:"flag"`
	// Tag for CAA records (e.g. "issue").
	Tag types.String `tfsdk:"tag"`
	// Generic value field used by CAA and TXT records.
	Value types.String `tfsdk:"value"`
	// String-form port for HTTPS/SVCB/TLSA records (accepts "*" or "_NNNN").
	Port types.String `tfsdk:"port"`
	// Scheme for HTTPS/SVCB/TLSA records (e.g. "_https", "_tcp").
	Scheme types.String `tfsdk:"scheme"`
	// Service priority for HTTPS/SVCB records (0-65535).
	SvcPriority types.Int64 `tfsdk:"svc_priority"`
	// Target name for HTTPS/SVCB records.
	TargetName types.String `tfsdk:"target_name"`
	// SvcParams string for HTTPS/SVCB records.
	SvcParams types.String `tfsdk:"svc_params"`
	// Mail exchange host for MX records.
	Exchange types.String `tfsdk:"exchange"`
	// Preference value for MX records (0-65535).
	Preference types.Int64 `tfsdk:"preference"`
	// Nameserver host for NS records.
	Nameserver types.String `tfsdk:"nameserver"`
	// Pointer target for PTR records.
	Pointer types.String `tfsdk:"pointer"`
	// Service label for SRV records (e.g. "_sip").
	Service types.String `tfsdk:"service"`
	// Protocol label for SRV/TLSA records (e.g. "_tcp").
	Protocol types.String `tfsdk:"protocol"`
	// Priority for SRV records (0-65535).
	Priority types.Int64 `tfsdk:"priority"`
	// Weight for SRV records (0-65535).
	Weight types.Int64 `tfsdk:"weight"`
	// Integer-form port for SRV records (1-65535).
	PortNumber types.Int64 `tfsdk:"port_number"`
	// Target host for SRV records.
	Target types.String `tfsdk:"target"`
	// Usage value for TLSA records (0-255).
	Usage types.Int64 `tfsdk:"usage"`
	// Selector value for TLSA records (0-255).
	Selector types.Int64 `tfsdk:"selector"`
	// Matching type for TLSA records (0-255).
	Matching types.Int64 `tfsdk:"matching"`
	// Association data (hex) for TLSA records.
	AssociationData types.String `tfsdk:"association_data"`
}

// dnsRecordObjectType is the runtime attr.Type twin of dnsRecordModel. The
// Plugin Framework needs an explicit attribute-type map to build the
// types.List / types.Object values the list resource works with; keep it in
// sync with dnsRecordModel's tfsdk tags.
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

// hydrateRecordModel copies every field of an API record into the Terraform
// model using the empty-string-as-null convention. Single source of truth for
// record→model conversion; called by both flattenDNSRecords (list resource)
// and the singular dns_record resource's Read.
func hydrateRecordModel(model *dnsRecordModel, record client.DNSRecord) {
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

	model.Type = types.StringValue(record.Type)
	model.Name = types.StringValue(record.Name)

	if record.TTL > 0 {
		model.TTL = types.Int64Value(int64(record.TTL))
	} else {
		model.TTL = types.Int64Null()
	}

	model.Address = stringOrNull(record.Address)
	model.AliasName = stringOrNull(record.AliasName)
	model.CName = stringOrNull(record.CName)
	model.Flag = intPointerOrNull(record.Flag)
	model.Tag = stringOrNull(record.Tag)
	model.Value = stringOrNull(record.Value)
	model.Port = types.StringNull()
	model.PortNumber = types.Int64Null()
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
}

// modelToDNSRecord converts a Terraform model into a client.DNSRecord, applying
// per-type field requirements. Diagnostics are emitted under attrPath — pass
// listPath.AtListIndex(i) when converting an element of a list resource, or
// path.Empty() when converting a top-level resource model. Records with
// validation errors come back with a populated diag.Diagnostics; the caller
// should skip the record when diags.HasError() is true.
func modelToDNSRecord(model dnsRecordModel, attrPath path.Path) (client.DNSRecord, diag.Diagnostics) {
	var diags diag.Diagnostics

	recordType := strings.ToUpper(strings.TrimSpace(model.Type.ValueString()))
	if recordType == "" {
		diags.AddAttributeError(attrPath.AtName("type"), "Missing record type", "Each DNS record must specify a type (e.g. A, MX, TXT).")
		return client.DNSRecord{}, diags
	}

	name := strings.TrimSpace(model.Name.ValueString())
	if name == "" {
		diags.AddAttributeError(attrPath.AtName("name"), "Missing record name", "Each DNS record must specify a name (use '@' for the apex).")
		return client.DNSRecord{}, diags
	}
	// TTL is Optional+Computed with a schema Default (see recordAttributes), so a
	// planned model always carries a value. This fallback only guards callers that
	// build the model without going through plan (e.g. unit tests).
	ttl := defaultRecordTTL
	if !model.TTL.IsNull() && !model.TTL.IsUnknown() {
		ttl = model.TTL.ValueInt64()
	}

	record := client.DNSRecord{
		Type: recordType,
		Name: name,
		// int64 (Terraform types.Int64) -> int (client.DNSRecord.TTL).
		TTL: int(ttl),
	}

	getString := func(value types.String) (string, bool) {
		if value.IsNull() || value.IsUnknown() {
			return "", false
		}
		return value.ValueString(), true
	}

	requireString := func(value types.String, attrName, description string) (string, bool) {
		if value.IsUnknown() {
			return "", false
		}
		if value.IsNull() || strings.TrimSpace(value.ValueString()) == "" {
			diags.AddAttributeError(attrPath.AtName(attrName), fmt.Sprintf("Missing %s", attrName), description)
			return "", false
		}
		return value.ValueString(), true
	}

	requireInt := func(value types.Int64, attrName, description string) (int, bool) {
		if value.IsUnknown() {
			return 0, false
		}
		if value.IsNull() {
			diags.AddAttributeError(attrPath.AtName(attrName), fmt.Sprintf("Missing %s", attrName), description)
			return 0, false
		}
		return int(value.ValueInt64()), true
	}

	switch recordType {
	case "A", "AAAA":
		if addr, ok := requireString(model.Address, "address", "Records of this type require the `address` attribute."); ok {
			record.Address = addr
		}

	case "ALIAS":
		if alias, ok := requireString(model.AliasName, "alias_name", "ALIAS records require the `alias_name` attribute."); ok {
			record.AliasName = alias
		}

	case "CAA":
		if flag, ok := requireInt(model.Flag, "flag", "CAA records require the `flag` attribute (0 or 128)."); ok {
			record.Flag = &flag
		}
		if tag, ok := requireString(model.Tag, "tag", "CAA records require the `tag` attribute (e.g. `issue`)."); ok {
			record.Tag = tag
		}
		if value, ok := requireString(model.Value, "value", "CAA records require the `value` attribute."); ok {
			record.Value = value
		}

	case "CNAME":
		if cname, ok := requireString(model.CName, "cname", "CNAME records require the `cname` attribute."); ok {
			record.CName = cname
		}

	case "HTTPS":
		if pri, ok := requireInt(model.SvcPriority, "svc_priority", "HTTPS records require the `svc_priority` attribute."); ok {
			record.SvcPriority = &pri
		}
		if target, ok := requireString(model.TargetName, "target_name", "HTTPS records require the `target_name` attribute."); ok {
			record.TargetName = target
		}
		if params, ok := getString(model.SvcParams); ok {
			record.SvcParams = params
		}
		if port, ok := getString(model.Port); ok {
			record.Port = client.NewStringPortValue(port)
			if _, hasScheme := getString(model.Scheme); !hasScheme {
				diags.AddAttributeError(attrPath.AtName("scheme"), "Missing scheme", "HTTPS records that specify `port` must also set `scheme` (usually `_https`).")
			}
		}
		if scheme, ok := getString(model.Scheme); ok {
			record.Scheme = scheme
		}

	case "MX":
		if exchange, ok := requireString(model.Exchange, "exchange", "MX records require the `exchange` attribute (mail server hostname)."); ok {
			record.Exchange = exchange
		}
		if pref, ok := requireInt(model.Preference, "preference", "MX records require the `preference` attribute (0-65535)."); ok {
			record.Preference = &pref
		}

	case "NS":
		if ns, ok := requireString(model.Nameserver, "nameserver", "NS records require the `nameserver` attribute."); ok {
			record.Nameserver = ns
		}

	case "PTR":
		if pointer, ok := requireString(model.Pointer, "pointer", "PTR records require the `pointer` attribute."); ok {
			record.Pointer = pointer
		}

	case "SRV":
		if service, ok := requireString(model.Service, "service", "SRV records require the `service` attribute (e.g. `_sip`)."); ok {
			record.Service = service
		}
		if protocol, ok := requireString(model.Protocol, "protocol", "SRV records require the `protocol` attribute (e.g. `_tcp`)."); ok {
			record.Protocol = protocol
		}
		if priority, ok := requireInt(model.Priority, "priority", "SRV records require the `priority` attribute (0-65535)."); ok {
			record.Priority = &priority
		}
		if weight, ok := requireInt(model.Weight, "weight", "SRV records require the `weight` attribute (0-65535)."); ok {
			record.Weight = &weight
		}
		if port, ok := requireInt(model.PortNumber, "port_number", "SRV records require the `port_number` attribute (1-65535)."); ok {
			record.Port = client.NewIntPortValue(port)
		}
		if target, ok := requireString(model.Target, "target", "SRV records require the `target` attribute."); ok {
			record.Target = target
		}

	case "SVCB":
		if pri, ok := requireInt(model.SvcPriority, "svc_priority", "SVCB records require the `svc_priority` attribute (0-65535)."); ok {
			record.SvcPriority = &pri
		}
		if target, ok := requireString(model.TargetName, "target_name", "SVCB records require the `target_name` attribute."); ok {
			record.TargetName = target
		}
		if params, ok := getString(model.SvcParams); ok {
			record.SvcParams = params
		}
		if port, ok := getString(model.Port); ok {
			record.Port = client.NewStringPortValue(port)
		}
		if scheme, ok := getString(model.Scheme); ok {
			record.Scheme = scheme
		}

	case "TLSA":
		if port, ok := requireString(model.Port, "port", "TLSA records require the `port` attribute (e.g. `_443`)."); ok {
			record.Port = client.NewStringPortValue(port)
		}
		if protocol, ok := requireString(model.Protocol, "protocol", "TLSA records require the `protocol` attribute (e.g. `_tcp`)."); ok {
			record.Protocol = protocol
		}
		if usage, ok := requireInt(model.Usage, "usage", "TLSA records require the `usage` attribute (0-255)."); ok {
			record.Usage = &usage
		}
		if selector, ok := requireInt(model.Selector, "selector", "TLSA records require the `selector` attribute (0-255)."); ok {
			record.Selector = &selector
		}
		if matching, ok := requireInt(model.Matching, "matching", "TLSA records require the `matching` attribute (0-255)."); ok {
			record.Matching = &matching
		}
		if assoc, ok := requireString(model.AssociationData, "association_data", "TLSA records require the `association_data` attribute (certificate associations)."); ok {
			record.AssociationData = assoc
		}
		if scheme, ok := getString(model.Scheme); ok {
			record.Scheme = scheme
		}

	case "TXT":
		if value, ok := requireString(model.Value, "value", "TXT records require the `value` attribute."); ok {
			record.Value = value
		}

	default:
		diags.AddAttributeError(attrPath.AtName("type"), "Unsupported record type", fmt.Sprintf("Type %q is not supported by the provider.", recordType))
	}

	return record, diags
}
