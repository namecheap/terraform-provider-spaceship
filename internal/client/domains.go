package client

import (
	"context"
	"net/http"
	"net/url"
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

	query := url.Values{}
	query.Set("take", "100")
	query.Set("skip", "0")
	query.Set("orderBy", "name")

	endpoint := c.endpointURL([]string{"domains"}, query)

	if _, err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &domainList); err != nil {
		return domainList, err
	}

	return domainList, nil
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

	endpoint := fmt.Sprintf("%s/domains/%s/autorenew", c.baseURL, domain)
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
