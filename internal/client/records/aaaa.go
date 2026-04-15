package records

import (
	"fmt"
	"net"
)

type AAAARecord struct {
	Address string
	Name    string
	TTL     int
}

// ValidateAddress checks that the address matches Spaceship's confirmed
// AAAA address rules:
//   - length: <= 39 characters (API cap; matches a fully expanded canonical
//     IPv6 address: 8 groups * 4 hex + 7 colons)
//   - format: must parse as IPv6 and must not be IPv4 or IPv4-mapped IPv6.
//     The API rejects both of those forms with a 422 response:
//     "Field address is not valid ipv6 address"
func (r *AAAARecord) ValidateAddress() error {
	if len(r.Address) > 39 {
		return fmt.Errorf("must be at most 39 characters, got %d", len(r.Address))
	}
	ip := net.ParseIP(r.Address)
	if ip == nil || ip.To4() != nil {
		return fmt.Errorf("must be a valid IPv6 address, got %q", r.Address)
	}
	return nil
}

// ValidateName checks that the record name is a valid hostname.
func (r *AAAARecord) ValidateName() error {
	return ValidateName(r.Name)
}

// ValidateTTL checks that the TTL is within the allowed range.
func (r *AAAARecord) ValidateTTL() error {
	return ValidateTTL(r.TTL)
}

// Validate checks all fields and returns all errors found.
func (r *AAAARecord) Validate() []error {
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
