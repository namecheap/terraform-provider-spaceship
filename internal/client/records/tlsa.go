package records

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	tlsaProtocolPattern        = regexp.MustCompile(`^_[a-zA-Z0-9-]+$`)
	tlsaAssociationDataPattern = regexp.MustCompile(`^[0-9a-fA-F]{2}(\s?[0-9a-fA-F]{2})*$`)
)

// TLSARecord represents a TLSA DNS record (RFC 6698).
// TLSA records bind a TLS server certificate or public key to the domain
// name on which the record is published, scoped by port and protocol.
type TLSARecord struct {
	Port            string
	Protocol        string
	Usage           int
	Selector        int
	Matching        int
	AssociationData string
	Name            string
	TTL             int
}

// ValidatePort checks that port is either "*" or an underscore followed by a
// port number 1-65535. Mirrors the SVCB/HTTPS port constraint.
func (r *TLSARecord) ValidatePort() error {
	if r.Port == "*" {
		return nil
	}
	if r.Port == "" || r.Port[0] != '_' {
		return fmt.Errorf("must be '*' or '_N' where N is 1-65535, got %q", r.Port)
	}
	suffix := r.Port[1:]
	for _, c := range suffix {
		if c < '0' || c > '9' {
			return fmt.Errorf("must be '*' or '_N' where N is 1-65535, got %q", r.Port)
		}
	}
	n, err := strconv.Atoi(suffix)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("must be '*' or '_N' where N is 1-65535, got %q", r.Port)
	}
	return nil
}

// ValidateProtocol checks that the protocol field starts with '_' and is
// 2-63 alphanumeric/hyphen characters (e.g. "_tcp", "_udp").
func (r *TLSARecord) ValidateProtocol() error {
	if len(r.Protocol) < 2 || len(r.Protocol) > 63 {
		return fmt.Errorf("must be between 2 and 63 characters, got %d", len(r.Protocol))
	}
	if !tlsaProtocolPattern.MatchString(r.Protocol) {
		return fmt.Errorf("must start with '_' and contain only alphanumeric characters or hyphens (e.g. '_tcp', '_udp'), got %q", r.Protocol)
	}
	return nil
}

// ValidateUsage checks that usage is within the uint8 range defined by the API (0-255).
func (r *TLSARecord) ValidateUsage() error {
	if r.Usage < 0 || r.Usage > 255 {
		return fmt.Errorf("must be between 0 and 255, got %d", r.Usage)
	}
	return nil
}

// ValidateSelector checks that selector is within the uint8 range defined by the API (0-255).
func (r *TLSARecord) ValidateSelector() error {
	if r.Selector < 0 || r.Selector > 255 {
		return fmt.Errorf("must be between 0 and 255, got %d", r.Selector)
	}
	return nil
}

// ValidateMatching checks that matching is within the uint8 range defined by the API (0-255).
func (r *TLSARecord) ValidateMatching() error {
	if r.Matching < 0 || r.Matching > 255 {
		return fmt.Errorf("must be between 0 and 255, got %d", r.Matching)
	}
	return nil
}

// ValidateAssociationData checks that the certificate association data is
// 64-65535 hex characters (any case), paired into bytes and optionally
// separated by single whitespace characters. The leading-pair anchor
// structurally rejects leading whitespace; the trailing-pair anchor
// structurally rejects trailing whitespace; `\s?` rejects double-separators
// and CRLF. Empirically aligned with the Spaceship API (verified via
// direct probe — the spec doc's lowercase-only rule is more restrictive
// than the API actually enforces).
func (r *TLSARecord) ValidateAssociationData() error {
	if len(r.AssociationData) < 64 || len(r.AssociationData) > 65535 {
		return fmt.Errorf("must be between 64 and 65535 characters, got %d", len(r.AssociationData))
	}
	if !tlsaAssociationDataPattern.MatchString(r.AssociationData) {
		return fmt.Errorf("must be hex byte pairs optionally separated by single spaces, got %q", r.AssociationData)
	}
	return nil
}

// ValidateName checks that the record name is a valid hostname.
func (r *TLSARecord) ValidateName() error {
	return ValidateName(r.Name)
}

// ValidateTTL checks that the TTL is within the allowed range.
func (r *TLSARecord) ValidateTTL() error {
	return ValidateTTL(r.TTL)
}

// Validate checks all fields and returns all errors found.
func (r *TLSARecord) Validate() []error {
	var errs []error
	validators := []func() error{
		r.ValidatePort,
		r.ValidateProtocol,
		r.ValidateUsage,
		r.ValidateSelector,
		r.ValidateMatching,
		r.ValidateAssociationData,
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
