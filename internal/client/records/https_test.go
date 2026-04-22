package records

import (
	"strings"
	"testing"
)

func validHTTPSRecord() *HTTPSRecord {
	return &HTTPSRecord{
		SvcPriority: 1,
		TargetName:  "svc.example.com",
		SvcParams:   "alpn=h2",
		Name:        "@",
		TTL:         3600,
	}
}

func TestHTTPSRecord_Validate_ValidRecord(t *testing.T) {
	rec := validHTTPSRecord()
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestHTTPSRecord_Validate_AliasModeMinimal(t *testing.T) {
	rec := &HTTPSRecord{
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

func TestHTTPSRecord_Validate_ServiceModeWithPort(t *testing.T) {
	rec := &HTTPSRecord{
		SvcPriority: 1,
		TargetName:  "svc.example.com",
		SvcParams:   "alpn=h2 port=8443",
		Port:        "_8443",
		Scheme:      "_https",
		Name:        "@",
		TTL:         3600,
	}
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors for ServiceMode record, got: %v", errs)
	}
}

func TestHTTPSRecord_ValidateSvcPriority(t *testing.T) {
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
			rec := validHTTPSRecord()
			rec.SvcPriority = tt.priority
			err := rec.ValidateSvcPriority()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSvcPriority(%d) error = %v, wantErr = %v", tt.priority, err, tt.wantErr)
			}
		})
	}
}

func TestHTTPSRecord_ValidateTargetName(t *testing.T) {
	tests := []struct {
		name       string
		targetName string
		wantErr    bool
	}{
		{"valid dot alias", ".", false},
		{"valid FQDN", "svc.example.com", false},
		{"valid underscored FQDN", "_443._https.www.example.com", false},
		{"valid single label", "host", false},
		{"apex rejected", "@", true},
		{"wildcard rejected", "*", true},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 254), true},
		{"starts with dot", ".invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validHTTPSRecord()
			rec.TargetName = tt.targetName
			err := rec.ValidateTargetName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTargetName(%q) error = %v, wantErr = %v", tt.targetName, err, tt.wantErr)
			}
		})
	}
}

func TestHTTPSRecord_ValidateSvcParams(t *testing.T) {
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
			rec := validHTTPSRecord()
			rec.SvcParams = tt.svcParams
			err := rec.ValidateSvcParams()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSvcParams(len=%d) error = %v, wantErr = %v", len(tt.svcParams), err, tt.wantErr)
			}
		})
	}
}

func TestHTTPSRecord_ValidatePort(t *testing.T) {
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
		{"valid _60000 lower boundary", "_60000", false},
		{"valid _60009 high range", "_60009", false},
		{"valid _60999 high range", "_60999", false},
		{"valid _64999 high range", "_64999", false},
		{"valid _65530 near upper boundary", "_65530", false},
		{"invalid no underscore", "443", true},
		{"invalid zero", "_0", true},
		{"invalid too high", "_65536", true},
		{"invalid non-numeric", "_abc", true},
		{"invalid only underscore", "_", true},
		{"invalid too long", "_123456", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validHTTPSRecord()
			rec.Port = tt.port
			err := rec.ValidatePort()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort(%q) error = %v, wantErr = %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestHTTPSRecord_ValidateScheme(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		scheme  string
		wantErr bool
	}{
		{"valid _https with port", "_443", "_https", false},
		{"valid empty scheme when no port", "", "", false},
		{"valid _https even without port", "", "_https", false},
		{"missing scheme when port set", "_443", "", true},
		{"non-https scheme rejected", "_443", "_http", true},
		{"non-https scheme rejected even without port", "", "_tcp", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validHTTPSRecord()
			rec.Port = tt.port
			rec.Scheme = tt.scheme
			err := rec.ValidateScheme()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateScheme(port=%q,scheme=%q) error = %v, wantErr = %v", tt.port, tt.scheme, err, tt.wantErr)
			}
		})
	}
}

func TestHTTPSRecord_ValidateName(t *testing.T) {
	tests := []struct {
		name    string
		recName string
		wantErr bool
	}{
		{"valid hostname", "myhost", false},
		{"valid apex", "@", false},
		{"valid wildcard", "*", false},
		{"valid subdomain", "sub.domain", false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 254), true},
		{"starts with dot", ".invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validHTTPSRecord()
			rec.Name = tt.recName
			err := rec.ValidateName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr = %v", tt.recName, err, tt.wantErr)
			}
		})
	}
}

func TestHTTPSRecord_ValidateTTL(t *testing.T) {
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
			rec := validHTTPSRecord()
			rec.TTL = tt.ttl
			err := rec.ValidateTTL()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTTL(%d) error = %v, wantErr = %v", tt.ttl, err, tt.wantErr)
			}
		})
	}
}
