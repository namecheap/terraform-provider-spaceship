package records

import (
	"strings"
	"testing"
)

func validSVCBRecord() *SVCBRecord {
	return &SVCBRecord{
		SvcPriority: 1,
		TargetName:  "svc.example.com",
		SvcParams:   "alpn=h2",
		Name:        "@",
		TTL:         3600,
	}
}

func TestSVCBRecord_Validate_ValidRecord(t *testing.T) {
	rec := validSVCBRecord()
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestSVCBRecord_Validate_AliasModeMinimal(t *testing.T) {
	rec := &SVCBRecord{
		SvcPriority: 0,
		TargetName:  ".",
		SvcParams:   "",
		Name:        "@",
		TTL:         60,
	}
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors for AliasMode record, got: %v", errs)
	}
}

func TestSVCBRecord_Validate_ServiceModeWithPortAndScheme(t *testing.T) {
	rec := &SVCBRecord{
		SvcPriority: 1,
		TargetName:  "svc.example.com",
		SvcParams:   "alpn=h2 port=8443",
		Port:        "_8443",
		Scheme:      "_tcp",
		Name:        "@",
		TTL:         3600,
	}
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors for ServiceMode record, got: %v", errs)
	}
}

func TestSVCBRecord_ValidateSvcPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		wantErr  bool
	}{
		{"valid zero AliasMode", 0, false},
		{"valid one ServiceMode", 1, false},
		{"valid max", 65535, false},
		{"invalid negative", -1, true},
		{"invalid too high", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validSVCBRecord()
			rec.SvcPriority = tt.priority
			err := rec.ValidateSvcPriority()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSvcPriority(%d) error = %v, wantErr = %v", tt.priority, err, tt.wantErr)
			}
		})
	}
}

func TestSVCBRecord_ValidateTargetName(t *testing.T) {
	// wantErrContains pins the rejection branch for @/* so the test fails
	// loudly if rejection ever fires from ValidateName instead of the pre-check.
	tests := []struct {
		name            string
		targetName      string
		wantErr         bool
		wantErrContains string
	}{
		{"valid dot alias", ".", false, ""},
		{"valid FQDN", "svc.example.com", false, ""},
		{"valid underscored FQDN", "_443._https.www.example.com", false, ""},
		{"valid single label", "host", false, ""},
		{"apex rejected", "@", true, "domain name"},
		{"wildcard rejected", "*", true, "domain name"},
		{"empty", "", true, ""},
		{"too long", strings.Repeat("a", 254), true, ""},
		{"starts with dot", ".invalid", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validSVCBRecord()
			rec.TargetName = tt.targetName
			err := rec.ValidateTargetName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTargetName(%q) error = %v, wantErr = %v", tt.targetName, err, tt.wantErr)
			}
			if tt.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrContains)) {
				t.Errorf("ValidateTargetName(%q) error = %v, want substring %q", tt.targetName, err, tt.wantErrContains)
			}
		})
	}
}

func TestSVCBRecord_ValidateSvcParams(t *testing.T) {
	tests := []struct {
		name      string
		svcParams string
		wantErr   bool
	}{
		{"valid empty", "", false},
		{"valid alpn", "alpn=h2,h3", false},
		{"valid multi", "alpn=h2 port=8443 no-default-alpn", false},
		{"valid max length", strings.Repeat("a", 65535), false},
		{"too long", strings.Repeat("a", 65536), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validSVCBRecord()
			rec.SvcParams = tt.svcParams
			err := rec.ValidateSvcParams()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSvcParams(len=%d) error = %v, wantErr = %v", len(tt.svcParams), err, tt.wantErr)
			}
		})
	}
}

func TestSVCBRecord_ValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr bool
	}{
		{"valid empty optional", "", false},
		{"valid wildcard", "*", false},
		{"valid _443", "_443", false},
		{"valid _1", "_1", false},
		{"valid _65535", "_65535", false},
		{"invalid no underscore", "443", true},
		{"invalid zero", "_0", true},
		{"invalid too high", "_65536", true},
		{"invalid non-numeric", "_abc", true},
		{"invalid only underscore", "_", true},
		{"invalid too long", "_123456", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validSVCBRecord()
			rec.Port = tt.port
			err := rec.ValidatePort()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort(%q) error = %v, wantErr = %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestSVCBRecord_ValidateScheme(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		scheme  string
		wantErr bool
	}{
		{"valid empty when no port", "", "", false},
		{"valid _tcp with port", "_443", "_tcp", false},
		{"valid _udp with port", "_8443", "_udp", false},
		{"valid _https without port", "", "_https", false},
		{"valid with digit", "", "_h2", false},
		{"valid with hyphen", "", "_foo-bar", false},
		{"valid max length", "", "_" + strings.Repeat("a", 62), false},
		{"missing scheme when port set", "_443", "", true},
		{"invalid missing underscore", "", "tcp", true},
		{"invalid only underscore", "", "_", true},
		{"invalid too long", "", "_" + strings.Repeat("a", 63), true},
		{"invalid illegal char", "", "_tcp!", true},
		{"invalid space", "", "_a b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validSVCBRecord()
			rec.Port = tt.port
			rec.Scheme = tt.scheme
			err := rec.ValidateScheme()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateScheme(port=%q,scheme=%q) error = %v, wantErr = %v", tt.port, tt.scheme, err, tt.wantErr)
			}
		})
	}
}
