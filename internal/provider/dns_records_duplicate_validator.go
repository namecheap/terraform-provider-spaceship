package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/namecheap/go-spaceship-sdk/client"
)

// duplicateRecordsValidator rejects records lists containing entries the
// Spaceship API would treat as the same record. The API silently dedups such
// entries on write, so the post-apply read returns fewer records than planned
// and Terraform fails with "inconsistent result after apply". Identity follows
// client.RecordKey: type + name + data case-insensitive, except TXT values.
type duplicateRecordsValidator struct{}

var _ validator.List = duplicateRecordsValidator{}

func (v duplicateRecordsValidator) Description(_ context.Context) string {
	return "records must be unique by type + name + data (case-insensitive, except TXT values)"
}

func (v duplicateRecordsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v duplicateRecordsValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	firstIndexByKey := make(map[string]int)

	for idx, element := range req.ConfigValue.Elements() {
		object, ok := element.(types.Object)
		if !ok || object.IsNull() || object.IsUnknown() {
			continue
		}
		// An unknown field ("known after apply") makes the record's API identity
		// uncomputable at plan time; comparing partial keys would flag false
		// duplicates, so skip the element instead.
		if recordHasUnknownAttribute(object) {
			continue
		}

		var model dnsRecordModel
		if diags := object.As(ctx, &model, basetypes.ObjectAsOptions{}); diags.HasError() {
			continue
		}

		// Conversion diagnostics (missing type/name/data fields) are owned by the
		// schema and per-type object validators; here they only mean the record
		// has no computable identity yet, so drop them and skip the element.
		record, diags := modelToDNSRecord(model, req.Path.AtListIndex(idx))
		if diags.HasError() {
			continue
		}

		key := client.RecordKey(record)
		firstIdx, seen := firstIndexByKey[key]
		if !seen {
			firstIndexByKey[key] = idx
			continue
		}

		resp.Diagnostics.AddAttributeError(
			req.Path.AtListIndex(idx),
			"Duplicate DNS Record",
			fmt.Sprintf(
				"This entry duplicates records[%d]. The Spaceship API matches records by type + name + data case-insensitively (TXT values are case-sensitive) and stores only one copy, so the apply can never converge. Remove one of the entries.",
				firstIdx,
			),
		)
	}
}

func recordHasUnknownAttribute(object types.Object) bool {
	for _, value := range object.Attributes() {
		if value.IsUnknown() {
			return true
		}
	}
	return false
}
