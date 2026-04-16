package records

import "fmt"

// ALIASRecord represents an ALIAS DNS record.
// ALIAS records resolve a canonical domain name, implementing CNAME-like
// behavior for the zone apex where CNAME is not allowed.
// The aliasName field must be a valid domain name (1-253 chars, hostNameValue
// pattern). Unlike the record name field, "@" and "*" are not accepted —
// the API requires aliasName to be a real domain name.
type ALIASRecord struct {
	AliasName string
	Name      string
	TTL       int
}

// ValidateAliasName checks that the alias target is a valid domain name.
// The API rejects "@" and "*" for aliasName with a 422 "aliasName is not a
// valid domain name" error, so we catch those early.
func (r *ALIASRecord) ValidateAliasName() error {
	if r.AliasName == "@" || r.AliasName == "*" {
		return fmt.Errorf("must be a valid domain name, got %q", r.AliasName)
	}
	return ValidateName(r.AliasName)
}

// ValidateName checks that the record name is a valid hostname.
func (r *ALIASRecord) ValidateName() error {
	return ValidateName(r.Name)
}

// ValidateTTL checks that the TTL is within the allowed range.
func (r *ALIASRecord) ValidateTTL() error {
	return ValidateTTL(r.TTL)
}

// Validate checks all fields and returns all errors found.
func (r *ALIASRecord) Validate() []error {
	var errs []error
	validators := []func() error{
		r.ValidateAliasName,
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
