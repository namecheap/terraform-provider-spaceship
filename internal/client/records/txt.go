package records

import (
	"fmt"
	"strings"
)

// TXTRecord represents a TXT DNS record.
// TXT records store arbitrary text data associated with a domain — commonly
// used for SPF, DKIM, domain-ownership verification, and other text-based
// metadata.
type TXTRecord struct {
	Value string
	Name  string
	TTL   int
}

// ValidateValue checks that the value is 1-65535 bytes long and is not
// whitespace-only. The spec lists minLength=1, maxLength=65535,
// pattern=".*", but live-API probing showed the server also rejects
// whitespace-only values (empty, space, tab, LF, CR, CRLF) with
// "Value field is required". The length cap is counted in bytes, not
// characters; len() of a Go string is already bytes.
func (r *TXTRecord) ValidateValue() error {
	if len(r.Value) < 1 || len(r.Value) > 65535 {
		return fmt.Errorf("must be between 1 and 65535 bytes, got %d", len(r.Value))
	}
	if strings.TrimSpace(r.Value) == "" {
		return fmt.Errorf("must not be whitespace-only")
	}
	return nil
}

// ValidateName checks that the record name is a valid hostname.
func (r *TXTRecord) ValidateName() error {
	return ValidateName(r.Name)
}

// ValidateTTL checks that the TTL is within the allowed range.
func (r *TXTRecord) ValidateTTL() error {
	return ValidateTTL(r.TTL)
}

// Validate checks all fields and returns all errors found.
func (r *TXTRecord) Validate() []error {
	var errs []error
	validators := []func() error{
		r.ValidateValue,
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
