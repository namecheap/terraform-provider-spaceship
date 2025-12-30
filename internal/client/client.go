package client

import (
	"net/http"
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

func (c *Client) applyAuth(req *http.Request) {
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-API-Secret", c.apiSecret)
}
