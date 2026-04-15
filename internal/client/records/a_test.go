package records

import (
	"testing"
)

func validARecord() *ARecord {
	return &ARecord{
		Address: "192.168.1.1",
		Name:    "myhost",
		TTL:     3600,
	}
}

func TestARecord_Validate_ValidRecord(t *testing.T) {
	rec := validARecord()
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestARecord_ValidateAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"valid", "192.168.1.1", false},
		{"valid zeros", "0.0.0.0", false},
		{"valid max", "255.255.255.255", false},
		{"valid loopback", "127.0.0.1", false},
		{"empty", "", true},
		{"ipv6", "::1", true},
		{"ipv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"not an ip", "notanip", true},
		{"too many octets", "1.2.3.4.5", true},
		{"letters", "abc.def.ghi.jkl", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validARecord()
			rec.Address = tt.address
			err := rec.ValidateAddress()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAddress(%q) error = %v, wantErr = %v", tt.address, err, tt.wantErr)
			}
		})
	}
}

func TestARecord_ValidateName(t *testing.T) {
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
			rec := validARecord()
			rec.Name = tt.recName
			err := rec.ValidateName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr = %v", tt.recName, err, tt.wantErr)
			}
		})
	}
}

func TestARecord_ValidateTTL(t *testing.T) {
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
			rec := validARecord()
			rec.TTL = tt.ttl
			err := rec.ValidateTTL()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTTL(%d) error = %v, wantErr = %v", tt.ttl, err, tt.wantErr)
			}
		})
	}
}
