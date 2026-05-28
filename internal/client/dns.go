package client

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// DNSRecord represents a DNS record managed through the Spaceship API.
type DNSRecord struct {
	//basic fields for all records
	Type string `json:"type"`
	Name string `json:"name"`
	TTL  int    `json:"ttl,omitempty"`

	//other fields for dns records
	Address         string     `json:"address,omitempty"`
	AliasName       string     `json:"aliasName,omitempty"`
	CName           string     `json:"cname,omitempty"`
	Flag            *int       `json:"flag,omitempty"`
	Tag             string     `json:"tag,omitempty"`
	Value           string     `json:"value,omitempty"`
	Port            *PortValue `json:"port,omitempty"`
	Scheme          string     `json:"scheme,omitempty"`
	SvcPriority     *int       `json:"svcPriority,omitempty"`
	TargetName      string     `json:"targetName,omitempty"`
	SvcParams       string     `json:"svcParams,omitempty"`
	Exchange        string     `json:"exchange,omitempty"`
	Preference      *int       `json:"preference,omitempty"`
	Nameserver      string     `json:"nameserver,omitempty"`
	Pointer         string     `json:"pointer,omitempty"`
	Service         string     `json:"service,omitempty"`
	Protocol        string     `json:"protocol,omitempty"`
	Priority        *int       `json:"priority,omitempty"`
	Weight          *int       `json:"weight,omitempty"`
	Target          string     `json:"target,omitempty"`
	Usage           *int       `json:"usage,omitempty"`
	Selector        *int       `json:"selector,omitempty"`
	Matching        *int       `json:"matching,omitempty"`
	AssociationData string     `json:"associationData,omitempty"`

	// Group is populated only on reads. The API returns one of three group
	// types (custom, product, personalNS); writes (PUT/DELETE) ignore it and
	// the server always assigns new records to "custom". Left nil on writes,
	// so omitempty keeps it out of request bodies. Used by
	// filterCustomDNSRecords to drop Spaceship-managed records.
	Group *RecordGroup `json:"group,omitempty"`
}

// RecordGroup is the read-only group classification the API attaches to each
// record. Type is one of: custom, product, personalNS.
type RecordGroup struct {
	Type string `json:"type"`
}

const (
	maxListPageSize     = 500
	defaultRecordsOrder = "type"

	// DNSGroupCustom is the group type for records managed via the external API.
	// Records in other groups (e.g. URL redirect) are owned by Spaceship features
	// and must not be touched by the Terraform provider.
	DNSGroupCustom = "custom"
)

// GetDNSRecords fetches custom-group DNS records for the supplied domain name. Records in other groups (product, personalNS) are filtered out.
func (c *Client) GetDNSRecords(ctx context.Context, domain string) ([]DNSRecord, error) {
	var (
		result []DNSRecord
		skip   = 0
		total  = -1
	)

	for {
		query := url.Values{}
		query.Set("take", strconv.Itoa(maxListPageSize))
		query.Set("skip", strconv.Itoa(skip))
		query.Set("orderBy", defaultRecordsOrder)

		endpoint := c.endpointURL([]string{"dns", "records", domain}, query)

		var payload struct {
			Items []DNSRecord `json:"items"`
			Total int         `json:"total"`
		}
		if _, err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &payload); err != nil {
			return nil, err
		}

		for i := range payload.Items {
			normalizeRecordPort(&payload.Items[i])
		}
		result = append(result, payload.Items...)

		if total == -1 {
			total = payload.Total
		}
		if len(result) >= total || len(payload.Items) == 0 {
			break
		}

		skip += maxListPageSize

	}
	return filterCustomDNSRecords(result), nil
}

var ErrRecordNotFound = errors.New("record not found")

// FindDNSRecord locates a single record in the domain's custom group by its
// API identity: type, name, and the type-specific data signature (the same
// signature returned by RecordValueSignature). Matching uses RecordKey, which
// applies the API's case rules (TYPE upper, name lower, type-specific lowering).
// Returns ErrRecordNotFound when no record matches.
func (c *Client) FindDNSRecord(ctx context.Context, domain, recordType, name, signature string) (DNSRecord, error) {
	records, err := c.GetDNSRecords(ctx, domain)
	if err != nil {
		return DNSRecord{}, err
	}

	target := strings.ToUpper(recordType) + "|" + strings.ToLower(name) + "|" + signature
	for _, record := range records {
		if RecordKey(record) == target {
			return record, nil
		}
	}

	return DNSRecord{}, ErrRecordNotFound

}

// filterCustomDNSRecords returns only records whose group type is "custom"
// (or that have no group at all, for backwards compatibility with ungrouped zones).
// Records belonging to Spaceship-managed groups (e.g. URL redirect) are excluded
// so the provider does not attempt to manage them.
func filterCustomDNSRecords(records []DNSRecord) []DNSRecord {
	filtered := make([]DNSRecord, 0, len(records))
	for _, r := range records {
		if r.Group == nil || r.Group.Type == DNSGroupCustom {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// UpsertDNSRecords creates or updated DNS records for the supplied domain
func (c *Client) UpsertDNSRecords(ctx context.Context, domain string, force bool, records []DNSRecord) error {
	if len(records) == 0 {
		return nil
	}

	endpoint := c.endpointURL([]string{"dns", "records", domain}, nil)

	payload := struct {
		Force bool        `json:"force"`
		Items []DNSRecord `json:"items"`
	}{
		Force: force,
		Items: records,
	}

	if _, err := c.doJSON(ctx, http.MethodPut, endpoint, payload, nil); err != nil {
		return err
	}
	return nil
}

// CreateDNSRecord adds a single custom DNS record to the domain.
//
// This calls the upsert endpoint (PUT /dns/records/{domain}) and is therefore
// **idempotent for records with matching (type, name, data)**: a call against
// an already-existing identical record is a no-op (or TTL update), not an
// error.
//
// Conflict cases (e.g. CNAME with a different target at the same hostname)
// still return 422 because they violate API-level uniqueness constraints.
func (c *Client) CreateDNSRecord(ctx context.Context, domain string, record DNSRecord) error {
	endpoint := c.endpointURL([]string{"dns", "records", domain}, nil)

	records := []DNSRecord{record}

	payload := struct {
		Items []DNSRecord `json:"items"`
	}{
		Items: records,
	}

	if _, err := c.doJSON(ctx, http.MethodPut, endpoint, payload, nil); err != nil {
		return err
	}
	return nil
}

func (c *Client) DeleteDNSRecord(ctx context.Context, domain string, record DNSRecord) error {
	records := []DNSRecord{record}

	err := c.DeleteDNSRecords(ctx, domain, records)
	if err != nil {
		return err
	}
	return nil

}

// DeleteDNSRecords removes the specified DNS records.
func (c *Client) DeleteDNSRecords(ctx context.Context, domain string, records []DNSRecord) error {
	if len(records) == 0 {
		return nil
	}

	endpoint := c.endpointURL([]string{"dns", "records", domain}, nil)

	if _, err := c.doJSON(ctx, http.MethodDelete, endpoint, records, nil); err != nil {
		if IsNotFoundError(err) {
			return nil
		}
		return err
	}
	return nil
}

// ClearDNSRecords removes all custom-group DNS records for the domain. Records in other groups (product, personalNS) are not affected.
func (c *Client) ClearDNSRecords(ctx context.Context, domain string, force bool) error {
	records, err := c.GetDNSRecords(ctx, domain)
	if err != nil {
		if IsNotFoundError(err) {
			return nil
		}

		return err
	}

	return c.DeleteDNSRecords(ctx, domain, records)
}
