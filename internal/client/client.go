package client

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DefaultBaseURL is the production Spaceship API endpoint.
const DefaultBaseURL = "https://spaceship.dev/api/v1"

// defaultMaxRetryWait is the client-level safety net for retry loops when
// no context deadline is set. Terraform timeouts (default 12 min) take
// precedence when present.
const defaultMaxRetryWait = 10 * time.Minute

// Client wraps the Spaceship API connection details and helpers used by
// the provider. It stores the base URL, credentials, and an HTTP client
// configured with a request timeout for all API calls.
type Client struct {
	baseURL      url.URL
	apiKey       string
	apiSecret    string
	httpClient   *http.Client
	clock        Clock
	rl           *rateLimiter
	maxRetryWait time.Duration
}

type ClientOptions func(*Client)

func WithClock(c Clock) ClientOptions {
	return func(client *Client) {
		client.clock = c
	}
}

func WithMaxRetryWait(d time.Duration) ClientOptions {
	return func(client *Client) {
		client.maxRetryWait = d
	}
}

// NewClient validates and parses the base URL, then returns a Client
// configured with the provided API credentials and a default timeout.
// The caller is responsible for supplying a full URL including scheme.
func NewClient(baseURL, apiKey, apiSecret string, opts ...ClientOptions) (*Client, error) {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	c := &Client{
		baseURL:   *parsedBaseURL,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		clock:        RealClock{},
		maxRetryWait: defaultMaxRetryWait,
	}

	for _, opt := range opts {
		opt(c)
	}

	// Initialize after opts so the rate limiter uses the injected clock (e.g. FakeClock in tests).
	c.rl = &rateLimiter{clock: c.clock}

	return c, nil
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
