package records

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// recordFieldsByType maps each supported record type to the type-specific
// attribute names that may legally appear on a record of that type. The
// universal attributes (type, name, ttl) are always allowed and are not
// listed here.
//
// This is the single source of truth for "which fields belong to which
// type." It catches configs that set an attribute from another type
// (e.g. `address` on a TLSA record) — without rejection, those values are
// silently dropped on write but reappear as state drift on the next read,
// triggering a no-op recreate against the upsert-idempotent API.
var recordFieldsByType = map[string]map[string]struct{}{
	"A":     fieldSet("address"),
	"AAAA":  fieldSet("address"),
	"ALIAS": fieldSet("alias_name"),
	"CAA":   fieldSet("flag", "tag", "value"),
	"CNAME": fieldSet("cname"),
	"HTTPS": fieldSet("svc_priority", "target_name", "svc_params", "port", "scheme"),
	"MX":    fieldSet("exchange", "preference"),
	"NS":    fieldSet("nameserver"),
	"PTR":   fieldSet("pointer"),
	"SRV":   fieldSet("service", "protocol", "priority", "weight", "port_number", "target"),
	"SVCB":  fieldSet("svc_priority", "target_name", "svc_params", "port", "scheme"),
	"TLSA":  fieldSet("port", "protocol", "usage", "selector", "matching", "association_data"),
	"TXT":   fieldSet("value"),
}

func fieldSet(names ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}
	return m
}

var universalRecordFields = map[string]struct{}{
	"type": {}, "name": {}, "ttl": {},
}

type irrelevantFieldsValidator struct{}

var _ validator.Object = &irrelevantFieldsValidator{}

// IrrelevantFieldsValidator rejects record attributes that aren't applicable
// to the declared record type. Pairs with the per-type validators (which
// check required fields and value shape) by catching the inverse: fields
// that shouldn't be present at all.
func IrrelevantFieldsValidator() validator.Object {
	return &irrelevantFieldsValidator{}
}

func (v *irrelevantFieldsValidator) Description(_ context.Context) string {
	return "rejects record attributes that are not applicable to the declared record type"
}

func (v *irrelevantFieldsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *irrelevantFieldsValidator) ValidateObject(_ context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()
	typeAttr, ok := attrs["type"].(types.String)
	if !ok || typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}

	recordType := strings.ToUpper(typeAttr.ValueString())
	allowed, known := recordFieldsByType[recordType]
	if !known {
		// Unknown type — leave that to the schema-level OneOf validator.
		return
	}

	for name, value := range attrs {
		if _, isUniversal := universalRecordFields[name]; isUniversal {
			continue
		}
		if _, isAllowed := allowed[name]; isAllowed {
			continue
		}
		if !isAttrSet(value) {
			continue
		}
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName(name),
			"Invalid Field for Record Type",
			fmt.Sprintf("The %q field is not applicable to %s records. Allowed type-specific fields for %s: %s.",
				name, recordType, recordType, sortedFieldList(allowed)),
		)
	}
}

func isAttrSet(value attr.Value) bool {
	return value != nil && !value.IsNull() && !value.IsUnknown()
}

func sortedFieldList(set map[string]struct{}) string {
	names := make([]string, 0, len(set))
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
