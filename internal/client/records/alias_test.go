package records

import (
	"strings"
	"testing"
)

func validALIASRecord() *ALIASRecord {
	return &ALIASRecord{
		AliasName: "other.example.com",
		Name:      "@",
		TTL:       3600,
	}
}

func TestALIASRecord_Validate_ValidRecord(t *testing.T) {
	rec := validALIASRecord()
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestALIASRecord_ValidateAliasName(t *testing.T) {
	// wantErrContains pins the rejection branch for @/* so the test fails
	// loudly if rejection ever fires from ValidateName instead of the pre-check.
	tests := []struct {
		name            string
		aliasName       string
		wantErr         bool
		wantErrContains string
	}{
		{"valid hostname", "other.example.com", false, ""},
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
			rec := validALIASRecord()
			rec.AliasName = tt.aliasName
			err := rec.ValidateAliasName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAliasName(%q) error = %v, wantErr = %v", tt.aliasName, err, tt.wantErr)
			}
			if tt.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrContains)) {
				t.Errorf("ValidateAliasName(%q) error = %v, want substring %q", tt.aliasName, err, tt.wantErrContains)
			}
		})
	}
}
