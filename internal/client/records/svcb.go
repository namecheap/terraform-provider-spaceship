package records

import (
	"fmt"
	"regexp"
	"strconv"
)

var svcbSchemePattern = regexp.MustCompile(`^_[a-zA-Z0-9-]+$`)

// SVCBRecord represents an SVCB DNS record (RFC 9460).
// SVCB records let a service be offered from alternative endpoints, each
// with its own parameters. Unlike HTTPS, the scheme label is generic
// (e.g. "_tcp", "_udp") rather than fixed to "_https".
type SVCBRecord struct {
	SvcPriority int
	TargetName  string
	SvcParams   string
	Port        string
	Scheme      string
	Name        string
	TTL         int
}

// ValidateSvcPriority checks that svcPriority is within uint16 range.
// A value of 0 indicates AliasMode; non-zero values indicate ServiceMode.
func (r *SVCBRecord) ValidateSvcPriority() error {
	if r.SvcPriority < 0 || r.SvcPriority > 65535 {
		return fmt.Errorf("must be between 0 and 65535, got %d", r.SvcPriority)
	}
	return nil
}

// ValidateTargetName checks that targetName is either the literal "." or a
// fully qualified domain name. The API rejects "@" and "*" for targetName
// with a 422, so we catch those early.
func (r *SVCBRecord) ValidateTargetName() error {
	if r.TargetName == "." {
		return nil
	}
	if r.TargetName == "@" || r.TargetName == "*" {
		return fmt.Errorf("must be '.' or a fully qualified domain name, got %q", r.TargetName)
	}
	return ValidateName(r.TargetName)
}

// ValidateSvcParams checks the 0-65535 character length bound from the API.
// The API pattern is ".*" (any content), so no structural check is applied.
func (r *SVCBRecord) ValidateSvcParams() error {
	if len(r.SvcParams) > 65535 {
		return fmt.Errorf("must be at most 65535 characters, got %d", len(r.SvcParams))
	}
	return nil
}

// ValidatePort checks that port, when set, is either "*" or an underscore
// followed by a port number 1-65535. Port is optional for SVCB records.
func (r *SVCBRecord) ValidatePort() error {
	if r.Port == "" || r.Port == "*" {
		return nil
	}
	if r.Port[0] != '_' {
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

// ValidateScheme checks scheme format and its coupling with port. When scheme
// is set, it must start with '_' and contain only alphanumerics or '-', up to
// 63 characters. When port is set, scheme is required (empirically the
// Spaceship API returns 422 for port-without-scheme even though the spec
// marks scheme as optional). Any well-formed label is accepted: "_tcp",
// "_udp", "_mqtt", etc.
func (r *SVCBRecord) ValidateScheme() error {
	if r.Scheme == "" {
		if r.Port != "" {
			return fmt.Errorf("is required when port is set")
		}
		return nil
	}
	if len(r.Scheme) > 63 {
		return fmt.Errorf("must be at most 63 characters, got %d", len(r.Scheme))
	}
	if !svcbSchemePattern.MatchString(r.Scheme) {
		return fmt.Errorf("must start with '_' followed by alphanumerics or '-', got %q", r.Scheme)
	}
	return nil
}

// ValidateName checks that the record name is a valid hostname.
func (r *SVCBRecord) ValidateName() error {
	return ValidateName(r.Name)
}

// ValidateTTL checks that the TTL is within the allowed range.
func (r *SVCBRecord) ValidateTTL() error {
	return ValidateTTL(r.TTL)
}

// Validate checks all fields and returns all errors found.
func (r *SVCBRecord) Validate() []error {
	var errs []error
	validators := []func() error{
		r.ValidateSvcPriority,
		r.ValidateTargetName,
		r.ValidateSvcParams,
		r.ValidatePort,
		r.ValidateScheme,
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
