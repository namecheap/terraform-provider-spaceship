package client

import (
	"encoding/json"
	"fmt"
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
