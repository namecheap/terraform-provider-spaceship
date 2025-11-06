package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var testRecordAttrTypes = map[string]attr.Type{
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
}

func buildRecordList(t *testing.T, models ...dnsRecordModel) types.List {
	t.Helper()
	ctx := context.Background()
	list, diag := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: testRecordAttrTypes}, models)
	if diag.HasError() {
		t.Fatalf("failed to build record list: %s", diag)
	}
	return list
}

func TestExpandDNSRecords_AllTypes(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name     string
		model    dnsRecordModel
		expected DNSRecord
	}{
		{
			name: "A",
			model: dnsRecordModel{
				Type:    types.StringValue("A"),
				Name:    types.StringValue("a"),
				Address: types.StringValue("192.0.2.1"),
			},
			expected: DNSRecord{Type: "A", Name: "a", TTL: 3600, Address: "192.0.2.1"},
		},
		{
			name: "AAAA",
			model: dnsRecordModel{
				Type:    types.StringValue("AAAA"),
				Name:    types.StringValue("ipv6"),
				Address: types.StringValue("2001:db8::1"),
			},
			expected: DNSRecord{Type: "AAAA", Name: "ipv6", TTL: 3600, Address: "2001:db8::1"},
		},
		{
			name: "ALIAS",
			model: dnsRecordModel{
				Type:      types.StringValue("ALIAS"),
				Name:      types.StringValue("alias"),
				AliasName: types.StringValue("@"),
			},
			expected: DNSRecord{Type: "ALIAS", Name: "alias", TTL: 3600, AliasName: "@"},
		},
		{
			name: "CAA",
			model: dnsRecordModel{
				Type:  types.StringValue("CAA"),
				Name:  types.StringValue("caa"),
				Flag:  types.Int64Value(0),
				Tag:   types.StringValue("issue"),
				Value: types.StringValue("letsencrypt.org"),
			},
			expected: DNSRecord{Type: "CAA", Name: "caa", TTL: 3600, Flag: intPtr(0), Tag: "issue", Value: "letsencrypt.org"},
		},
		{
			name: "CNAME",
			model: dnsRecordModel{
				Type:  types.StringValue("CNAME"),
				Name:  types.StringValue("cname"),
				CName: types.StringValue("origin.example.com"),
			},
			expected: DNSRecord{Type: "CNAME", Name: "cname", TTL: 3600, CName: "origin.example.com"},
		},
		{
			name: "HTTPS",
			model: dnsRecordModel{
				Type:        types.StringValue("HTTPS"),
				Name:        types.StringValue("https"),
				SvcPriority: types.Int64Value(1),
				TargetName:  types.StringValue("_8443._https.example.com"),
				SvcParams:   types.StringValue("alpn=h2"),
				Port:        types.StringValue("_8443"),
				Scheme:      types.StringValue("_https"),
			},
			expected: DNSRecord{Type: "HTTPS", Name: "https", TTL: 3600, SvcPriority: intPtr(1), TargetName: "_8443._https.example.com", SvcParams: "alpn=h2", Port: NewStringPortValue("_8443"), Scheme: "_https"},
		},
		{
			name: "MX",
			model: dnsRecordModel{
				Type:       types.StringValue("MX"),
				Name:       types.StringValue("mx"),
				Exchange:   types.StringValue("mail.example.com"),
				Preference: types.Int64Value(10),
			},
			expected: DNSRecord{Type: "MX", Name: "mx", TTL: 3600, Exchange: "mail.example.com", Preference: intPtr(10)},
		},
		{
			name: "NS",
			model: dnsRecordModel{
				Type:       types.StringValue("NS"),
				Name:       types.StringValue("ns"),
				Nameserver: types.StringValue("ns1.example.com"),
			},
			expected: DNSRecord{Type: "NS", Name: "ns", TTL: 3600, Nameserver: "ns1.example.com"},
		},
		{
			name: "PTR",
			model: dnsRecordModel{
				Type:    types.StringValue("PTR"),
				Name:    types.StringValue("ptr"),
				Pointer: types.StringValue("ptr.example.com"),
			},
			expected: DNSRecord{Type: "PTR", Name: "ptr", TTL: 3600, Pointer: "ptr.example.com"},
		},
		{
			name: "SRV",
			model: dnsRecordModel{
				Type:       types.StringValue("SRV"),
				Name:       types.StringValue("_sip._tcp"),
				Service:    types.StringValue("_sip"),
				Protocol:   types.StringValue("_tcp"),
				Priority:   types.Int64Value(5),
				Weight:     types.Int64Value(10),
				PortNumber: types.Int64Value(5060),
				Target:     types.StringValue("srv.example.com"),
			},
			expected: DNSRecord{Type: "SRV", Name: "_sip._tcp", TTL: 3600, Service: "_sip", Protocol: "_tcp", Priority: intPtr(5), Weight: intPtr(10), Port: NewIntPortValue(5060), Target: "srv.example.com"},
		},
		{
			name: "SVCB",
			model: dnsRecordModel{
				Type:        types.StringValue("SVCB"),
				Name:        types.StringValue("svcb"),
				SvcPriority: types.Int64Value(1),
				TargetName:  types.StringValue("svc.example.com"),
				SvcParams:   types.StringValue("alpn=h2"),
				Port:        types.StringValue("_853"),
				Scheme:      types.StringValue("_dot"),
			},
			expected: DNSRecord{Type: "SVCB", Name: "svcb", TTL: 3600, SvcPriority: intPtr(1), TargetName: "svc.example.com", SvcParams: "alpn=h2", Port: NewStringPortValue("_853"), Scheme: "_dot"},
		},
		{
			name: "TLSA",
			model: dnsRecordModel{
				Type:            types.StringValue("TLSA"),
				Name:            types.StringValue("_443._tcp"),
				Port:            types.StringValue("_443"),
				Protocol:        types.StringValue("_tcp"),
				Usage:           types.Int64Value(2),
				Selector:        types.Int64Value(1),
				Matching:        types.Int64Value(1),
				AssociationData: types.StringValue("7F83B1657FF1FC53"),
			},
			expected: DNSRecord{Type: "TLSA", Name: "_443._tcp", TTL: 3600, Port: NewStringPortValue("_443"), Protocol: "_tcp", Usage: intPtr(2), Selector: intPtr(1), Matching: intPtr(1), AssociationData: "7F83B1657FF1FC53"},
		},
		{
			name: "TXT",
			model: dnsRecordModel{
				Type:  types.StringValue("TXT"),
				Name:  types.StringValue("txt"),
				Value: types.StringValue("hello"),
			},
			expected: DNSRecord{Type: "TXT", Name: "txt", TTL: 3600, Value: "hello"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list := buildRecordList(t, tc.model)
			records, diags := expandDNSRecords(ctx, list, path.Root("records"))
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}
			if len(records) != 1 {
				t.Fatalf("expected one record, got %d", len(records))
			}
			if !reflect.DeepEqual(tc.expected, records[0]) {
				t.Fatalf("unexpected record: %#v", records[0])
			}
		})
	}
}

func TestExpandDNSRecords_MissingAddress(t *testing.T) {
	ctx := context.Background()
	list := buildRecordList(t, dnsRecordModel{
		Type: types.StringValue("A"),
		Name: types.StringValue("missing"),
	})

	_, diags := expandDNSRecords(ctx, list, path.Root("records"))
	if !diags.HasError() {
		t.Fatalf("expected diagnostics for missing address")
	}
}

func TestDiffDNSRecords_NoChanges(t *testing.T) {
	existing := []DNSRecord{
		{Type: "A", Name: "tf", Address: "198.51.100.10", TTL: 3600},
	}

	toDelete, toUpsert := diffDNSRecords(existing, existing)

	if len(toDelete) != 0 {
		t.Fatalf("expected no deletions, got %#v", toDelete)
	}

	if len(toUpsert) != 0 {
		t.Fatalf("expected no upserts, got %#v", toUpsert)
	}
}

func TestDiffDNSRecords_AddressChange(t *testing.T) {
	existing := []DNSRecord{
		{Type: "A", Name: "tf", Address: "198.51.100.10", TTL: 3600},
	}
	desired := []DNSRecord{
		{Type: "A", Name: "tf", Address: "198.51.100.11", TTL: 3600},
	}

	toDelete, toUpsert := diffDNSRecords(existing, desired)

	if !reflect.DeepEqual(toDelete, existing) {
		t.Fatalf("expected deletion of existing record, got %#v", toDelete)
	}

	if !reflect.DeepEqual(toUpsert, desired) {
		t.Fatalf("expected upsert of desired record, got %#v", toUpsert)
	}
}

func TestDiffDNSRecords_TTLChange(t *testing.T) {
	existing := []DNSRecord{
		{Type: "A", Name: "tf", Address: "198.51.100.10", TTL: 3600},
	}
	desired := []DNSRecord{
		{Type: "A", Name: "tf", Address: "198.51.100.10", TTL: 600},
	}

	toDelete, toUpsert := diffDNSRecords(existing, desired)

	if len(toDelete) != 0 {
		t.Fatalf("expected no deletions, got %#v", toDelete)
	}

	if !reflect.DeepEqual(toUpsert, desired) {
		t.Fatalf("expected TTL update, got %#v", toUpsert)
	}
}

func TestDiffDNSRecords_RecordRemoval(t *testing.T) {
	existing := []DNSRecord{
		{Type: "A", Name: "tf", Address: "198.51.100.10", TTL: 3600},
		{Type: "A", Name: "tf2", Address: "198.51.100.11", TTL: 3600},
	}
	desired := []DNSRecord{
		{Type: "A", Name: "tf", Address: "198.51.100.10", TTL: 3600},
	}

	toDelete, toUpsert := diffDNSRecords(existing, desired)

	if len(toDelete) != 1 {
		t.Fatalf("expected one deletion, got %#v", toDelete)
	}

	if len(toUpsert) != 0 {
		t.Fatalf("expected no upserts, got %#v", toUpsert)
	}
}

func TestOrderDNSRecordsLike(t *testing.T) {
	reference := []DNSRecord{
		{Type: "A", Name: "@", Address: "198.51.100.10", TTL: 3600},
		{Type: "TXT", Name: "@", Value: "hi", TTL: 3600},
		{Type: "AAAA", Name: "@", Address: "2001:db8::1", TTL: 3600},
	}

	records := []DNSRecord{
		{Type: "TXT", Name: "@", Value: "hi", TTL: 3600},
		{Type: "AAAA", Name: "@", Address: "2001:db8::1", TTL: 3600},
		{Type: "A", Name: "@", Address: "198.51.100.10", TTL: 3600},
	}

	ordered := orderDNSRecordsLike(reference, records)

	if !reflect.DeepEqual(ordered, reference) {
		t.Fatalf("expected ordered records %#v, got %#v", reference, ordered)
	}
}

func TestRecordValueSignatureTLSA(t *testing.T) {
	rec1 := DNSRecord{
		Type:            "TLSA",
		Name:            "_443._tcp",
		Port:            NewStringPortValue("_443"),
		Protocol:        "_tcp",
		Usage:           intPtr(2),
		Selector:        intPtr(1),
		Matching:        intPtr(1),
		AssociationData: "7F83B1 657FF1FC53",
	}

	rec2 := DNSRecord{
		Type:            "TLSA",
		Name:            "_443._tcp",
		Port:            NewStringPortValue("_443"),
		Protocol:        "_tcp",
		Usage:           intPtr(2),
		Selector:        intPtr(1),
		Matching:        intPtr(1),
		AssociationData: "7f83b1657ff1fc53",
	}

	if recordValueSignature(rec1) != recordValueSignature(rec2) {
		t.Fatalf("expected TLSA signatures to match despite spacing and case differences")
	}
}

func intPtr(v int) *int {
	return &v
}
