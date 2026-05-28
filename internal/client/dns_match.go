package client

import (
	"fmt"
	"strings"
)

// RecordKey returns the API-identity key for a DNS record: TYPE|name|signature.
// Two records with the same key are treated as the same record by the Spaceship
// API (case-insensitive comparison, except for TXT values which are sensitive).
// This is the single source of truth for "are these the same record" across
// both the client and provider layers.
func RecordKey(record DNSRecord) string {
	return strings.ToUpper(record.Type) + "|" + strings.ToLower(record.Name) + "|" + RecordValueSignature(record)
}

// RecordValueSignature is a normalized, pipe-separated representation of the
// type-specific data fields for a record. It is the data portion of RecordKey
// and is exported because the Terraform provider stores it in its composite
// resource ID for the singular dns_record resource.
func RecordValueSignature(record DNSRecord) string {
	var builder strings.Builder
	write := func(parts ...string) {
		for _, part := range parts {
			if builder.Len() > 0 {
				builder.WriteString("|")
			}
			builder.WriteString(part)
		}
	}

	switch strings.ToUpper(record.Type) {
	case "A", "AAAA":
		write(strings.ToLower(record.Address))
	case "ALIAS":
		write(strings.ToLower(record.AliasName))
	case "CAA":
		write(intToString(record.Flag), strings.ToLower(record.Tag), strings.ToLower(record.Value))
	case "CNAME":
		write(strings.ToLower(record.CName))
	case "HTTPS":
		write(intToString(record.SvcPriority), strings.ToLower(record.TargetName), strings.ToLower(record.SvcParams), portValueSignature(record.Port), strings.ToLower(record.Scheme))
	case "MX":
		write(strings.ToLower(record.Exchange), intToString(record.Preference))
	case "NS":
		write(strings.ToLower(record.Nameserver))
	case "PTR":
		write(strings.ToLower(record.Pointer))
	case "SRV":
		write(strings.ToLower(record.Service), strings.ToLower(record.Protocol), intToString(record.Priority), intToString(record.Weight), portValueSignature(record.Port), strings.ToLower(record.Target))
	case "SVCB":
		write(intToString(record.SvcPriority), strings.ToLower(record.TargetName), strings.ToLower(record.SvcParams), portValueSignature(record.Port), strings.ToLower(record.Scheme))
	case "TLSA":
		normalized := strings.ReplaceAll(strings.ToLower(record.AssociationData), " ", "")
		write(portValueSignature(record.Port), strings.ToLower(record.Protocol), intToString(record.Usage), intToString(record.Selector), intToString(record.Matching), normalized)
	case "TXT":
		// TXT values are case-sensitive per the API.
		write(record.Value)
	default:
		write(record.Address)
	}
	return builder.String()
}

func intToString(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

// portValueSignature renders a PortValue as a normalized comparison key for
// record matching: the string form is lowercased so port labels match
// case-insensitively. This is intentionally distinct from PortValue.MarshalJSON,
// which preserves the exact wire format the API requires — do not merge them.
func portValueSignature(port *PortValue) string {
	if port == nil {
		return ""
	}
	if port.Int != nil {
		return fmt.Sprintf("%d", *port.Int)
	}
	if port.String != nil {
		return strings.ToLower(*port.String)
	}
	return ""
}
