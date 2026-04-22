package records

import (
	"fmt"
	"regexp"
)

var srvServicePattern = regexp.MustCompile(`^_[a-zA-Z0-9-]+$`)

type SRVRecord struct {
	Service  string
	Protocol string
	Priority int
	Weight   int
	Port     int
	Target   string
	Name     string
	TTL      int
}

// ValidateService checks that the service field starts with '_' and is 2-63 alphanumeric/hyphen characters.
func (r *SRVRecord) ValidateService() error {
	if len(r.Service) < 2 || len(r.Service) > 63 {
		return fmt.Errorf("must be between 2 and 63 characters, got %d", len(r.Service))
	}
	if !srvServicePattern.MatchString(r.Service) {
		return fmt.Errorf("must start with '_' and contain only alphanumeric characters or hyphens (e.g. '_sip', '_ldap'), got %q", r.Service)
	}
	return nil
}

// ValidateProtocol checks that the protocol field starts with '_' and is 2-63 alphanumeric/hyphen characters.
func (r *SRVRecord) ValidateProtocol() error {
	if len(r.Protocol) < 2 || len(r.Protocol) > 63 {
		return fmt.Errorf("must be between 2 and 63 characters, got %d", len(r.Protocol))
	}
	if !srvServicePattern.MatchString(r.Protocol) {
		return fmt.Errorf("must start with '_' and contain only alphanumeric characters or hyphens (e.g. '_tcp', '_udp'), got %q", r.Protocol)
	}
	return nil
}

// ValidatePriority checks that the priority is between 0 and 65535.
func (r *SRVRecord) ValidatePriority() error {
	if r.Priority < 0 || r.Priority > 65535 {
		return fmt.Errorf("must be between 0 and 65535, got %d", r.Priority)
	}
	return nil
}

// ValidateWeight checks that the weight is between 0 and 65535.
func (r *SRVRecord) ValidateWeight() error {
	if r.Weight < 0 || r.Weight > 65535 {
		return fmt.Errorf("must be between 0 and 65535, got %d", r.Weight)
	}
	return nil
}

// ValidatePort checks that the port is between 1 and 65535.
func (r *SRVRecord) ValidatePort() error {
	if r.Port < 1 || r.Port > 65535 {
		return fmt.Errorf("must be between 1 and 65535, got %d", r.Port)
	}
	return nil
}

// ValidateTarget checks that the target is a valid domain name.
// "@" and "*" are rejected: runtime API returns 422 "target is not a valid
// domain name" for those values (empirically confirmed), even though the
// hostNameValue schema in the API docs formally admits them.
func (r *SRVRecord) ValidateTarget() error {
	if r.Target == "@" || r.Target == "*" {
		return fmt.Errorf("must be a valid domain name, got %q", r.Target)
	}
	if len(r.Target) < 1 || len(r.Target) > 253 {
		return fmt.Errorf("must be between 1 and 253 characters, got %d", len(r.Target))
	}
	matched, err := hostnamePattern.MatchString(r.Target)
	if err != nil {
		return fmt.Errorf("hostname pattern match failed: %w", err)
	}
	if !matched {
		return fmt.Errorf("must be a valid hostname (e.g. 'server.example.com'), got %q", r.Target)
	}
	return nil
}

// ValidateName checks that the record name is a valid hostname.
func (r *SRVRecord) ValidateName() error {
	return ValidateName(r.Name)
}

// ValidateTTL checks that the TTL is within the allowed range.
func (r *SRVRecord) ValidateTTL() error {
	return ValidateTTL(r.TTL)
}

// Validate checks all fields and returns all errors found.
func (r *SRVRecord) Validate() []error {
	var errs []error
	validators := []func() error{
		r.ValidateService,
		r.ValidateProtocol,
		r.ValidatePriority,
		r.ValidateWeight,
		r.ValidatePort,
		r.ValidateTarget,
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
