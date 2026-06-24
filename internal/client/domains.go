package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
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

const maxDomainListPageSize = 100

// GetDomainList fetches all domains in the account, following pagination until
// every page has been retrieved.
func (c *Client) GetDomainList(ctx context.Context) (DomainList, error) {
	return c.listDomains(ctx, maxDomainListPageSize)
}

// listDomains fetches all domains using the given page size, following
// pagination. The page size is a parameter so tests can force multi-page
// behavior against accounts with only a handful of domains; production callers
// use GetDomainList, which passes the API maximum (100).
func (c *Client) listDomains(ctx context.Context, pageSize int) (DomainList, error) {
	var (
		result DomainList
		skip   = 0
		total  = int64(-1)
	)

	for {
		query := url.Values{}
		query.Set("take", strconv.Itoa(pageSize))
		query.Set("skip", strconv.Itoa(skip))
		query.Set("orderBy", "name")

		endpoint := c.endpointURL([]string{"domains"}, query)

		var page DomainList
		if _, err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
			return DomainList{}, err
		}

		result.Items = append(result.Items, page.Items...)

		if total == -1 {
			total = page.Total
		}
		if int64(len(result.Items)) >= total || len(page.Items) == 0 {
			break
		}

		skip += pageSize
	}

	result.Total = total
	return result, nil
}

func (c *Client) GetDomainInfo(ctx context.Context, domain string) (DomainInfo, error) {
	var domainInfo DomainInfo

	endpoint := c.endpointURL([]string{"domains", domain}, nil)

	// overcome insane API rate limiting
	// by using alternative endpoint that does the same
	// but has 60x times higher limits
	statusCode, err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &domainInfo)
	if statusCode == http.StatusTooManyRequests {
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

type AutoRenewalResponse struct {
	IsEnabled bool `json:"isEnabled"`
}

func (c *Client) UpdateAutoRenew(ctx context.Context, domain string, value bool) (AutoRenewalResponse, error) {
	var resp AutoRenewalResponse

	endpoint := c.endpointURL([]string{"domains", domain, "autorenew"}, nil)

	payload := struct {
		IsEnabled bool `json:"isEnabled"`
	}{
		IsEnabled: value,
	}

	_, err := c.doJSON(ctx, http.MethodPut, endpoint, payload, &resp)

	if err != nil {
		return resp, err
	}
	return resp, nil

}

type NameserverProvider string

const (
	BasicNameserverProvider NameserverProvider = "basic"
	CustomNamerverProvider  NameserverProvider = "custom"
)

func (p NameserverProvider) Valid() bool {
	return p == BasicNameserverProvider || p == CustomNamerverProvider
}

type UpdateNameserverRequest struct {
	Provider NameserverProvider
	Hosts    []string
}

func DefaultBasicNameserverHosts() []string {
	return []string{"launch1.spaceship.net", "launch2.spaceship.net"}
}

/*
UpdateDomainNameServers updates the nameserver configuration for a domain.
The request Provider must be one of BasicNameserverProvider or
CustomNamerverProvider. When Provider is basic, Hosts must be empty and the
default Spaceship nameservers are used. When Provider is custom, Hosts must
contain the desired nameserver hostnames.

Docs: https://docs.spaceship.dev/#tag/Domains/operation/setDomainNameservers
*/
func (c *Client) UpdateDomainNameServers(ctx context.Context, domain string, request UpdateNameserverRequest) error {
	endpoint := c.endpointURL([]string{"domains", domain, "nameservers"}, nil)

	payload := struct {
		Provider NameserverProvider `json:"provider"`
		Hosts    []string           `json:"hosts,omitempty"` // omitempty handles conditional
	}{
		Provider: request.Provider,
		Hosts:    request.Hosts,
	}

	_, err := c.doJSON(ctx, http.MethodPut, endpoint, payload, nil)

	if err != nil {
		return err
	}
	return nil
}
