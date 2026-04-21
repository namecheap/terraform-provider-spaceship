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
	tests := []struct {
		name    string
		cname   string
		wantErr bool
	}{
		{"valid hostname", "target.example.com", false},
		{"valid subdomain", "sub.domain", false},
		{"valid single label", "myhost", false},
		{"apex rejected", "@", true},
		{"wildcard rejected", "*", true},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 254), true},
		{"starts with dot", ".invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validCNAMERecord()
			rec.CName = tt.cname
			err := rec.ValidateCName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCName(%q) error = %v, wantErr = %v", tt.cname, err, tt.wantErr)
			}
		})
	}
}

func TestCNAMERecord_ValidateName(t *testing.T) {
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
			rec := validCNAMERecord()
			rec.Name = tt.recName
			err := rec.ValidateName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr = %v", tt.recName, err, tt.wantErr)
			}
		})
	}
}

func TestCNAMERecord_ValidateTTL(t *testing.T) {
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
			rec := validCNAMERecord()
			rec.TTL = tt.ttl
			err := rec.ValidateTTL()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTTL(%d) error = %v, wantErr = %v", tt.ttl, err, tt.wantErr)
			}
		})
	}
}
