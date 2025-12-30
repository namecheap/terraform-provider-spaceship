package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

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

	if resp.StatusCode >= 300 {
		return domainInfo, c.errorFromResponse(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(&domainInfo); err != nil {
		return domainInfo, fmt.Errorf("decode response: %w", err)
	}
	return domainInfo, nil
}
