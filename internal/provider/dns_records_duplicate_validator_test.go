package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Duplicate detection must mirror the API's record identity (client.RecordKey):
// type + name + data case-insensitive, except TXT values which are case-sensitive.
// TTL is not part of the identity.
func TestDuplicateRecordsValidator(t *testing.T) {
	aRecord := func(name, address string) dnsRecordModel {
		return dnsRecordModel{
			Type:    types.StringValue("A"),
			Name:    types.StringValue(name),
			Address: types.StringValue(address),
		}
	}
	txtRecord := func(name, value string) dnsRecordModel {
		return dnsRecordModel{
			Type:  types.StringValue("TXT"),
			Name:  types.StringValue(name),
			Value: types.StringValue(value),
		}
	}

	cases := []struct {
		name       string
		records    []dnsRecordModel
		wantErrors int
	}{
		{
			name:       "exact duplicates",
			records:    []dnsRecordModel{aRecord("www", "192.0.2.1"), aRecord("www", "192.0.2.1")},
			wantErrors: 1,
		},
		{
			name:       "name case-folded duplicates",
			records:    []dnsRecordModel{aRecord("WWW", "192.0.2.1"), aRecord("www", "192.0.2.1")},
			wantErrors: 1,
		},
		{
			name: "data case-folded duplicates",
			records: []dnsRecordModel{
				{Type: types.StringValue("CNAME"), Name: types.StringValue("app"), CName: types.StringValue("Target.Example.COM")},
				{Type: types.StringValue("CNAME"), Name: types.StringValue("app"), CName: types.StringValue("target.example.com")},
			},
			wantErrors: 1,
		},
		{
			name:       "duplicates differing only in TTL",
			records:    []dnsRecordModel{aRecord("www", "192.0.2.1"), withTTL(aRecord("www", "192.0.2.1"), 300)},
			wantErrors: 1,
		},
		{
			name:       "TXT values differing only in case are distinct",
			records:    []dnsRecordModel{txtRecord("@", "Hello"), txtRecord("@", "hello")},
			wantErrors: 0,
		},
		{
			name:       "TXT identical values are duplicates",
			records:    []dnsRecordModel{txtRecord("@", "hello"), txtRecord("@", "hello")},
			wantErrors: 1,
		},
		{
			name:       "same host different data is distinct",
			records:    []dnsRecordModel{aRecord("www", "192.0.2.1"), aRecord("www", "192.0.2.2")},
			wantErrors: 0,
		},
		{
			name:       "triplicate reports each later occurrence",
			records:    []dnsRecordModel{aRecord("www", "192.0.2.1"), aRecord("www", "192.0.2.1"), aRecord("www", "192.0.2.1")},
			wantErrors: 2,
		},
		{
			name: "unknown data fields are skipped, no false positive",
			records: []dnsRecordModel{
				{Type: types.StringValue("A"), Name: types.StringValue("www"), Address: types.StringUnknown()},
				{Type: types.StringValue("A"), Name: types.StringValue("www"), Address: types.StringUnknown()},
			},
			wantErrors: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &validator.ListResponse{}
			duplicateRecordsValidator{}.ValidateList(context.Background(), validator.ListRequest{
				Path:        path.Root("records"),
				ConfigValue: buildRecordList(t, tc.records...),
			}, resp)

			if got := resp.Diagnostics.ErrorsCount(); got != tc.wantErrors {
				t.Fatalf("expected %d errors, got %d: %s", tc.wantErrors, got, resp.Diagnostics)
			}
		})
	}
}

// The error must land on the later duplicate's list index and point back at the
// first occurrence so the user knows which line to keep.
func TestDuplicateRecordsValidator_ErrorPlacement(t *testing.T) {
	records := []dnsRecordModel{
		{Type: types.StringValue("A"), Name: types.StringValue("keep"), Address: types.StringValue("192.0.2.10")},
		{Type: types.StringValue("TXT"), Name: types.StringValue("@"), Value: types.StringValue("v=spf1 -all")},
		{Type: types.StringValue("TXT"), Name: types.StringValue("@"), Value: types.StringValue("v=spf1 -all")},
	}

	resp := &validator.ListResponse{}
	duplicateRecordsValidator{}.ValidateList(context.Background(), validator.ListRequest{
		Path:        path.Root("records"),
		ConfigValue: buildRecordList(t, records...),
	}, resp)

	if got := resp.Diagnostics.ErrorsCount(); got != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %s", got, resp.Diagnostics)
	}

	diagnostic := resp.Diagnostics.Errors()[0]
	wantPath := path.Root("records").AtListIndex(2)
	if withPath, ok := diagnostic.(interface{ Path() path.Path }); !ok || !withPath.Path().Equal(wantPath) {
		t.Errorf("expected error at %s, got %v", wantPath, diagnostic)
	}
	if !strings.Contains(diagnostic.Detail(), "records[1]") {
		t.Errorf("expected detail to reference first occurrence records[1], got: %s", diagnostic.Detail())
	}
}

// Null and unknown lists are not validatable and must not panic or error.
func TestDuplicateRecordsValidator_NullUnknownList(t *testing.T) {
	for name, value := range map[string]types.List{
		"null":    types.ListNull(dnsRecordObjectType),
		"unknown": types.ListUnknown(dnsRecordObjectType),
	} {
		t.Run(name, func(t *testing.T) {
			resp := &validator.ListResponse{}
			duplicateRecordsValidator{}.ValidateList(context.Background(), validator.ListRequest{
				Path:        path.Root("records"),
				ConfigValue: value,
			}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("expected no errors, got: %s", resp.Diagnostics)
			}
		})
	}
}

func withTTL(model dnsRecordModel, ttl int64) dnsRecordModel {
	model.TTL = types.Int64Value(ttl)
	return model
}
