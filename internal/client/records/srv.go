package records

import (
	"fmt"
	"regexp"

	"github.com/dlclark/regexp2"
)

var (
	srvServicePattern  = regexp.MustCompile(`^_[a-zA-Z0-9-]+$`)
	srvHostnamePattern = regexp2.MustCompile(`^(?!\.)(@|\*|([_*]\.)?(?:(?!-)(?=[^\.]*[^\W_])[\w-]{1,63}(?<!-)($|\.)){1,127}(?<!\.))$`, regexp2.None)
)

type SRVRecord struct {
	Type     string
	Service  string
	Protocol string
	Priority uint16
	Weight   uint16
	Port     uint16
	Target   string
	Name     string
	TTL      int
}

func (r *SRVRecord) ValidateService() error {
	if len(r.Service) < 2 || len(r.Service) > 63 {
		return fmt.Errorf("must be between 2 and 63 characters, got %d", len(r.Service))
	}
	if !srvServicePattern.MatchString(r.Service) {
		return fmt.Errorf("must start with '_' and contain only alphanumeric characters or hyphens (e.g. '_sip', '_ldap'), got %q", r.Service)
	}
	return nil
}

func (r *SRVRecord) ValidateProtocol() error {
	if len(r.Protocol) < 2 || len(r.Protocol) > 63 {
		return fmt.Errorf("must be between 2 and 63 characters, got %d", len(r.Protocol))
	}
	if !srvServicePattern.MatchString(r.Protocol) {
		return fmt.Errorf("must start with '_' and contain only alphanumeric characters or hyphens (e.g. '_tcp', '_udp'), got %q", r.Protocol)
	}
	return nil
}

func (r *SRVRecord) ValidatePort() error {
	if r.Port < 1 {
		return fmt.Errorf("must be between 1 and 65535, got %d", r.Port)
	}
	return nil
}

func (r *SRVRecord) ValidateTarget() error {
	if len(r.Target) < 1 || len(r.Target) > 253 {
		return fmt.Errorf("must be between 1 and 253 characters, got %d", len(r.Target))
	}
	if matched, _ := srvHostnamePattern.MatchString(r.Target); !matched {
		return fmt.Errorf("must be a valid hostname (e.g. 'server.example.com' or '@'), got %q", r.Target)
	}
	return nil
}

func (r *SRVRecord) ValidateName() error {
	if len(r.Name) < 1 || len(r.Name) > 253 {
		return fmt.Errorf("must be between 1 and 253 characters, got %d", len(r.Name))
	}
	if matched, _ := srvHostnamePattern.MatchString(r.Name); !matched {
		return fmt.Errorf("must be a valid hostname (e.g. 'myhost' or '@'), got %q", r.Name)
	}
	return nil
}

func (r *SRVRecord) ValidateTTL() error {
	if r.TTL < 60 || r.TTL > 3600 {
		return fmt.Errorf("must be between 60 and 3600, got %d", r.TTL)
	}
	return nil
}

// Validate checks all fields and returns all errors found.
func (r *SRVRecord) Validate() []error {
	var errs []error
	validators := []func() error{
		r.ValidateService,
		r.ValidateProtocol,
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
