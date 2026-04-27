package records

import (
	"strings"
	"testing"
)

func validPTRRecord() *PTRRecord {
	return &PTRRecord{
		Pointer: "host.example.com",
		Name:    "1",
		TTL:     3600,
	}
}

func TestPTRRecord_Validate_ValidRecord(t *testing.T) {
	rec := validPTRRecord()
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestPTRRecord_ValidatePointer(t *testing.T) {
	// wantErrContains pins the rejection branch for @/* so the test fails
	// loudly if rejection ever fires from ValidateName instead of the pre-check.
	tests := []struct {
		name            string
		pointer         string
		wantErr         bool
		wantErrContains string
	}{
		{"valid hostname", "host.example.com", false, ""},
		{"valid subdomain", "sub.domain", false, ""},
		{"valid single label", "myhost", false, ""},
		{"apex rejected", "@", true, "domain name"},
		{"wildcard rejected", "*", true, "domain name"},
		{"wildcard prefix rejected", "*.example.com", true, "domain name"},
		{"empty", "", true, ""},
		{"too long", strings.Repeat("a", 254), true, ""},
		{"starts with dot", ".invalid", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validPTRRecord()
			rec.Pointer = tt.pointer
			err := rec.ValidatePointer()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePointer(%q) error = %v, wantErr = %v", tt.pointer, err, tt.wantErr)
			}
			if tt.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrContains)) {
				t.Errorf("ValidatePointer(%q) error = %v, want substring %q", tt.pointer, err, tt.wantErrContains)
			}
		})
	}
}

func TestPTRRecord_ValidateName(t *testing.T) {
	tests := []struct {
		name    string
		recName string
		wantErr bool
	}{
		{"valid hostname", "1", false},
		{"valid apex", "@", false},
		{"valid wildcard", "*", false},
		{"valid subdomain", "1.0.168.192", false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 254), true},
		{"starts with dot", ".invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validPTRRecord()
			rec.Name = tt.recName
			err := rec.ValidateName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr = %v", tt.recName, err, tt.wantErr)
			}
		})
	}
}

func TestPTRRecord_ValidateTTL(t *testing.T) {
	tests := []struct {
		name    string
		ttl     int
		wantErr bool
	}{
		{"valid", 3500, false},
		{"min valid", 60, false},
		{"max valid", 3600, false},
		{"too low", 59, true},
		{"too high", 3601, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validPTRRecord()
			rec.TTL = tt.ttl
			err := rec.ValidateTTL()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTTL(%d) error = %v, wantErr = %v", tt.ttl, err, tt.wantErr)
			}
		})
	}
}
