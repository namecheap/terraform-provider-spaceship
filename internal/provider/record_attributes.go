package provider

import (
	"terraform-provider-spaceship/internal/provider/records"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// record attributes retunn the schema attributes that describe a single DNS record.
// shared by the list resource and the single-record resource.
// merged with id + domain at the resource root
func recordAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"type": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "DNS record type(A, AAAA, ALIAS, CAA, CNAME, HTTPS, MX, NS, PTR, SRV, SVCB, TLSA, TXT).",
			Validators: []validator.String{
				stringvalidator.OneOf("A", "AAAA", "ALIAS", "CAA", "CNAME", "HTTPS", "MX", "NS", "PTR", "SRV", "SVCB", "TLSA", "TXT"),
			},
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Record host. Use `@` for the zone apex.",
			Validators: []validator.String{
				stringvalidator.LengthBetween(1, 253),
				records.NameValidator(),
			},
		},
		"ttl": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Record TTL in seconds. Defaults to 3600 if omitted.",
			Default:             int64default.StaticInt64(defaultRecordTTL),
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
			MarkdownDescription: "Canonical domain name for ALIAS records. Implements CNAME-like behavior for the zone apex where CNAME is not allowed.",
		},
		"cname": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Canonical name for CNAME records.",
		},
		"flag": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "Flag for CAA records (0 or 128).",
		},
		"tag": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Tag for CAA records (e.g. `issue`)",
		},
		"value": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Generic value field used by several record types (CAA, TXT).",
		},
		"port": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Port for HTTPS, SVCB and TLSA records(accepts `*` or `_NNNN`).",
		},
		"scheme": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Scheme for HTTPS/SVCB/TLSA records (for example `_https`, `_tcp`)",
		},
		"svc_priority": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "Service priority for HTTPS/SVCB records (0-65535).",
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
		},
		"priority": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "Priority for SRV records (0-65535).",
		},
		"weight": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "Weight for SRV records (0-65535).",
		},
		"port_number": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "Port for SRV records (1-65535).",
		},
		"target": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Target host for SRV records.",
		},
		"usage": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "Usage value for TLSA records (0-255).",
		},
		"selector": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "Selector value for TLSA records (0-255).",
		},
		"matching": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "Matching type for TLSA records (0-255).",
		},
		"association_data": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Association data (hex) for TLSA records.",
		},
	}
}

// recordTypeObjectValidators returns the per-type record validators that run
// against a single record object. Shared between the list resource (as
// nested-object validators) and the single resource (via a config-validator
// adapter that synthesizes an Object from the flat resource attributes).
func recordTypeObjectValidators() []validator.Object {
	return []validator.Object{
		records.IrrelevantFieldsValidator(),
		records.AValidator(),
		records.AAAAValidator(),
		records.ALIASValidator(),
		records.CAAValidator(),
		records.HTTPSValidator(),
		records.CNAMEValidator(),
		records.MXValidator(),
		records.NSValidator(),
		records.PTRValidator(),
		records.SRVValidator(),
		records.SVCBValidator(),
		records.TLSAValidator(),
		records.TXTValidator(),
	}
}
