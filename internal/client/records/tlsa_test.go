package records

import (
	"strings"
	"testing"
)

// 64-char SHA-256 hex digest used as a known-valid associationData payload.
const validTLSAAssociation = "7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069"

func validTLSARecord() *TLSARecord {
	return &TLSARecord{
		Port:            "_443",
		Protocol:        "_tcp",
		Usage:           2,
		Selector:        1,
		Matching:        1,
		AssociationData: validTLSAAssociation,
		Name:            "@",
		TTL:             3600,
	}
}

func TestTLSARecord_Validate_ValidRecord(t *testing.T) {
	rec := validTLSARecord()
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestTLSARecord_Validate_BoundaryValues(t *testing.T) {
	rec := &TLSARecord{
		Port:            "*",
		Protocol:        "_udp",
		Usage:           0,
		Selector:        0,
		Matching:        0,
		AssociationData: validTLSAAssociation,
		Name:            "@",
		TTL:             60,
	}
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors for boundary values, got: %v", errs)
	}
}

func TestTLSARecord_ValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr bool
	}{
		{"valid wildcard", "*", false},
		{"valid _443", "_443", false},
		{"valid _1", "_1", false},
		{"valid _65535", "_65535", false},
		{"empty rejected", "", true},
		{"invalid no underscore", "443", true},
		{"invalid zero", "_0", true},
		{"invalid too high", "_65536", true},
		{"invalid non-numeric", "_abc", true},
		{"invalid only underscore", "_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validTLSARecord()
			rec.Port = tt.port
			err := rec.ValidatePort()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort(%q) error = %v, wantErr = %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestTLSARecord_ValidateProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		wantErr  bool
	}{
		{"valid _tcp", "_tcp", false},
		{"valid _udp", "_udp", false},
		{"valid with hyphen", "_my-proto", false},
		{"valid max length", "_" + strings.Repeat("a", 62), false},
		{"empty", "", true},
		{"too short", "_", true},
		{"missing underscore", "tcp", true},
		{"too long", "_" + strings.Repeat("a", 63), true},
		{"invalid char", "_tcp!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validTLSARecord()
			rec.Protocol = tt.protocol
			err := rec.ValidateProtocol()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProtocol(%q) error = %v, wantErr = %v", tt.protocol, err, tt.wantErr)
			}
		})
	}
}

func TestTLSARecord_ValidateUsage(t *testing.T) {
	tests := []struct {
		name    string
		usage   int
		wantErr bool
	}{
		{"valid zero", 0, false},
		{"valid mid", 3, false},
		{"valid max", 255, false},
		{"invalid negative", -1, true},
		{"invalid too high", 256, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validTLSARecord()
			rec.Usage = tt.usage
			err := rec.ValidateUsage()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsage(%d) error = %v, wantErr = %v", tt.usage, err, tt.wantErr)
			}
		})
	}
}

func TestTLSARecord_ValidateSelector(t *testing.T) {
	tests := []struct {
		name     string
		selector int
		wantErr  bool
	}{
		{"valid zero", 0, false},
		{"valid one", 1, false},
		{"valid max", 255, false},
		{"invalid negative", -1, true},
		{"invalid too high", 256, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validTLSARecord()
			rec.Selector = tt.selector
			err := rec.ValidateSelector()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSelector(%d) error = %v, wantErr = %v", tt.selector, err, tt.wantErr)
			}
		})
	}
}

func TestTLSARecord_ValidateMatching(t *testing.T) {
	tests := []struct {
		name     string
		matching int
		wantErr  bool
	}{
		{"valid zero", 0, false},
		{"valid one", 1, false},
		{"valid max", 255, false},
		{"invalid negative", -1, true},
		{"invalid too high", 256, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validTLSARecord()
			rec.Matching = tt.matching
			err := rec.ValidateMatching()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMatching(%d) error = %v, wantErr = %v", tt.matching, err, tt.wantErr)
			}
		})
	}
}

func TestTLSARecord_ValidateAssociationData(t *testing.T) {
	// 32 lowercase hex byte pairs separated by single spaces = 95 chars.
	spaced := strings.TrimSpace(strings.Repeat("ab ", 32))
	// 65535-char input: a single space pushes the total up by one over an
	// otherwise-even hex stream, so the inclusive upper bound is reachable.
	upperBound := "ab " + strings.Repeat("ab", 32766)

	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{"valid lowercase 64", strings.Repeat("ab", 32), false},
		{"valid sha256 digest", validTLSAAssociation, false},
		{"valid uppercase", strings.ToUpper(validTLSAAssociation), false},
		{"valid mixed case", "Ab" + validTLSAAssociation[2:], false},
		{"valid spaced byte pairs", spaced, false},
		{"valid max length 65534", strings.Repeat("ab", 32767), false},
		{"valid max length 65535", upperBound, false},
		{"empty", "", true},
		{"too short 62", strings.Repeat("ab", 31), true},
		{"too short 63", strings.Repeat("ab", 31) + "a", true},
		{"too long 65536", strings.Repeat("ab", 32768), true},
		{"non-hex char", strings.Repeat("zz", 32), true},
		{"odd hex length", validTLSAAssociation + "a", true},
		{"leading space", " " + validTLSAAssociation, true},
		{"trailing space", validTLSAAssociation + " ", true},
		{"double space between pairs", "ab" + strings.Repeat("  ab", 31), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validTLSARecord()
			rec.AssociationData = tt.data
			err := rec.ValidateAssociationData()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAssociationData(len=%d) error = %v, wantErr = %v", len(tt.data), err, tt.wantErr)
			}
		})
	}
}
