package client

import (
	"testing"
)

func TestNewStringPortValue(t *testing.T) {
	pv := NewStringPortValue("_443")
	if pv.String == nil || *pv.String != "_443" {
		t.Errorf("expected string port _443, got %v", pv)
	}
	if pv.Int != nil {
		t.Error("expected Int to be nil")
	}
}

func TestNewIntPortValue(t *testing.T) {
	pv := NewIntPortValue(8080)
	if pv.Int == nil || *pv.Int != 8080 {
		t.Errorf("expected int port 8080, got %v", pv)
	}
	if pv.String != nil {
		t.Error("expected String to be nil")
	}
}

func TestPortValue_MarshallJSON(t *testing.T) {
	tests := []struct {
		name     string
		pv       *PortValue
		expected string
	}{
		{"nil", nil, "null"},
		{"int", NewIntPortValue(443), "443"},
		{"string", NewStringPortValue("_443"), `"_443"`},
		{"empty", &PortValue{}, "null"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.pv.MarshallJSON()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(data) != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, string(data))
			}
		})
	}
}

func TestPortValue_UnmarshallJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectInt *int
		expectStr *string
	}{
		{"null", "null", nil, nil},
		{"int", "443", intP(443), nil},
		{"string", `"_443"`, nil, strP("_443")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pv := &PortValue{}
			err := pv.UnmarshallJSON([]byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.expectInt != nil {
				if pv.Int == nil || *pv.Int != *tc.expectInt {
					t.Errorf("expected int %d", *tc.expectInt)
				}
			}
			if tc.expectStr != nil {
				if pv.String == nil || *pv.String != *tc.expectStr {
					t.Errorf("expected string %q", *tc.expectStr)
				}
			}
		})
	}
}

func TestPortValue_UnmarshallJSON_Invalid(t *testing.T) {
	pv := &PortValue{}
	err := pv.UnmarshallJSON([]byte(`[1,2,3]`))
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func intP(v int) *int       { return &v }
func strP(v string) *string { return &v }
