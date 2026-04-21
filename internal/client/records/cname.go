package records

import "fmt"

// CNAMERecord represents a CNAME DNS record.
// CNAME records map an alias or subdomain to its canonical (true) domain name.
// The cname field must be a valid domain name (1-253 chars, hostNameValue
// pattern). "@" and "*" are rejected up-front because a CNAME pointing at the
// zone apex or a wildcard target is never a valid configuration.
type CNAMERecord struct {
	CName string
	Name  string
	TTL   int
}

// ValidateCName checks that the canonical-name target is a valid domain name.
// "@" and "*" are rejected: a CNAME target must be a real hostname, not the
// apex placeholder or a wildcard.
func (r *CNAMERecord) ValidateCName() error {
	if r.CName == "@" || r.CName == "*" {
		return fmt.Errorf("must be a valid domain name, got %q", r.CName)
	}
	return ValidateName(r.CName)
}

// ValidateName checks that the record name is a valid hostname.
func (r *CNAMERecord) ValidateName() error {
	return ValidateName(r.Name)
}

// ValidateTTL checks that the TTL is within the allowed range.
func (r *CNAMERecord) ValidateTTL() error {
	return ValidateTTL(r.TTL)
}

// Validate checks all fields and returns all errors found.
func (r *CNAMERecord) Validate() []error {
	var errs []error
	validators := []func() error{
		r.ValidateCName,
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
