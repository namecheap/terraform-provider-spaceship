package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"terraform-provider-spaceship/internal/client/records"
)

// ErrPersonalNameserverNotFound is returned by FindPersonalNameserver when no
// host matching the requested name exists on the domain.
var ErrPersonalNameserverNotFound = errors.New("personal nameserver not found")

// PersonalNameserver is a single personal nameserver host and its glue IPs.
// Host is the label relative to the domain (e.g. "ns1"), which the registry
// joins with the domain to form "ns1.example.com"; IPs are the glue record
// addresses (IPv4 or IPv6) the registry serves for that host.
type PersonalNameserver struct {
	Host string   `json:"host"`
	IPs  []string `json:"ips"`
}

// PersonalNameserverList is the response envelope of the list endpoint.
type PersonalNameserverList struct {
	Records []PersonalNameserver `json:"records"`
}

// ValidateHost checks that the host is 1-255 chars and a valid hostname.
// "@" and "*" are rejected up-front: a personal nameserver host must be a real
// registry label, not the zone-apex placeholder or a wildcard — mirroring the
// other target-hostname fields (ALIAS/CNAME/MX/SRV/NS/PTR).
func (ns *PersonalNameserver) ValidateHost() error {
	if len(ns.Host) < 1 || len(ns.Host) > 255 {
		return fmt.Errorf("must be between 1 and 255 characters, got %d", len(ns.Host))
	}
	if ns.Host == "@" || ns.Host == "*" {
		return fmt.Errorf("must be a valid hostname, got %q", ns.Host)
	}
	return records.ValidateNamePattern(ns.Host)
}

// ValidateIPs checks that there are 1-16 addresses and each parses as an IP.
func (ns *PersonalNameserver) ValidateIPs() error {
	if len(ns.IPs) < 1 || len(ns.IPs) > 16 {
		return fmt.Errorf("must contain between 1 and 16 IP addresses, got %d", len(ns.IPs))
	}
	for _, ip := range ns.IPs {
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("must be a valid IP address, got %q", ip)
		}
	}
	return nil
}

// Validate checks all fields and returns all errors found.
func (ns *PersonalNameserver) Validate() []error {
	var errs []error
	validators := []func() error{
		ns.ValidateHost,
		ns.ValidateIPs,
	}
	for _, v := range validators {
		if err := v(); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// ListPersonalNameservers returns every personal nameserver host configured on
// the domain.
//
// Docs: https://docs.spaceship.dev/#tag/Personal-Nameservers
func (c *Client) ListPersonalNameservers(ctx context.Context, domain string) (PersonalNameserverList, error) {
	var list PersonalNameserverList

	endpoint := c.endpointURL([]string{"domains", domain, "personal-nameservers"}, nil)

	if _, err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &list); err != nil {
		return list, err
	}
	return list, nil
}

// FindPersonalNameserver returns the configuration for a single host.
//
// WORKAROUND: the dedicated single-host endpoint
//
//	GET /v1/domains/{domain}/personal-nameservers/{currentHost}
//
// is still under development and currently returns HTTP 501. Until it ships we
// read the working list endpoint (ListPersonalNameservers) and filter by host.
//
// TODO(api-501): when the single-host GET becomes available, this method can
// call it directly — map its response { "ips": [...] } onto a PersonalNameserver
// (the host comes from the path arg) and translate HTTP 404 into
// ErrPersonalNameserverNotFound. Weigh the switch carefully: the list+filter
// approach costs one API call per domain regardless of host count, whereas a
// per-host GET costs one call per host on every Read and may reintroduce the
// list endpoint's rate-limit pressure (5 requests / domain / 300s).
//
// Hostnames are compared case-insensitively. Returns ErrPersonalNameserverNotFound
// when no host matches.
func (c *Client) FindPersonalNameserver(ctx context.Context, domain, host string) (PersonalNameserver, error) {
	list, err := c.ListPersonalNameservers(ctx, domain)
	if err != nil {
		return PersonalNameserver{}, err
	}

	for _, ns := range list.Records {
		if strings.EqualFold(ns.Host, host) {
			return ns, nil
		}
	}
	return PersonalNameserver{}, ErrPersonalNameserverNotFound
}

// UpsertPersonalNameserver creates or updates a personal nameserver host.
//
// currentHost identifies the host in the path; ns is the desired state in the
// body. When ns.Host differs from currentHost the API renames the host (the
// old host then returns 404). For a plain create or IP-only update, pass
// currentHost equal to ns.Host.
func (c *Client) UpsertPersonalNameserver(ctx context.Context, domain, currentHost string, ns PersonalNameserver) (PersonalNameserver, error) {
	var result PersonalNameserver

	endpoint := c.endpointURL([]string{"domains", domain, "personal-nameservers", currentHost}, nil)

	if _, err := c.doJSON(ctx, http.MethodPut, endpoint, ns, &result); err != nil {
		return result, err
	}
	return result, nil
}

// DeletePersonalNameserver removes a personal nameserver host from the domain.
func (c *Client) DeletePersonalNameserver(ctx context.Context, domain, host string) error {
	endpoint := c.endpointURL([]string{"domains", domain, "personal-nameservers", host}, nil)

	if _, err := c.doJSON(ctx, http.MethodDelete, endpoint, nil, nil); err != nil {
		// A 404 means the host is already gone — the desired end state for a
		// delete — so treat it as success (mirrors DeleteDNSRecords).
		if IsNotFoundError(err) {
			return nil
		}
		return err
	}
	return nil
}
