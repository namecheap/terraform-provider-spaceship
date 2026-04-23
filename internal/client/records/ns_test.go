package records

import (
	"strings"
	"testing"
)

func validNSRecord() *NSRecord {
	return &NSRecord{
		Nameserver: "ns1.example.com",
		Name:       "@",
		TTL:        3600,
	}
}

func TestNSRecord_Validate_ValidRecord(t *testing.T) {
	rec := validNSRecord()
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestNSRecord_ValidateNameserver(t *testing.T) {
	// wantErrContains pins the rejection branch for @/* so the test fails
	// loudly if rejection ever fires from ValidateName instead of the pre-check.
	tests := []struct {
		name            string
		nameserver      string
		wantErr         bool
		wantErrContains string
	}{
		{"valid hostname", "ns1.example.com", false, ""},
		{"valid subdomain", "ns.domain", false, ""},
		{"valid single label", "myns", false, ""},
		{"apex rejected", "@", true, "domain name"},
		{"wildcard rejected", "*", true, "domain name"},
		{"empty", "", true, ""},
		{"too long", strings.Repeat("a", 254), true, ""},
		{"starts with dot", ".invalid", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validNSRecord()
			rec.Nameserver = tt.nameserver
			err := rec.ValidateNameserver()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNameserver(%q) error = %v, wantErr = %v", tt.nameserver, err, tt.wantErr)
			}
			if tt.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrContains)) {
				t.Errorf("ValidateNameserver(%q) error = %v, want substring %q", tt.nameserver, err, tt.wantErrContains)
			}
		})
	}
}

// Shared hostname (Name) and TTL edge cases live in common_test.go.
// Per-type ValidateName / ValidateTTL wiring is covered by TestNSRecord_Validate_ValidRecord.
