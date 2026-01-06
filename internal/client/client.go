package client

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client wraps the Spaceship API connection details and helpers used by
// the provider. It stores the base URL, credentials, and an HTTP client
// configured with a request timeout for all API calls.
type Client struct {
	baseURL    url.URL
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// NewClient validates and parses the base URL, then returns a Client
// configured with the provided API credentials and a default timeout.
// The caller is responsible for supplying a full URL including scheme.
func NewClient(baseURL, apiKey, apiSecret string) (*Client, error) {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	return &Client{
		baseURL:   *parsedBaseURL,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *Client) applyAuth(req *http.Request) {
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-API-Secret", c.apiSecret)
}

func (c *Client) endpointURL(pathParts []string, query url.Values) string {
	endpoint := c.baseURL.JoinPath(pathParts...)
	if query != nil {
		endpoint.RawQuery = query.Encode()
	} else {
		endpoint.RawQuery = ""
	}
	return endpoint.String()
}
