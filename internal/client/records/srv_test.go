package records

import (
	"strings"
	"testing"
)

func validSRVRecord() *SRVRecord {
	return &SRVRecord{
		Name:     "myhost",
		Service:  "_sip",
		Protocol: "_tcp",
		Priority: 10,
		Weight:   60,
		Port:     5060,
		Target:   "sipserver.example.com",
		TTL:      3600,
	}
}

func TestSRVRecord_Validate_ValidRecord(t *testing.T) {
	rec := validSRVRecord()
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestSRVRecord_Validate_BoundaryValues(t *testing.T) {
	rec := &SRVRecord{
		Name:     "@",
		Service:  "_s",       // min 2 chars
		Protocol: "_u",       // min 2 chars
		Priority: 0,          // min
		Weight:   0,          // min
		Port:     1,          // min
		Target:   "a",        // min 1 char
		TTL:      60,         // min
	}
	if errs := rec.Validate(); len(errs) > 0 {
		t.Errorf("expected no errors for boundary values, got: %v", errs)
	}
}

func TestSRVRecord_ValidateService(t *testing.T) {
	tests := []struct {
		name    string
		service string
		wantErr bool
	}{
		{"valid", "_sip", false},
		{"valid with hyphen", "_my-service", false},
		{"valid min length", "_s", false},
		{"too short", "_", true},
		{"empty", "", true},
		{"missing underscore", "sip", true},
		{"too long", "_" + strings.Repeat("a", 63), true},
		{"max length", "_" + strings.Repeat("a", 62), false},
		{"invalid chars", "_sip!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validSRVRecord()
			rec.Service = tt.service
			err := rec.ValidateService()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateService(%q) error = %v, wantErr = %v", tt.service, err, tt.wantErr)
			}
		})
	}
}

func TestSRVRecord_ValidateProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		wantErr  bool
	}{
		{"valid tcp", "_tcp", false},
		{"valid udp", "_udp", false},
		{"too short", "_", true},
		{"empty", "", true},
		{"missing underscore", "tcp", true},
		{"too long", "_" + strings.Repeat("a", 63), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validSRVRecord()
			rec.Protocol = tt.protocol
			err := rec.ValidateProtocol()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProtocol(%q) error = %v, wantErr = %v", tt.protocol, err, tt.wantErr)
			}
		})
	}
}

func TestSRVRecord_ValidatePriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		wantErr  bool
	}{
		{"valid", 10, false},
		{"min valid", 0, false},
		{"max valid", 65535, false},
		{"negative", -1, true},
		{"too high", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validSRVRecord()
			rec.Priority = tt.priority
			err := rec.ValidatePriority()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePriority(%d) error = %v, wantErr = %v", tt.priority, err, tt.wantErr)
			}
		})
	}
}

func TestSRVRecord_ValidateWeight(t *testing.T) {
	tests := []struct {
		name    string
		weight  int
		wantErr bool
	}{
		{"valid", 60, false},
		{"min valid", 0, false},
		{"max valid", 65535, false},
		{"negative", -1, true},
		{"too high", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validSRVRecord()
			rec.Weight = tt.weight
			err := rec.ValidateWeight()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWeight(%d) error = %v, wantErr = %v", tt.weight, err, tt.wantErr)
			}
		})
	}
}

func TestSRVRecord_ValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid", 5060, false},
		{"min valid", 1, false},
		{"max valid", 65535, false},
		{"zero", 0, true},
		{"negative", -1, true},
		{"too high", 65536, true},
		{"wrapping value", 131073, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validSRVRecord()
			rec.Port = tt.port
			err := rec.ValidatePort()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort(%d) error = %v, wantErr = %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestSRVRecord_ValidateTarget(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{"valid hostname", "sipserver.example.com", false},
		{"valid single char", "a", false},
		{"valid apex", "@", false},
		{"valid wildcard", "*", false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 254), true},
		{"max length", strings.Repeat("a", 63) + "." + strings.Repeat("b", 63), false},
		{"starts with dot", ".invalid.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validSRVRecord()
			rec.Target = tt.target
			err := rec.ValidateTarget()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTarget(%q) error = %v, wantErr = %v", tt.target, err, tt.wantErr)
			}
		})
	}
}

func TestSRVRecord_ValidateName(t *testing.T) {
	tests := []struct {
		name     string
		recName  string
		wantErr  bool
	}{
		{"valid hostname", "myhost", false},
		{"valid apex", "@", false},
		{"valid wildcard", "*", false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 254), true},
		{"starts with dot", ".invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := validSRVRecord()
			rec.Name = tt.recName
			err := rec.ValidateName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr = %v", tt.recName, err, tt.wantErr)
			}
		})
	}
}

func TestSRVRecord_ValidateTTL(t *testing.T) {
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
			rec := validSRVRecord()
			rec.TTL = tt.ttl
			err := rec.ValidateTTL()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTTL(%d) error = %v, wantErr = %v", tt.ttl, err, tt.wantErr)
			}
		})
	}
}
