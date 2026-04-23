package records

import "fmt"

// NSRecord represents an NS DNS record.
// NS records delegate a DNS zone to the listed authoritative nameservers.
// The nameserver field must be a valid domain name (1-253 chars, hostNameValue
// pattern). "@" and "*" are rejected up-front: pointing a delegation at the
// zone apex is circular, and a wildcard nameserver is nonsensical.
type NSRecord struct {
	Nameserver string
	Name       string
	TTL        int
}

// ValidateNameserver checks that the nameserver target is a valid domain name.
// "@" and "*" are rejected: an NS target must be a real hostname, not the
// apex placeholder or a wildcard.
func (r *NSRecord) ValidateNameserver() error {
	if r.Nameserver == "@" || r.Nameserver == "*" {
		return fmt.Errorf("must be a valid domain name, got %q", r.Nameserver)
	}
	return ValidateName(r.Nameserver)
}

// ValidateName checks that the record name is a valid hostname.
func (r *NSRecord) ValidateName() error {
	return ValidateName(r.Name)
}

// ValidateTTL checks that the TTL is within the allowed range.
func (r *NSRecord) ValidateTTL() error {
	return ValidateTTL(r.TTL)
}

// Validate checks all fields and returns all errors found.
func (r *NSRecord) Validate() []error {
	var errs []error
	validators := []func() error{
		r.ValidateNameserver,
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
