package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

// The timeouts block covers all four operations; delete makes a real API call.
func TestDNSRecordSchema_HasTimeoutsBlock(t *testing.T) {
	resp := &fwresource.SchemaResponse{}
	(&dnsRecordResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Blocks["timeouts"]; !ok {
		t.Fatal("expected a timeouts block in the spaceship_dns_record schema")
	}
}

func TestParseRecordID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		wantOK   bool
		wantDom  string
		wantType string
		wantName string
		wantSig  string
	}{
		{
			name:     "valid",
			id:       "example.com/TXT/@/v=spf1 -all",
			wantOK:   true,
			wantDom:  "example.com",
			wantType: "TXT",
			wantName: "@",
			wantSig:  "v=spf1 -all",
		},
		{
			name:    "signature may contain slashes",
			id:      "example.com/TXT/@/foo/bar/baz",
			wantOK:  true,
			wantSig: "foo/bar/baz",
		},
		{name: "trailing slash empties signature", id: "example.com/TXT/@/"},
		{name: "missing signature segment", id: "example.com/TXT/@"},
		{name: "empty domain", id: "/TXT/@/sig"},
		{name: "empty type", id: "example.com//@/sig"},
		{name: "empty name", id: "example.com/TXT//sig"},
		{name: "empty string", id: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dom, rt, nm, sig, ok := parseRecordID(tc.id)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if tc.wantDom != "" && dom != tc.wantDom {
				t.Errorf("domain = %q, want %q", dom, tc.wantDom)
			}
			if tc.wantType != "" && rt != tc.wantType {
				t.Errorf("type = %q, want %q", rt, tc.wantType)
			}
			if tc.wantName != "" && nm != tc.wantName {
				t.Errorf("name = %q, want %q", nm, tc.wantName)
			}
			if sig != tc.wantSig {
				t.Errorf("signature = %q, want %q", sig, tc.wantSig)
			}
		})
	}
}
