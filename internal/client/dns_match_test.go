package client

import "testing"

func intPtr(v int) *int {
	return &v
}

func TestRecordValueSignature_AllTypes(t *testing.T) {
	tests := []struct {
		name   string
		record DNSRecord
	}{
		{"A", DNSRecord{Type: "A", Name: "@", Address: "1.2.3.4"}},
		{"AAAA", DNSRecord{Type: "AAAA", Name: "@", Address: "2001:db8::1"}},
		{"ALIAS", DNSRecord{Type: "ALIAS", Name: "@", AliasName: "other.com"}},
		{"CAA", DNSRecord{Type: "CAA", Name: "@", Flag: intPtr(0), Tag: "issue", Value: "letsencrypt.org"}},
		{"CNAME", DNSRecord{Type: "CNAME", Name: "www", CName: "example.com"}},
		{"HTTPS", DNSRecord{Type: "HTTPS", Name: "@", SvcPriority: intPtr(1), TargetName: "target.com", SvcParams: "alpn=h2", Port: NewStringPortValue("_443"), Scheme: "_https"}},
		{"MX", DNSRecord{Type: "MX", Name: "@", Exchange: "mail.example.com", Preference: intPtr(10)}},
		{"NS", DNSRecord{Type: "NS", Name: "@", Nameserver: "ns1.example.com"}},
		{"PTR", DNSRecord{Type: "PTR", Name: "1", Pointer: "host.example.com"}},
		{"SRV", DNSRecord{Type: "SRV", Name: "_sip._tcp", Service: "_sip", Protocol: "_tcp", Priority: intPtr(5), Weight: intPtr(10)}},
		{"SVCB", DNSRecord{Type: "SVCB", Name: "@", SvcPriority: intPtr(1), TargetName: "svc.com", SvcParams: "alpn=h2", Port: NewStringPortValue("_853"), Scheme: "_dot"}},
		{"TXT", DNSRecord{Type: "TXT", Name: "@", Value: "v=spf1"}},
		{"unknown", DNSRecord{Type: "UNKNOWN", Name: "@", Address: "fallback"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sig := RecordValueSignature(tc.record)
			if sig == "" && tc.name != "unknown" {
				t.Error("expected non-empty signature")
			}
			if sig != RecordValueSignature(tc.record) {
				t.Error("signature should be deterministic")
			}
		})
	}
}

func TestRecordValueSignature_CAAValueCaseInsensitive(t *testing.T) {
	r1 := DNSRecord{Type: "CAA", Name: "@", Flag: intPtr(0), Tag: "issue", Value: "LetsEncrypt.org"}
	r2 := DNSRecord{Type: "CAA", Name: "@", Flag: intPtr(0), Tag: "issue", Value: "letsencrypt.org"}
	if RecordValueSignature(r1) != RecordValueSignature(r2) {
		t.Error("expected case-insensitive match for CAA value")
	}
}

func TestRecordValueSignature_SvcParamsCaseInsensitive(t *testing.T) {
	r1 := DNSRecord{Type: "HTTPS", Name: "@", SvcPriority: intPtr(1), TargetName: "target.com", SvcParams: "ALPN=H2", Port: NewStringPortValue("_443"), Scheme: "_https"}
	r2 := DNSRecord{Type: "HTTPS", Name: "@", SvcPriority: intPtr(1), TargetName: "target.com", SvcParams: "alpn=h2", Port: NewStringPortValue("_443"), Scheme: "_https"}
	if RecordValueSignature(r1) != RecordValueSignature(r2) {
		t.Error("expected case-insensitive match for HTTPS SvcParams")
	}

	r3 := DNSRecord{Type: "SVCB", Name: "@", SvcPriority: intPtr(1), TargetName: "svc.com", SvcParams: "ALPN=H2", Port: NewStringPortValue("_853"), Scheme: "_dot"}
	r4 := DNSRecord{Type: "SVCB", Name: "@", SvcPriority: intPtr(1), TargetName: "svc.com", SvcParams: "alpn=h2", Port: NewStringPortValue("_853"), Scheme: "_dot"}
	if RecordValueSignature(r3) != RecordValueSignature(r4) {
		t.Error("expected case-insensitive match for SVCB SvcParams")
	}
}

func TestRecordValueSignature_TXTValueCaseSensitive(t *testing.T) {
	r1 := DNSRecord{Type: "TXT", Name: "@", Value: "v=DKIM1; p=ABC"}
	r2 := DNSRecord{Type: "TXT", Name: "@", Value: "v=dkim1; p=abc"}
	if RecordValueSignature(r1) == RecordValueSignature(r2) {
		t.Error("expected case-sensitive match for TXT value")
	}
}

func TestRecordValueSignature_CaseInsensitive(t *testing.T) {
	r1 := DNSRecord{Type: "A", Name: "@", Address: "1.2.3.4"}
	r2 := DNSRecord{Type: "a", Name: "@", Address: "1.2.3.4"}
	if RecordValueSignature(r1) != RecordValueSignature(r2) {
		t.Error("expected case-insensitive match for A records")
	}
}

func TestPortValueSignature(t *testing.T) {
	tests := []struct {
		name     string
		port     *PortValue
		expected string
	}{
		{"nil", nil, ""},
		{"int", NewIntPortValue(443), "443"},
		{"string", NewStringPortValue("_443"), "_443"},
		{"empty", &PortValue{}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := portValueSignature(tc.port)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestIntToString(t *testing.T) {
	if intToString(nil) != "" {
		t.Error("expected empty for nil")
	}
	v := 42
	if intToString(&v) != "42" {
		t.Errorf("expected 42, got %q", intToString(&v))
	}
}

func TestRecordKey(t *testing.T) {
	r := DNSRecord{Type: "A", Name: "test", Address: "1.2.3.4"}
	key := RecordKey(r)
	if key == "" {
		t.Error("expected non-empty key")
	}
	if key != RecordKey(r) {
		t.Error("expected deterministic key")
	}
	r2 := DNSRecord{Type: "a", Name: "TEST", Address: "1.2.3.4"}
	if key != RecordKey(r2) {
		t.Error("expected case-insensitive key match")
	}
}
