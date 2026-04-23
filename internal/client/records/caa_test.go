package records

import (
	"strings"
	"testing"
)

func validCAARecord() *CAARecord {
	return &CAARecord{
		Name:  "@",
		Flag:  0,
		Tag:   "issue",
		Value: "letsencrypt.org",
		TTL:   3600,
	}
}

func TestCAARecord_Validate_ValidRecord(t *testing.T) {
	rec := validCAARecord()
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestCAARecord_Validate_BoundaryValues(t *testing.T) {
	rec := &CAARecord{
		Name:  "@",
		Flag:  128,
		Tag:   "iodef",
		Value: "mailto:ca@example.com",
		TTL:   60,
	}
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors for boundary values, got: %v", errs)
	}
}

func TestCAARecord_ValidateFlag(t *testing.T) {
	tests := []struct {
		name    string
		flag    int
		wantErr bool
	}{
		{"valid zero", 0, false},
		{"valid critical", 128, false},
		{"invalid one", 1, true},
		{"invalid negative", -1, true},
		{"invalid high", 256, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validCAARecord()
			rec.Flag = tt.flag
			err := rec.ValidateFlag()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFlag(%d) error = %v, wantErr = %v", tt.flag, err, tt.wantErr)
			}
		})
	}
}

func TestCAARecord_ValidateTag(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		wantErr bool
	}{
		{"valid issue", "issue", false},
		{"valid issuewild", "issuewild", false},
		{"valid iodef", "iodef", false},
		{"empty", "", true},
		{"unknown", "unknown", true},
		{"uppercase rejected", "ISSUE", true},
		{"mixed case rejected", "Issue", true},
		{"whitespace", " issue", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validCAARecord()
			rec.Tag = tt.tag
			err := rec.ValidateTag()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTag(%q) error = %v, wantErr = %v", tt.tag, err, tt.wantErr)
			}
		})
	}
}

func TestCAARecord_ValidateValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid CA", "letsencrypt.org", false},
		{"valid mailto", "mailto:ca@example.com", false},
		{"valid parameters", "letsencrypt.org; account=12345", false},
		{"valid semicolon", ";", false},
		{"valid max length", strings.Repeat("a", 256), false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 257), true},
		{"non-printable tab", "letsencrypt.org\t", true},
		{"non-printable newline", "letsencrypt.org\n", true},
		{"non-ASCII", "café.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validCAARecord()
			rec.Value = tt.value
			err := rec.ValidateValue()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateValue(%q) error = %v, wantErr = %v", tt.value, err, tt.wantErr)
			}
		})
	}
}
