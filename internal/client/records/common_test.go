package records

import (
	"strings"
	"testing"
)

func TestValidateName_EdgeCases(t *testing.T) {
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
		{"single char label", "a", false},
		{"two single char labels", "a.b", false},
		{"exact max length 253", name253, false},
		{"over max length 254", name254, true},
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
