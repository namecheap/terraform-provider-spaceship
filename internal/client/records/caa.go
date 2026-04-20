package records

import (
	"fmt"
	"regexp"
)

var caaValuePattern = regexp.MustCompile(`^[ -~]+$`)

// CAARecord represents a CAA DNS record.
// CAA records let a domain owner authorize specific Certificate Authorities
// to issue certificates for the domain.
type CAARecord struct {
	Flag  int
	Tag   string
	Value string
	Name  string
	TTL   int
}

// ValidateFlag checks that the flag is 0 or 128, the only values defined by
// the API (0 = no flags, 128 = critical bit set).
func (r *CAARecord) ValidateFlag() error {
	if r.Flag != 0 && r.Flag != 128 {
		return fmt.Errorf("must be 0 or 128, got %d", r.Flag)
	}
	return nil
}

// ValidateTag checks that the tag is one of the three CAA property tags
// defined by the API: issue, issuewild, iodef.
func (r *CAARecord) ValidateTag() error {
	switch r.Tag {
	case "issue", "issuewild", "iodef":
		return nil
	default:
		return fmt.Errorf("must be one of 'issue', 'issuewild', 'iodef', got %q", r.Tag)
	}
}

// ValidateValue checks that the value is 1-256 ASCII-printable characters,
// matching the API's pattern ^[ -~]+$ and maxLength 256.
func (r *CAARecord) ValidateValue() error {
	if len(r.Value) < 1 || len(r.Value) > 256 {
		return fmt.Errorf("must be between 1 and 256 characters, got %d", len(r.Value))
	}
	if !caaValuePattern.MatchString(r.Value) {
		return fmt.Errorf("must contain only printable ASCII characters, got %q", r.Value)
	}
	return nil
}

// ValidateName checks that the record name is a valid hostname.
func (r *CAARecord) ValidateName() error {
	return ValidateName(r.Name)
}

// ValidateTTL checks that the TTL is within the allowed range.
func (r *CAARecord) ValidateTTL() error {
	return ValidateTTL(r.TTL)
}

// Validate checks all fields and returns all errors found.
func (r *CAARecord) Validate() []error {
	var errs []error
	validators := []func() error{
		r.ValidateFlag,
		r.ValidateTag,
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
