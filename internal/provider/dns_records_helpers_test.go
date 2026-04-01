package provider

import (
	"context"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBoolOrDefault_Null(t *testing.T) {
	if boolOrDefault(types.BoolNull(), true) != true {
		t.Error("expected true for null with fallback true")
	}
	if boolOrDefault(types.BoolNull(), false) != false {
		t.Error("expected false for null with fallback false")
	}
}

func TestBoolOrDefault_Unknown(t *testing.T) {
	if boolOrDefault(types.BoolUnknown(), true) != true {
		t.Error("expected true for unknown with fallback true")
	}
}

func TestBoolOrDefault_Set(t *testing.T) {
	if boolOrDefault(types.BoolValue(false), true) != false {
		t.Error("expected false when explicitly set to false")
	}
	if boolOrDefault(types.BoolValue(true), false) != true {
		t.Error("expected true when explicitly set to true")
	}
}

func TestRecordValueSignature_AllTypes(t *testing.T) {
	tests := []struct {
		name   string
		record client.DNSRecord
	}{
		{"A", client.DNSRecord{Type: "A", Name: "@", Address: "1.2.3.4"}},
		{"AAAA", client.DNSRecord{Type: "AAAA", Name: "@", Address: "2001:db8::1"}},
		{"ALIAS", client.DNSRecord{Type: "ALIAS", Name: "@", AliasName: "other.com"}},
		{"CAA", client.DNSRecord{Type: "CAA", Name: "@", Flag: intPtr(0), Tag: "issue", Value: "letsencrypt.org"}},
		{"CNAME", client.DNSRecord{Type: "CNAME", Name: "www", CName: "example.com"}},
		{"HTTPS", client.DNSRecord{Type: "HTTPS", Name: "@", SvcPriority: intPtr(1), TargetName: "target.com", SvcParams: "alpn=h2", Port: client.NewStringPortValue("_443"), Scheme: "_https"}},
		{"MX", client.DNSRecord{Type: "MX", Name: "@", Exchange: "mail.example.com", Preference: intPtr(10)}},
		{"NS", client.DNSRecord{Type: "NS", Name: "@", Nameserver: "ns1.example.com"}},
		{"PTR", client.DNSRecord{Type: "PTR", Name: "1", Pointer: "host.example.com"}},
		{"SRV", client.DNSRecord{Type: "SRV", Name: "_sip._tcp", Service: "_sip", Protocol: "_tcp", Priority: intPtr(5), Weight: intPtr(10)}},
		{"SVCB", client.DNSRecord{Type: "SVCB", Name: "@", SvcPriority: intPtr(1), TargetName: "svc.com", SvcParams: "alpn=h2", Port: client.NewStringPortValue("_853"), Scheme: "_dot"}},
		{"TXT", client.DNSRecord{Type: "TXT", Name: "@", Value: "v=spf1"}},
		{"unknown", client.DNSRecord{Type: "UNKNOWN", Name: "@", Address: "fallback"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sig := recordValueSignature(tc.record)
			if sig == "" && tc.name != "unknown" {
				t.Error("expected non-empty signature")
			}
			// signature should be deterministic
			if sig != recordValueSignature(tc.record) {
				t.Error("signature should be deterministic")
			}
		})
	}
}

func TestRecordValueSignature_CaseInsensitive(t *testing.T) {
	r1 := client.DNSRecord{Type: "A", Name: "@", Address: "1.2.3.4"}
	r2 := client.DNSRecord{Type: "a", Name: "@", Address: "1.2.3.4"}
	if recordValueSignature(r1) != recordValueSignature(r2) {
		t.Error("expected case-insensitive match for A records")
	}
}

func TestPortValueSignature(t *testing.T) {
	tests := []struct {
		name     string
		port     *client.PortValue
		expected string
	}{
		{"nil", nil, ""},
		{"int", client.NewIntPortValue(443), "443"},
		{"string", client.NewStringPortValue("_443"), "_443"},
		{"empty", &client.PortValue{}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := portValueSignature(tc.port)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestIntToString(t *testing.T) {
	if intToString(nil) != "" {
		t.Error("expected empty for nil")
	}
	v := 42
	if intToString(&v) != "42" {
		t.Errorf("expected 42, got %q", intToString(&v))
	}
}

func TestRecordKey(t *testing.T) {
	r := client.DNSRecord{Type: "A", Name: "test", Address: "1.2.3.4"}
	key := recordKey(r)
	if key == "" {
		t.Error("expected non-empty key")
	}
	// same record should produce same key
	if key != recordKey(r) {
		t.Error("expected deterministic key")
	}
	// type should be uppercased, name lowercased
	r2 := client.DNSRecord{Type: "a", Name: "TEST", Address: "1.2.3.4"}
	if key != recordKey(r2) {
		t.Error("expected case-insensitive key match")
	}
}

func TestDiffDNSRecords_NewRecords(t *testing.T) {
	existing := []client.DNSRecord{}
	desired := []client.DNSRecord{
		{Type: "A", Name: "new", Address: "1.2.3.4", TTL: 3600},
	}

	toDelete, toUpsert := diffDNSRecords(existing, desired)
	if len(toDelete) != 0 {
		t.Errorf("expected no deletions, got %d", len(toDelete))
	}
	if len(toUpsert) != 1 {
		t.Errorf("expected 1 upsert, got %d", len(toUpsert))
	}
}

func TestDiffDNSRecords_DuplicateDesired(t *testing.T) {
	existing := []client.DNSRecord{}
	desired := []client.DNSRecord{
		{Type: "A", Name: "dup", Address: "1.2.3.4", TTL: 3600},
		{Type: "A", Name: "dup", Address: "1.2.3.4", TTL: 3600},
	}

	_, toUpsert := diffDNSRecords(existing, desired)
	if len(toUpsert) != 1 {
		t.Errorf("expected deduplication to 1 upsert, got %d", len(toUpsert))
	}
}

func TestDiffDNSRecords_DoesNotDeleteFilteredRecords(t *testing.T) {
	// Simulates the scenario after GetDNSRecords filters out non-custom records.
	// The existing slice only contains custom records (product/personalNS already removed).
	// The diff should only delete custom records not in the desired set.
	existing := []client.DNSRecord{
		{Type: "A", Name: "@", Address: "1.2.3.4", TTL: 3600, Group: &client.RecordGroup{Type: "custom"}},
		{Type: "TXT", Name: "@", Value: "old-txt", TTL: 3600, Group: &client.RecordGroup{Type: "custom"}},
	}
	desired := []client.DNSRecord{
		{Type: "A", Name: "@", Address: "1.2.3.4", TTL: 3600},
	}

	toDelete, toUpsert := diffDNSRecords(existing, desired)
	if len(toDelete) != 1 {
		t.Errorf("expected 1 deletion (old TXT), got %d", len(toDelete))
	}
	if len(toDelete) > 0 && toDelete[0].Value != "old-txt" {
		t.Errorf("expected deleted record to be old TXT, got %+v", toDelete[0])
	}
	if len(toUpsert) != 0 {
		t.Errorf("expected 0 upserts (A record unchanged), got %d", len(toUpsert))
	}
}

func TestOrderDNSRecordsLike_EmptyReference(t *testing.T) {
	records := []client.DNSRecord{{Type: "A", Name: "@", Address: "1.2.3.4"}}
	result := orderDNSRecordsLike(nil, records)
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
}

func TestOrderDNSRecordsLike_SingleRecord(t *testing.T) {
	records := []client.DNSRecord{{Type: "A", Name: "@", Address: "1.2.3.4"}}
	ref := []client.DNSRecord{{Type: "A", Name: "@", Address: "1.2.3.4"}}
	result := orderDNSRecordsLike(ref, records)
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
}

func TestOrderDNSRecordsLike_NewRecordsAppended(t *testing.T) {
	reference := []client.DNSRecord{
		{Type: "A", Name: "@", Address: "1.1.1.1", TTL: 3600},
	}
	records := []client.DNSRecord{
		{Type: "TXT", Name: "@", Value: "new", TTL: 3600},
		{Type: "A", Name: "@", Address: "1.1.1.1", TTL: 3600},
	}

	ordered := orderDNSRecordsLike(reference, records)
	if ordered[0].Type != "A" {
		t.Errorf("expected A first, got %s", ordered[0].Type)
	}
	if ordered[1].Type != "TXT" {
		t.Errorf("expected TXT second, got %s", ordered[1].Type)
	}
}

func TestFlattenDNSRecords_RoundTrip(t *testing.T) {
	ctx := context.Background()
	records := []client.DNSRecord{
		{Type: "A", Name: "@", TTL: 3600, Address: "1.2.3.4"},
		{Type: "MX", Name: "@", TTL: 3600, Exchange: "mail.example.com", Preference: intPtr(10)},
		{Type: "CNAME", Name: "www", TTL: 600, CName: "example.com"},
		{Type: "TXT", Name: "@", TTL: 3600, Value: "v=spf1 include:example.com ~all"},
		{Type: "SRV", Name: "_sip._tcp", TTL: 3600, Service: "_sip", Protocol: "_tcp", Priority: intPtr(5), Weight: intPtr(10), Port: client.NewIntPortValue(5060), Target: "srv.example.com"},
		{Type: "HTTPS", Name: "@", TTL: 3600, SvcPriority: intPtr(1), TargetName: "target.com", SvcParams: "alpn=h2", Port: client.NewStringPortValue("_443"), Scheme: "_https"},
		{Type: "TLSA", Name: "_443._tcp", TTL: 3600, Port: client.NewStringPortValue("_443"), Protocol: "_tcp", Usage: intPtr(2), Selector: intPtr(1), Matching: intPtr(1), AssociationData: "AABB"},
		{Type: "CAA", Name: "@", TTL: 3600, Flag: intPtr(0), Tag: "issue", Value: "letsencrypt.org"},
		{Type: "NS", Name: "@", TTL: 3600, Nameserver: "ns1.example.com"},
		{Type: "PTR", Name: "1", TTL: 3600, Pointer: "host.example.com"},
		{Type: "AAAA", Name: "@", TTL: 3600, Address: "2001:db8::1"},
		{Type: "ALIAS", Name: "@", TTL: 3600, AliasName: "other.com"},
	}

	list, diags := flattenDNSRecords(ctx, records)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if list.IsNull() || list.IsUnknown() {
		t.Fatal("expected non-null list")
	}
	if len(list.Elements()) != len(records) {
		t.Errorf("expected %d elements, got %d", len(records), len(list.Elements()))
	}
}

func TestFlattenDNSRecords_Empty(t *testing.T) {
	ctx := context.Background()
	list, diags := flattenDNSRecords(ctx, nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if len(list.Elements()) != 0 {
		t.Errorf("expected 0 elements, got %d", len(list.Elements()))
	}
}

func TestFlattenDNSRecords_ZeroTTL(t *testing.T) {
	ctx := context.Background()
	records := []client.DNSRecord{
		{Type: "A", Name: "@", TTL: 0, Address: "1.2.3.4"},
	}
	list, diags := flattenDNSRecords(ctx, records)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	var models []dnsRecordModel
	diags = list.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !models[0].TTL.IsNull() {
		t.Error("expected null TTL for zero value")
	}
}

func TestExpandDNSRecords_EmptyType(t *testing.T) {
	ctx := context.Background()
	list := buildRecordList(t, dnsRecordModel{
		Type: types.StringValue(""),
		Name: types.StringValue("test"),
	})
	_, diags := expandDNSRecords(ctx, list, path.Root("records"))
	if !diags.HasError() {
		t.Error("expected error for empty type")
	}
}

func TestExpandDNSRecords_EmptyName(t *testing.T) {
	ctx := context.Background()
	list := buildRecordList(t, dnsRecordModel{
		Type: types.StringValue("A"),
		Name: types.StringValue(""),
	})
	_, diags := expandDNSRecords(ctx, list, path.Root("records"))
	if !diags.HasError() {
		t.Error("expected error for empty name")
	}
}

func TestExpandDNSRecords_UnsupportedType(t *testing.T) {
	ctx := context.Background()
	list := buildRecordList(t, dnsRecordModel{
		Type: types.StringValue("INVALID"),
		Name: types.StringValue("test"),
	})
	_, diags := expandDNSRecords(ctx, list, path.Root("records"))
	if !diags.HasError() {
		t.Error("expected error for unsupported type")
	}
}

func TestExpandDNSRecords_NullList(t *testing.T) {
	ctx := context.Background()
	nullList := types.ListNull(types.ObjectType{AttrTypes: testRecordAttrTypes})
	records, diags := expandDNSRecords(ctx, nullList, path.Root("records"))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if records != nil {
		t.Error("expected nil records for null list")
	}
}

func TestExpandDNSRecords_CustomTTL(t *testing.T) {
	ctx := context.Background()
	list := buildRecordList(t, dnsRecordModel{
		Type:    types.StringValue("A"),
		Name:    types.StringValue("test"),
		TTL:     types.Int64Value(600),
		Address: types.StringValue("1.2.3.4"),
	})
	records, diags := expandDNSRecords(ctx, list, path.Root("records"))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if records[0].TTL != 600 {
		t.Errorf("expected TTL 600, got %d", records[0].TTL)
	}
}

func TestExpandDNSRecords_MissingRequiredFields(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name  string
		model dnsRecordModel
	}{
		{"ALIAS missing alias_name", dnsRecordModel{Type: types.StringValue("ALIAS"), Name: types.StringValue("test")}},
		{"CNAME missing cname", dnsRecordModel{Type: types.StringValue("CNAME"), Name: types.StringValue("test")}},
		{"MX missing exchange", dnsRecordModel{Type: types.StringValue("MX"), Name: types.StringValue("test")}},
		{"NS missing nameserver", dnsRecordModel{Type: types.StringValue("NS"), Name: types.StringValue("test")}},
		{"PTR missing pointer", dnsRecordModel{Type: types.StringValue("PTR"), Name: types.StringValue("test")}},
		{"TXT missing value", dnsRecordModel{Type: types.StringValue("TXT"), Name: types.StringValue("test")}},
		{"CAA missing fields", dnsRecordModel{Type: types.StringValue("CAA"), Name: types.StringValue("test")}},
		{"HTTPS missing fields", dnsRecordModel{Type: types.StringValue("HTTPS"), Name: types.StringValue("test")}},
		{"SRV missing fields", dnsRecordModel{Type: types.StringValue("SRV"), Name: types.StringValue("test")}},
		{"SVCB missing fields", dnsRecordModel{Type: types.StringValue("SVCB"), Name: types.StringValue("test")}},
		{"TLSA missing fields", dnsRecordModel{Type: types.StringValue("TLSA"), Name: types.StringValue("test")}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			list := buildRecordList(t, tc.model)
			_, diags := expandDNSRecords(ctx, list, path.Root("records"))
			if !diags.HasError() {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}
