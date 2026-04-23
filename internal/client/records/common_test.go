package records

import (
	"strings"
	"testing"
)

// Shared hostname (Name) and TTL edge cases for all record types live in this
// file. Per-type _test.go files cover only fields unique to that record type
// (AliasName, CAA tag/flags/value, SRV target/priority/weight/port, etc.).
// Whether each type's Validate() actually invokes ValidateName / ValidateTTL
// is not asserted at the client layer; the full write → read round-trip in
// acceptance tests is the backstop for that wiring.

func TestValidateName(t *testing.T) {
	// Build a valid hostname of exactly 253 characters:
	// 3 labels of 63 chars + 1 label of 61 chars + 3 dots = 253
	label63 := strings.Repeat("a", 63)
	label61 := strings.Repeat("b", 61)
	name253 := label63 + "." + label63 + "." + label63 + "." + label61
	name254 := name253 + "c"

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid hostname", "myhost", false},
		{"valid apex", "@", false},
		{"valid wildcard", "*", false},
		{"valid subdomain", "sub.domain", false},
		{"single char label", "a", false},
		{"two single char labels", "a.b", false},
		{"exact max length 253", name253, false},
		{"empty", "", true},
		{"over max length 254", name254, true},
		{"starts with dot", ".invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q [len=%d]) error = %v, wantErr = %v", tt.input, len(tt.input), err, tt.wantErr)
			}
		})
	}
}

func TestValidateTTL(t *testing.T) {
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
			err := ValidateTTL(tt.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTTL(%d) error = %v, wantErr = %v", tt.ttl, err, tt.wantErr)
			}
		})
	}
}
