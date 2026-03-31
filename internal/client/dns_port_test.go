package client

import (
	"encoding/json"
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

func TestPortValue_MarshalJSON(t *testing.T) {
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
			data, err := tc.pv.MarshalJSON()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(data) != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, string(data))
			}
		})
	}
}

func TestPortValue_UnmarshalJSON(t *testing.T) {
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
			err := pv.UnmarshalJSON([]byte(tc.input))
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

func TestPortValue_UnmarshalJSON_Invalid(t *testing.T) {
	pv := &PortValue{}
	err := pv.UnmarshalJSON([]byte(`[1,2,3]`))
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestPortValue_JSONRoundTrip(t *testing.T) {
	// This is the actual code path that failed: json.Unmarshal calling UnmarshalJSON
	type wrapper struct {
		Port *PortValue `json:"port,omitempty"`
	}

	tests := []struct {
		name      string
		input     string
		expectInt *int
		expectStr *string
	}{
		{"numeric port from SRV record", `{"port":5060}`, intP(5060), nil},
		{"string port from HTTPS record", `{"port":"_443"}`, nil, strP("_443")},
		{"null port", `{"port":null}`, nil, nil},
		{"absent port", `{}`, nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var w wrapper
			if err := json.Unmarshal([]byte(tc.input), &w); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}
			if tc.expectInt != nil {
				if w.Port == nil || w.Port.Int == nil || *w.Port.Int != *tc.expectInt {
					t.Errorf("expected int port %d, got %+v", *tc.expectInt, w.Port)
				}
			}
			if tc.expectStr != nil {
				if w.Port == nil || w.Port.String == nil || *w.Port.String != *tc.expectStr {
					t.Errorf("expected string port %q, got %+v", *tc.expectStr, w.Port)
				}
			}
		})
	}
}

func TestDNSRecord_UnmarshalNumericPort(t *testing.T) {
	payload := `{"items":[{"type":"SRV","name":"_autodiscover._tcp","ttl":300,"port":443,"service":"_autodiscover","protocol":"_tcp","priority":0,"weight":0,"target":"autoconfig.spacemail.com"}],"total":1}`

	var result struct {
		Items []DNSRecord `json:"items"`
		Total int         `json:"total"`
	}
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		t.Fatalf("failed to unmarshal DNS response with numeric port: %v", err)
	}
	if result.Items[0].Port == nil || result.Items[0].Port.Int == nil || *result.Items[0].Port.Int != 443 {
		t.Errorf("expected numeric port 443, got %+v", result.Items[0].Port)
	}
}

func TestDNSRecord_UnmarshalMixedPortTypes(t *testing.T) {
	payload := `{"items":[` +
		`{"type":"SRV","name":"_autodiscover._tcp","ttl":300,"port":443,"service":"_autodiscover","protocol":"_tcp","priority":0,"weight":0,"target":"autoconfig.spacemail.com"},` +
		`{"type":"TLSA","name":"_25._tcp","ttl":300,"port":"_25","protocol":"_tcp","usage":3,"selector":1,"matching":1,"associationData":"aabbccdd"}` +
		`],"total":2}`

	var result struct {
		Items []DNSRecord `json:"items"`
		Total int         `json:"total"`
	}
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		t.Fatalf("failed to unmarshal mixed port types: %v", err)
	}

	// SRV record: numeric port
	srv := result.Items[0]
	if srv.Port == nil || srv.Port.Int == nil || *srv.Port.Int != 443 {
		t.Errorf("SRV: expected numeric port 443, got %+v", srv.Port)
	}

	// TLSA record: string port
	tlsa := result.Items[1]
	if tlsa.Port == nil || tlsa.Port.String == nil || *tlsa.Port.String != "_25" {
		t.Errorf("TLSA: expected string port \"_25\", got %+v", tlsa.Port)
	}
}

func intP(v int) *int       { return &v }
func strP(v string) *string { return &v }
