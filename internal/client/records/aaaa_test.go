package records

import (
	"testing"
)

func validAAAARecord() *AAAARecord {
	return &AAAARecord{
		Address: "2001:db8::1",
		Name:    "myhost",
		TTL:     3600,
	}
}

func TestAAAARecord_Validate_ValidRecord(t *testing.T) {
	rec := validAAAARecord()
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestAAAARecord_ValidateAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"valid compressed", "2001:db8::1", false},
		{"valid loopback", "::1", false},
		{"valid unspecified", "::", false},
		{"valid max length", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false}, // exactly 39 chars (API cap)
		{"empty", "", true},
		{"not an ip", "notanip", true},
		{"too long", "2001:0db8:85a3:0000:0000:8a2e:0370:73344", true}, // 40 chars, over the API cap
		{"ipv4 rejected", "192.0.2.1", true},                           // API rejects plain IPv4 for AAAA
		{"ipv4 mapped rejected", "::ffff:192.0.2.1", true},             // API rejects IPv4-mapped IPv6 too
		{"ipv4 broadcast rejected", "255.255.255.255", true},           // another IPv4 form, same rule
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validAAAARecord()
			rec.Address = tt.address
			err := rec.ValidateAddress()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAddress(%q) error = %v, wantErr = %v", tt.address, err, tt.wantErr)
			}
		})
	}
}
