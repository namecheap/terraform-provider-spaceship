package client

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

var (
	_ json.Marshaler   = (*PortValue)(nil)
	_ json.Unmarshaler = (*PortValue)(nil)
)

// PortValue is the port field shared by SVCB/HTTPS/TLSA records. The Spaceship
// API is polymorphic here: it accepts and returns the port either as a JSON
// string (e.g. "_443", "*") or as a JSON number, so PortValue holds whichever
// form is present and marshals back in kind.
type PortValue struct {
	String *string
	Int    *int
}

func NewStringPortValue(value string) *PortValue {
	return &PortValue{String: &value}
}

func NewIntPortValue(value int) *PortValue {
	return &PortValue{Int: &value}
}

func (p *PortValue) MarshalJSON() ([]byte, error) {
	if p == nil {
		return []byte("null"), nil
	}
	if p.Int != nil {
		return json.Marshal(*p.Int)
	}
	if p.String != nil {
		return json.Marshal(*p.String)
	}
	return []byte("null"), nil
}

// normalizeRecordPort coerces a record's polymorphic Port into the canonical
// form for its type so signature comparison and provider-state hydration are
// stable regardless of whether the API returned the JSON port as a string or
// a number. Without this, a TLSA record sent as "_443" but echoed back as 443
// produces a different RecordKey on Read, FindDNSRecord misses, state is
// dropped, and the next plan re-creates a record the upsert API treats as
// identical — i.e. the "ID changes, server record doesn't" symptom.
//
// Canonical forms (matching what callers send and how each type's schema
// surfaces the port in the provider model):
//   - HTTPS, SVCB, TLSA: "_NNN" string (or "*" wildcard, untouched)
//   - SRV:               integer
//
// Other record types have no port and are left untouched.
func normalizeRecordPort(record *DNSRecord) {
	if record == nil || record.Port == nil {
		return
	}
	switch strings.ToUpper(record.Type) {
	case "HTTPS", "SVCB", "TLSA":
		if record.Port.Int != nil && record.Port.String == nil {
			canonical := "_" + strconv.Itoa(*record.Port.Int)
			record.Port = &PortValue{String: &canonical}
		}
	case "SRV":
		if record.Port.String != nil && record.Port.Int == nil {
			raw := strings.TrimPrefix(*record.Port.String, "_")
			if n, err := strconv.Atoi(raw); err == nil {
				record.Port = &PortValue{Int: &n}
			}
		}
	}
}

func (p *PortValue) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var intValue int
	if err := json.Unmarshal(data, &intValue); err == nil {
		p.Int = &intValue
		p.String = nil
		return nil
	}

	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err == nil {
		p.String = &stringValue
		p.Int = nil
		return nil
	}

	return fmt.Errorf("invalid port value: %s", string(data))
}
