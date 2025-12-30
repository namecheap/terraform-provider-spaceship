package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// DNSRecord represents a DNS record managed through the Spaceship API.
type DNSRecord struct {
	Type            string       `json:"type"`
	Name            string       `json:"name"`
	TTL             int          `json:"ttl,omitempty"`
	Address         string       `json:"address,omitempty"`
	AliasName       string       `json:"aliasName,omitempty"`
	CName           string       `json:"cname,omitempty"`
	Flag            *int         `json:"flag,omitempty"`
	Tag             string       `json:"tag,omitempty"`
	Value           string       `json:"value,omitempty"`
	Port            *PortValue   `json:"port,omitempty"`
	Scheme          string       `json:"scheme,omitempty"`
	SvcPriority     *int         `json:"svcPriority,omitempty"`
	TargetName      string       `json:"targetName,omitempty"`
	SvcParams       string       `json:"svcParams,omitempty"`
	Exchange        string       `json:"exchange,omitempty"`
	Preference      *int         `json:"preference,omitempty"`
	Nameserver      string       `json:"nameserver,omitempty"`
	Pointer         string       `json:"pointer,omitempty"`
	Service         string       `json:"service,omitempty"`
	Protocol        string       `json:"protocol,omitempty"`
	Priority        *int         `json:"priority,omitempty"`
	Weight          *int         `json:"weight,omitempty"`
	Target          string       `json:"target,omitempty"`
	Usage           *int         `json:"usage,omitempty"`
	Selector        *int         `json:"selector,omitempty"`
	Matching        *int         `json:"matching,omitempty"`
	AssociationData string       `json:"associationData,omitempty"`
	Group           *RecordGroup `json:"group,omitempty"`
}

type RecordGroup struct {
	Type string `json:"type"`
}

type PortValue struct {
	String *string
	Int    *int
}

func NewStringPortValue(value string) *PortValue {
	return &PortValue{String: &value}
}

func NewIntPortValue(value int) *PortValue {
	return &PortValue{Int: &value}
}

func (p *PortValue) MarshallJSON() ([]byte, error) {
	if p == nil {
		return []byte("null"), nil
	}
	if p.Int != nil {
		return json.Marshal(*p.Int)
	}
	if p.String != nil {
		return json.Marshal(*p.String)
	}
	return []byte("null"), nil
}

func (p *PortValue) UnmarshallJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var intValue int
	if err := json.Unmarshal(data, &intValue); err == nil {
		p.Int = &intValue
		p.String = nil
		return nil
	}

	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err == nil {
		p.String = &stringValue
		p.Int = nil
		return nil
	}

	return fmt.Errorf("invalid port value: %s", string(data))
}

const (
	maxListPageSize     = 500
	defaultRecordsOrder = "type"
)

// fetches DNS records for the supplied domain name.
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

		endpoint := fmt.Sprintf("%s/dns/records/%s?%s", c.baseURL, url.PathEscape(domain), query.Encode())

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("create reqeust %w", err)
		}
		c.applyAuth(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("execute reqeust: %w", err)
		}

		var payload struct {
			Items []DNSRecord `json:"items"`
			Total int         `json:"total"`
		}

		func() {
			defer resp.Body.Close()

			if resp.StatusCode >= 300 {
				err = c.errorFromResponse(resp)
				return
			}

			if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr != nil {
				err = fmt.Errorf("decode response: %w", decodeErr)
				return
			}
		}()

		if err != nil {
			return nil, err
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
	return result, nil
}

// UpsertDNSRecords creates or updated DNS records for the supplied domain
func (c *Client) UpsertDNSRecords(ctx context.Context, domain string, force bool, records []DNSRecord) error {
	if len(records) == 0 {
		return nil
	}

	endpoint := fmt.Sprintf("%s/dns/records/%s", c.baseURL, url.PathEscape((domain)))

	payload := struct {
		Force bool        `json:"force"` // where it comes from?
		Items []DNSRecord `json:"items"`
	}{
		Force: force,
		Items: records,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request %w", err)
	}

	c.applyAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("execute reqeust: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return c.errorFromResponse(resp)
	}
	return nil
}


// DeleteDNSRecords removed the specified DNS records.
func (c *Client) DeleteDNSRecords(ctx context.Context, domain string, records []DNSRecord) error {
	if len(records) == 0 {
		return nil
	}

	endpoint := fmt.Sprintf("%s/dns/records/%s", c.baseURL, url.PathEscape(domain))

	body, err := json.Marshal(records)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.applyAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("execute request:%w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		err := c.errorFromResponse(resp)
		if IsNotFoundError(err) {
			return nil
		}

		return err
	}

	return nil
}

// ClearDNSRecords removed all DNS records that aree managed through Terraform for the domain
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