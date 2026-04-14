package records

import (
	"fmt"
	"net"
	"strings"
)

type ARecord struct {
	Address string
	Name    string
	TTL     int
}

// ValidateAddress checks that the address is a valid IPv4 address.
func (r *ARecord) ValidateAddress() error {
	ip := net.ParseIP(r.Address)
	if ip == nil || strings.Contains(r.Address, ":") {
		return fmt.Errorf("must be a valid IPv4 address, got %q", r.Address)
	}
	return nil
}

// ValidateName checks that the record name is a valid hostname.
func (r *ARecord) ValidateName() error {
	return ValidateName(r.Name)
}

// ValidateTTL checks that the TTL is within the allowed range.
func (r *ARecord) ValidateTTL() error {
	return ValidateTTL(r.TTL)
}

// Validate checks all fields and returns all errors found.
func (r *ARecord) Validate() []error {
	var errs []error
	validators := []func() error{
		r.ValidateAddress,
		r.ValidateName,
		r.ValidateTTL,
	}
	for _, v := range validators {
		if err := v(); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
