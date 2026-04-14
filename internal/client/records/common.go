package records

import (
	"fmt"

	"github.com/dlclark/regexp2"
)

var hostnamePattern = regexp2.MustCompile(`^(?!\.)(@|\*|([_*]\.)?(?:(?!-)(?=[^\.]*[^\W_])[\w-]{1,63}(?<!-)($|\.)){1,127}(?<!\.))$`, regexp2.None)

// ValidateName checks that a record name is 1-253 chars and matches the hostname pattern.
func ValidateName(name string) error {
	if len(name) < 1 || len(name) > 253 {
		return fmt.Errorf("must be between 1 and 253 characters, got %d", len(name))
	}
	matched, err := hostnamePattern.MatchString(name)
	if err != nil {
		return fmt.Errorf("hostname pattern match failed: %w", err)
	}
	if !matched {
		return fmt.Errorf("must be a valid hostname (e.g. 'myhost' or '@'), got %q", name)
	}
	return nil
}

// ValidateTTL checks that a TTL is in the range 60-3600.
func ValidateTTL(ttl int) error {
	if ttl < 60 || ttl > 3600 {
		return fmt.Errorf("must be between 60 and 3600, got %d", ttl)
	}
	return nil
}
