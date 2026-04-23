package records

import (
	"strings"
	"testing"
)

func validCNAMERecord() *CNAMERecord {
	return &CNAMERecord{
		CName: "target.example.com",
		Name:  "www",
		TTL:   3600,
	}
}

func TestCNAMERecord_Validate_ValidRecord(t *testing.T) {
	rec := validCNAMERecord()
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestCNAMERecord_ValidateCName(t *testing.T) {
	// wantErrContains pins the rejection branch for @/* so the test fails
	// loudly if rejection ever fires from ValidateName instead of the pre-check.
	tests := []struct {
		name            string
		cname           string
		wantErr         bool
		wantErrContains string
	}{
		{"valid hostname", "target.example.com", false, ""},
		{"valid subdomain", "sub.domain", false, ""},
		{"valid single label", "myhost", false, ""},
		{"apex rejected", "@", true, "domain name"},
		{"wildcard rejected", "*", true, "domain name"},
		{"empty", "", true, ""},
		{"too long", strings.Repeat("a", 254), true, ""},
		{"starts with dot", ".invalid", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validCNAMERecord()
			rec.CName = tt.cname
			err := rec.ValidateCName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCName(%q) error = %v, wantErr = %v", tt.cname, err, tt.wantErr)
			}
			if tt.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrContains)) {
				t.Errorf("ValidateCName(%q) error = %v, want substring %q", tt.cname, err, tt.wantErrContains)
			}
		})
	}
}
