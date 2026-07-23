package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

// The timeouts block covers all four operations — unlike spaceship_domain,
// Delete here makes a real API call.
func TestPersonalNameserverSchema_HasTimeoutsBlock(t *testing.T) {
	resp := &fwresource.SchemaResponse{}
	(&personalNameserverResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Blocks["timeouts"]; !ok {
		t.Fatal("expected a timeouts block in the spaceship_personal_nameserver schema")
	}
}

func TestPersonalNameserverID_RoundTrip(t *testing.T) {
	id := personalNameserverID("example.com", "ns1")
	if id != "example.com/ns1" {
		t.Fatalf("unexpected id %q", id)
	}

	domain, host, ok := parsePersonalNameserverID(id)
	if !ok || domain != "example.com" || host != "ns1" {
		t.Fatalf("round-trip failed: domain=%q host=%q ok=%v", domain, host, ok)
	}
}

func TestParsePersonalNameserverID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		wantOK     bool
		wantDomain string
		wantHost   string
	}{
		{"valid", "example.com/ns1", true, "example.com", "ns1"},
		{"empty domain", "/ns1", false, "", ""},
		{"empty host", "example.com/", false, "", ""},
		{"no separator", "example.com", false, "", ""},
		{"empty string", "", false, "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			domain, host, ok := parsePersonalNameserverID(tc.id)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && (domain != tc.wantDomain || host != tc.wantHost) {
				t.Errorf("got (%q, %q), want (%q, %q)", domain, host, tc.wantDomain, tc.wantHost)
			}
		})
	}
}
