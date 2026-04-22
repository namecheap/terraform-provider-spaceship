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
	tests := []struct {
		name     string
		exchange string
		wantErr  bool
	}{
		{"valid hostname", "mail.example.com", false},
		{"valid single char", "a", false},
		{"valid subdomain", "sub.domain", false},
		{"max length", strings.Repeat("a", 63) + "." + strings.Repeat("b", 63), false},
		{"empty", "", true},
		{"apex rejected", "@", true},
		{"wildcard rejected", "*", true},
		{"too long", strings.Repeat("a", 254), true},
		{"starts with dot", ".mail.example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validMXRecord()
			rec.Exchange = tt.exchange
			err := rec.ValidateExchange()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExchange(%q) error = %v, wantErr = %v", tt.exchange, err, tt.wantErr)
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

func TestMXRecord_ValidateName(t *testing.T) {
	tests := []struct {
		name    string
		recName string
		wantErr bool
	}{
		{"valid hostname", "myhost", false},
		{"valid apex", "@", false},
		{"valid wildcard", "*", false},
		{"valid subdomain", "sub.domain", false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 254), true},
		{"starts with dot", ".invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validMXRecord()
			rec.Name = tt.recName
			err := rec.ValidateName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr = %v", tt.recName, err, tt.wantErr)
			}
		})
	}
}

func TestMXRecord_ValidateTTL(t *testing.T) {
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
			rec := validMXRecord()
			rec.TTL = tt.ttl
			err := rec.ValidateTTL()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTTL(%d) error = %v, wantErr = %v", tt.ttl, err, tt.wantErr)
			}
		})
	}
}
