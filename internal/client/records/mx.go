package records

import "fmt"

// MXRecord represents an MX DNS record.
// MX records specify the mail servers responsible for receiving email
// for a domain. Each record pairs an exchange host with a preference —
// lower preference values are tried first. The exchange field must be a
// real domain name: the API returns 422 for "@" and "*" even though the
// hostNameValue schema formally admits them, so we reject those early.
type MXRecord struct {
	Exchange   string
	Preference int
	Name       string
	TTL        int
}

// ValidateExchange checks that the exchange is a valid domain name.
// "@" and "*" are rejected: runtime API returns 422 for those values on
// hostname-target fields (empirically confirmed for ALIAS, CNAME, and MX).
func (r *MXRecord) ValidateExchange() error {
	if r.Exchange == "@" || r.Exchange == "*" {
		return fmt.Errorf("must be a valid domain name, got %q", r.Exchange)
	}
	return ValidateName(r.Exchange)
}

// ValidatePreference checks that the preference is a uint16 value (0-65535).
// The field is stored as int so out-of-range values surface here as
// validation errors rather than being silently truncated.
func (r *MXRecord) ValidatePreference() error {
	if r.Preference < 0 || r.Preference > 65535 {
		return fmt.Errorf("must be between 0 and 65535, got %d", r.Preference)
	}
	return nil
}

// ValidateName checks that the record name is a valid hostname.
func (r *MXRecord) ValidateName() error {
	return ValidateName(r.Name)
}

// ValidateTTL checks that the TTL is within the allowed range.
func (r *MXRecord) ValidateTTL() error {
	return ValidateTTL(r.TTL)
}

// Validate checks all fields and returns all errors found.
func (r *MXRecord) Validate() []error {
	var errs []error
	validators := []func() error{
		r.ValidateExchange,
		r.ValidatePreference,
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
