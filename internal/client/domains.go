package client

import (
	"context"
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

	if _, err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &domainList); err != nil {
		return domainList, err
	}

	return domainList, nil
}

func (c *Client) GetDomainInfo(ctx context.Context, domain string) (DomainInfo, error) {
	var domainInfo DomainInfo

	endpoint := fmt.Sprintf("%s/domains/%s", c.baseURL, domain)

	// overcome insane API rate limiting
	// by using alternative endpoint that does the same
	// but has 60x times higher limits
	statusCode, err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &domainInfo)
	if statusCode == 429 {
		domainList, _ := c.GetDomainList(ctx)

		domainInfo, ok := findDomainByNameFromDomainList(domainList, domain)

		if ok {
			return domainInfo, nil
		}

	}

	if err != nil {
		return domainInfo, err
	}
	return domainInfo, nil
}

func findDomainByNameFromDomainList(dl DomainList, domain string) (DomainInfo, bool) {
	for _, domainItem := range dl.Items {
		if domainItem.Name == domain {
			return domainItem, true
		}
	}
	return DomainInfo{}, false
}
