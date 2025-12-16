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
	"strconv"
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

type DomainList struct {
	Items []DomainInfo `json:"items"`
	Total int64        `json:"total"`
}
type DomainInfo struct {
	Name               string            `json:"name"`
	UnicodeName        string            `json:"unicodeName"`
	IsPremium          bool              `json:"isPremium"`
	AutoRenew          bool              `json:"autoRenew"`
	RegistrationDate   string            `json:"registrationDate"`
	ExpirationDate     string            `json:"expirationDate"`
	LifecycleStatus    string            `json:"lifecycleStatus"`
	VerificationStatus string            `json:"verificationStatus"`
	EPPStatuses        []string          `json:"eppStatuses"`
	Suspensions        []ReasonCode      `json:"suspensions"`
	PrivacyProtection  PrivacyProtection `json:"privacyProtection"`
	Nameservers        Nameservers       `json:"nameservers"`
	Contacts           Contacts          `json:"contacts"`
}

type ReasonCode struct {
	ReasonCode string `json:"reasonCode"`
}

type PrivacyProtection struct {
	ContactForm bool   `json:"contactForm"`
	Level       string `json:"level"`
}

type Nameservers struct {
	Provider string   `json:"provider"`
	Hosts    []string `json:"hosts"`
}

type Contacts struct {
	Registrant string   `json:"registrant"`
	Admin      string   `json:"admin"`
	Tech       string   `json:"tech"`
	Billing    string   `json:"billing"`
	Attributes []string `json:"attributes"`
}

// TODO
// create pagination later when there are more than 100 domains in account
func (c *Client) GetDomainList(ctx context.Context) (DomainList, error) {
	var domainList DomainList

	endpoint := fmt.Sprintf("%s/domains?take=100&skip=0&orderBy=name", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)

	if err != nil {
		return domainList, fmt.Errorf("create request: %w", err)
	}

	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return domainList, fmt.Errorf("execute request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return domainList, c.errorFromResponse(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(&domainList); err != nil {
		return domainList, fmt.Errorf("decode response: %w", err)
	}

	return domainList, nil
}

func (c *Client) GetDomainInfo(ctx context.Context, domain string) (DomainInfo, error) {

	var domainInfo DomainInfo

	endpoint := fmt.Sprintf("%s/domains/%s", c.baseURL, domain)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)

	if err != nil {
		return domainInfo, fmt.Errorf("create request: %w", err)
	}

	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return domainInfo, fmt.Errorf("execute request: %w", err)
	}

	defer resp.Body.Close()

	// overcome insane API rate limiting
	// by using alternative endpoint that does the same
	// but has 60x times higher limits
	if resp.StatusCode == 429 {
		domainList, _ := c.GetDomainList(ctx)

		domainInfo, ok := findDomainByNameFromDomainList(domainList, domain)
		if ok {
			return domainInfo, nil
		}

	}

	if resp.StatusCode >= 300 {
		return domainInfo, c.errorFromResponse(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(&domainInfo); err != nil {
		return domainInfo, fmt.Errorf("decode response: %w", err)
	}
	return domainInfo, nil
}

type AutoRenewalResponse struct {
	IsEnabled bool `json:"isEnabled"`
}

func (c *Client) UpdateAutoRenew(ctx context.Context, domain string, autoRenew bool) (AutoRenewalResponse, error) {
	var autoRenewalResponse AutoRenewalResponse

	endpoint := fmt.Sprintf("%s/domains/%s/autorenew", c.baseURL, domain)
	payload := struct {
		IsEnabled bool `json:"isEnabled"`
	}{
		IsEnabled: autoRenew,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return autoRenewalResponse, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return autoRenewalResponse, fmt.Errorf("create request %w", err)
	}

	c.applyAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return autoRenewalResponse, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode != 429 {
		return autoRenewalResponse, c.errorFromResponse(resp)
	}

	if resp.StatusCode == 429 {
		time.Sleep(10 * time.Second)
		return c.UpdateAutoRenew(ctx, domain, autoRenew)
	}

	if err := json.NewDecoder(resp.Body).Decode(&autoRenewalResponse); err != nil {
		return autoRenewalResponse, fmt.Errorf("decode response: %w", err)
	}
	return autoRenewalResponse, nil
}

func findDomainByNameFromDomainList(domainList DomainList, domain string) (DomainInfo, bool) {
	for _, domainItem := range domainList.Items {
		if domainItem.Name == domain {
			return domainItem, true
		}
	}
	return DomainInfo{}, false

}

func (c *Client) UpdateTransferLock(ctx context.Context, domain string, transferLock bool) (TransferUpdateResponse, error) {
	var transferLockResponse TransferUpdateResponse

	endpoint := fmt.Sprintf("%s/domains/%s/transfer/lock", c.baseURL, domain)
	payload := struct {
		IsLocked bool `json:"isLocked"`
	}{
		IsLocked: transferLock,
	}

	body, err := json.Marshal(payload)

	if err != nil {
		return transferLockResponse, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return transferLockResponse, fmt.Errorf("create request: %w", err)
	}

	c.applyAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return transferLockResponse, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return transferLockResponse, c.errorFromResponse(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(&transferLockResponse); err != nil {
		return transferLockResponse, fmt.Errorf("decode response: %w", err)
	}

	return transferLockResponse, nil

}

type TransferUpdateResponse struct {
	IsLocked bool `json:"isLocked"`
}

type PrivacyLevel string

const (
	PrivacyLevelPublic PrivacyLevel = "public"
	PrivacyLevelHigh   PrivacyLevel = "high"
)

/*
https://docs.spaceship.dev/#tag/Domains/operation/updateDomainPrivacyPreference
*/
func (c *Client) UpdateDomainPrivacyPreference(ctx context.Context, domain string, level PrivacyLevel) error {
	switch level {
	case PrivacyLevelHigh, PrivacyLevelPublic:
	default:
		return fmt.Errorf("invalid privacy level %q", level)
	}

	endpoint := fmt.Sprintf("%s/domains/%s/privacy/preference", c.baseURL, domain)
	payload := struct {
		PrivacyLevel PrivacyLevel `json:"privacyLevel"`
		UserConsent  bool         `json:"userConsent"`
	}{
		PrivacyLevel: level,
		//user consent strictly defined as true to update this field
		UserConsent: true,
	}

	body, err := json.Marshal(payload)

	if err != nil {
		return fmt.Errorf("marshall payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.applyAuth(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode != 429 {
		return c.errorFromResponse(resp)
	}

	if resp.StatusCode == 429 {
		time.Sleep(10 * time.Second)
		return c.UpdateDomainPrivacyPreference(ctx, domain, level)
	}

	return nil
}

/*
https://docs.spaceship.dev/#tag/Domains/operation/updateDomainEmailProtectionPreference
Indicates whether WHOIS should display the contact form link
*/
func (c *Client) UpdateDomainEmailProtectionPreference(ctx context.Context, domain string, contactForm bool) error {
	//https://spaceship.dev/api/v1/domains/{domain}/privacy/email-protection-preference
	endpoint := fmt.Sprintf("%s/domains/%s/privacy/email-protection-preference", c.baseURL, domain)
	payload := struct {
		ContactForm bool `json:"contactForm"`
	}{
		ContactForm: contactForm,
	}

	body, err := json.Marshal(payload)

	if err != nil {
		return fmt.Errorf("marshall payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.applyAuth(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode != 429 {
		return c.errorFromResponse(resp)
	}

	if resp.StatusCode == 429 {
		time.Sleep(10 * time.Second)
		return c.UpdateDomainEmailProtectionPreference(ctx, domain, contactForm)
	}

	return nil
}

/*
https://docs.spaceship.dev/#tag/Domains/operation/setDomainNameservers
*/
func (c *Client) UpdateDomainNameServers(ctx context.Context, domain string, request UpdateNameserverRequest) error {
	endpoint := fmt.Sprintf("%s/domains/%s/nameservers", c.baseURL, domain)
	payload := struct {
		Provider NameserverProvider `json:"provider"`
		Hosts    []string           `json:"hosts,omitempty"` // omitempty handles conditional
	}{
		Provider: request.Provider,
		Hosts:    request.Hosts,
	}

	body, err := json.Marshal(payload)

	if err != nil {
		return fmt.Errorf("marshall payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.applyAuth(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode != 429 {
		return c.errorFromResponse(resp)
	}

	if resp.StatusCode == 429 {
		time.Sleep(10 * time.Second)
		return c.UpdateDomainNameServers(ctx, domain, request)
	}

	return nil

}

type NameserverProvider string

const (
	BasicNameserverProvider NameserverProvider = "basic"
	CustomNamerverProvider  NameserverProvider = "custom"
)

type UpdateNameserverRequest struct {
	Provider NameserverProvider
	Hosts    []string
}
