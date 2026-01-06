package client

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client provides a thin wrapper around Spaceship API
type Client struct {
	baseURL    url.URL
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// NewClient creates a new Spaceship API client.
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
