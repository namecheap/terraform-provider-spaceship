package records

import (
	"strings"
	"testing"
)

func TestARecord_Validate_MultipleErrors(t *testing.T) {
	rec := &ARecord{
		Address: "not-an-ip",
		Name:    "",
		TTL:     0,
	}

	errs := rec.Validate()
	if len(errs) != 3 {
		t.Errorf("expected 3 errors, got %d: %v", len(errs), errs)
	}
}

func TestSRVRecord_Validate_MultipleErrors(t *testing.T) {
	rec := &SRVRecord{
		Service:  "",
		Protocol: "",
		Priority: -1,
		Weight:   -1,
		Port:     0,
		Target:   "",
		Name:     "",
		TTL:      0,
	}

	errs := rec.Validate()
	if len(errs) != 8 {
		t.Errorf("expected 8 errors (one per field), got %d: %v", len(errs), errs)
	}
}

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

func TestSRVRecord_ValidateTarget_EdgeCases(t *testing.T) {
	// Build a long valid target: 3 labels of 63 chars + 1 label of 61 chars + 3 dots = 253
	label63 := strings.Repeat("a", 63)
	label61 := strings.Repeat("b", 61)
	longValid := label63 + "." + label63 + "." + label63 + "." + label61

	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{"root dot", ".", true},
		{"long valid target at 253", longValid, false},
		{"consecutive dots", "..", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validSRVRecord()
			rec.Target = tt.target
			err := rec.ValidateTarget()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTarget(%q [len=%d]) error = %v, wantErr = %v", tt.target, len(tt.target), err, tt.wantErr)
			}
		})
	}
}
