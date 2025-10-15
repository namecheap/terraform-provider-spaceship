package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client provides a thin wrapper around Spaceship API
type Client struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// TODO
// need to be changed not all dns records has these fields
// would work only for A and AAAA records
// need something where all these are defined maybe?
type DNSRecord struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	TTL     int    `json:"ttl"`
	Address string `json:"address"`
}

// represents an error response from the spaceship api
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Message != "" {
		return fmt.Sprintf("spaceship api error (status %d): %s", e.Status, e.Message)
	}

	return fmt.Sprintf("spaceship api error (status %d)", e.Status)
}

// created a new Spaceship API Client
func NewClient(baseURL, apiKey, apiSecret string) *Client {
	return &Client{
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		apiKey:    apiKey,
		apiSecret: apiSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

const maxListPageSize = 500

// fetches DNS records for the supplied domain name.
func (c *Client) GetDNSRecords(ctx context.Context, domain string) ([]DNSRecord, error) {
	var (
		result []DNSRecord
		skip   = 0
		total  = -1
	)

	for {
		endpoint := fmt.Sprintf("%s/dns/records/%s?take=%d&skip=%d&orderBy=name", c.baseURL, url.PathEscape(domain), maxListPageSize, skip)

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
				err = fmt.Errorf("decode response %w", &decodeErr)
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

func (c *Client) applyAuth(req *http.Request) {
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-API-Secret", c.apiSecret)
}

func (c *Client) errorFromResponse(resp *http.Response) error {
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return &APIError{
			Status: resp.StatusCode,
		}
	}

	return &APIError{
		Status:  resp.StatusCode,
		Message: strings.TrimSpace(string(data)),
	}
}

// returns true if the err represents 404 response
// todo why it is needed?
func IsNotFoundError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Status == http.StatusNotFound
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
// TODO
// WHY this is needed?
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
