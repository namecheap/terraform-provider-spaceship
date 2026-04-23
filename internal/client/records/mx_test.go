package records

import (
	"strings"
	"testing"
)

func validMXRecord() *MXRecord {
	return &MXRecord{
		Name:       "@",
		Exchange:   "mail.example.com",
		Preference: 10,
		TTL:        3600,
	}
}

func TestMXRecord_Validate_ValidRecord(t *testing.T) {
	rec := validMXRecord()
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestMXRecord_Validate_BoundaryValues(t *testing.T) {
	rec := &MXRecord{
		Name:       "@",
		Exchange:   "a",
		Preference: 0,
		TTL:        60,
	}
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors for boundary values, got: %v", errs)
	}
}

func TestMXRecord_ValidateExchange(t *testing.T) {
	// wantErrContains pins the rejection branch for @/* so the test fails
	// loudly if rejection ever fires from ValidateName instead of the pre-check.
	tests := []struct {
		name            string
		exchange        string
		wantErr         bool
		wantErrContains string
	}{
		{"valid hostname", "mail.example.com", false, ""},
		{"valid single char", "a", false, ""},
		{"valid subdomain", "sub.domain", false, ""},
		{"max length", strings.Repeat("a", 63) + "." + strings.Repeat("b", 63), false, ""},
		{"empty", "", true, ""},
		{"apex rejected", "@", true, "domain name"},
		{"wildcard rejected", "*", true, "domain name"},
		{"too long", strings.Repeat("a", 254), true, ""},
		{"starts with dot", ".mail.example.com", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validMXRecord()
			rec.Exchange = tt.exchange
			err := rec.ValidateExchange()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExchange(%q) error = %v, wantErr = %v", tt.exchange, err, tt.wantErr)
			}
			if tt.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrContains)) {
				t.Errorf("ValidateExchange(%q) error = %v, want substring %q", tt.exchange, err, tt.wantErrContains)
			}
		})
	}
}

func TestMXRecord_ValidatePreference(t *testing.T) {
	tests := []struct {
		name       string
		preference int
		wantErr    bool
	}{
		{"valid zero", 0, false},
		{"valid mid", 10, false},
		{"valid max", 65535, false},
		{"invalid negative", -1, true},
		{"invalid over max", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validMXRecord()
			rec.Preference = tt.preference
			err := rec.ValidatePreference()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePreference(%d) error = %v, wantErr = %v", tt.preference, err, tt.wantErr)
			}
		})
	}
}

// Shared hostname (Name) and TTL edge cases live in common_test.go.
// Per-type ValidateName / ValidateTTL wiring is covered by TestMXRecord_Validate_ValidRecord.
