package records

import (
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
	tests := []struct {
		name      string
		aliasName string
		wantErr   bool
	}{
		{"valid hostname", "other.example.com", false},
		{"valid subdomain", "sub.domain", false},
		{"valid single label", "myhost", false},
		{"apex rejected", "@", true},
		{"wildcard rejected", "*", true},
		{"empty", "", true},
		{"too long", string(make([]byte, 254)), true},
		{"starts with dot", ".invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validALIASRecord()
			rec.AliasName = tt.aliasName
			err := rec.ValidateAliasName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAliasName(%q) error = %v, wantErr = %v", tt.aliasName, err, tt.wantErr)
			}
		})
	}
}

func TestALIASRecord_ValidateName(t *testing.T) {
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
		{"too long", string(make([]byte, 254)), true},
		{"starts with dot", ".invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validALIASRecord()
			rec.Name = tt.recName
			err := rec.ValidateName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr = %v", tt.recName, err, tt.wantErr)
			}
		})
	}
}

func TestALIASRecord_ValidateTTL(t *testing.T) {
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
			rec := validALIASRecord()
			rec.TTL = tt.ttl
			err := rec.ValidateTTL()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTTL(%d) error = %v, wantErr = %v", tt.ttl, err, tt.wantErr)
			}
		})
	}
}
