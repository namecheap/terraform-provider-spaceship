package records

import (
	"fmt"
	"regexp"
)

var httpsPortPattern = regexp.MustCompile(`^_(6[0-5]{2}[0-3][0-5]|[1-5]?([0-9]){2,4}|[1-9]?)$`)

// HTTPSRecord represents an HTTPS DNS record (RFC 9460).
// HTTPS records deliver configuration information for accessing a service via HTTPS.
type HTTPSRecord struct {
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
func (r *HTTPSRecord) ValidateSvcPriority() error {
	if r.SvcPriority < 0 || r.SvcPriority > 65535 {
		return fmt.Errorf("must be between 0 and 65535, got %d", r.SvcPriority)
	}
	return nil
}

// ValidateTargetName checks that targetName is either the literal "." or a
// fully qualified domain name. The API rejects "@" and "*" for targetName
// with a 422, so we catch those early.
func (r *HTTPSRecord) ValidateTargetName() error {
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
func (r *HTTPSRecord) ValidateSvcParams() error {
	if len(r.SvcParams) > 65535 {
		return fmt.Errorf("must be at most 65535 characters, got %d", len(r.SvcParams))
	}
	return nil
}

// ValidatePort checks that port, when set, is either "*" or an underscore
// followed by a port number 1-65535. Port is optional for HTTPS records.
func (r *HTTPSRecord) ValidatePort() error {
	if r.Port == "" {
		return nil
	}
	if r.Port == "*" {
		return nil
	}
	if len(r.Port) < 2 || len(r.Port) > 6 {
		return fmt.Errorf("must be '*' or an underscore followed by a port number (2-6 chars), got %d", len(r.Port))
	}
	if !httpsPortPattern.MatchString(r.Port) {
		return fmt.Errorf("must be '*' or '_N' where N is 1-65535, got %q", r.Port)
	}
	return nil
}

// ValidateScheme checks that scheme is "_https" when set. Scheme is required
// whenever port is set, and must be "_https" for HTTPS records.
func (r *HTTPSRecord) ValidateScheme() error {
	if r.Scheme == "" {
		if r.Port != "" {
			return fmt.Errorf("is required when port is set and must be '_https'")
		}
		return nil
	}
	if r.Scheme != "_https" {
		return fmt.Errorf("must be '_https', got %q", r.Scheme)
	}
	return nil
}

// ValidateName checks that the record name is a valid hostname.
func (r *HTTPSRecord) ValidateName() error {
	return ValidateName(r.Name)
}

// ValidateTTL checks that the TTL is within the allowed range.
func (r *HTTPSRecord) ValidateTTL() error {
	return ValidateTTL(r.TTL)
}

// Validate checks all fields and returns all errors found.
func (r *HTTPSRecord) Validate() []error {
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
