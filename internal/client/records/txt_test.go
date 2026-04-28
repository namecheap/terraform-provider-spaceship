package records

import (
	"strings"
	"testing"
)

func validTXTRecord() *TXTRecord {
	return &TXTRecord{
		Name:  "@",
		Value: "v=spf1 a mx -all",
		TTL:   3600,
	}
}

func TestTXTRecord_Validate_ValidRecord(t *testing.T) {
	rec := validTXTRecord()
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestTXTRecord_Validate_BoundaryValues(t *testing.T) {
	rec := &TXTRecord{
		Name:  "host",
		Value: strings.Repeat("a", 65535),
		TTL:   60,
	}
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors for boundary values, got: %v", errs)
	}
}

func TestTXTRecord_ValidateValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid spf", "v=spf1 a mx -all", false},
		{"valid dkim", "v=DKIM1; k=rsa; p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQ", false},
		{"valid verification token", "google-site-verification=abcdef1234567890", false},
		{"valid with quotes", `"quoted segment" "another"`, false},
		{"valid single char", "a", false},
		{"valid multibyte unicode", "café résumé naïve", false},
		{"valid newline (pattern is .*)", "line1\nline2", false},
		{"valid leading space (api accepts)", " v=spf1", false},
		{"valid trailing space (api accepts)", "v=spf1 ", false},
		{"valid trailing LF (api accepts)", "v=spf1\n", false},
		{"valid max length", strings.Repeat("a", 65535), false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 65536), true},
		// Whitespace-only is rejected: the API returns "Value field is
		// required" for these even though the spec says minLength=1.
		{"single space (api rejects)", " ", true},
		{"single tab (api rejects)", "\t", true},
		{"single LF (api rejects)", "\n", true},
		{"single CR (api rejects)", "\r", true},
		{"CRLF only (api rejects)", "\r\n", true},
		{"multiple spaces (api rejects)", "    ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validTXTRecord()
			rec.Value = tt.value
			err := rec.ValidateValue()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateValue(len=%d) error = %v, wantErr = %v", len(tt.value), err, tt.wantErr)
			}
		})
	}
}
