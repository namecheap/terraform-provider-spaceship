package records

import (
	"fmt"
	"strings"
)

// PTRRecord represents a PTR DNS record.
// PTR records map an IP address to its corresponding domain name in a reverse
// DNS lookup. The pointer field must be a valid domain name (1-253 chars,
// hostNameValue pattern). "@" and "*" are rejected up-front because a PTR
// target must be a real hostname, not the apex placeholder or a wildcard.
type PTRRecord struct {
	Pointer string
	Name    string
	TTL     int
}

// ValidatePointer checks that the pointer target is a valid domain name.
// "@" and "*" are rejected: a PTR target must be a real hostname, not the
// apex placeholder or a wildcard.
func (r *PTRRecord) ValidatePointer() error {
	if r.Pointer == "@" || r.Pointer == "*" || strings.HasPrefix(r.Pointer, "*") {
		return fmt.Errorf("must be a valid domain name, got %q", r.Pointer)
	}
	return ValidateName(r.Pointer)
}

// ValidateName checks that the record name is a valid hostname.
func (r *PTRRecord) ValidateName() error {
	return ValidateName(r.Name)
}

// ValidateTTL checks that the TTL is within the allowed range.
func (r *PTRRecord) ValidateTTL() error {
	return ValidateTTL(r.TTL)
}

// Validate checks all fields and returns all errors found.
func (r *PTRRecord) Validate() []error {
	var errs []error
	validators := []func() error{
		r.ValidatePointer,
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
