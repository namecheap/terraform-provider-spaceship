package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testDomain = "test-domain.com"

// rateLimitMockServer simulates the Spaceship API with controllable 429 responses.
// Toggle rateLimitGetDomain / rateLimitPutAutoRenew between test steps via PreConfig.
type rateLimitMockServer struct {
	mu        sync.Mutex
	autoRenew bool

	getDomainInfoCalls atomic.Int32
	getDomainListCalls atomic.Int32
	putAutoRenewCalls  atomic.Int32

	// When true, GET /domains/{name} returns 429 (client falls back to list).
	rateLimitGetDomain atomic.Bool
	// When true, GET /domains (list) returns 429 with Retry-After: 60.
	rateLimitGetDomainList atomic.Bool
	// When > 0, PUT autorenew returns 429 with Retry-After: 1, then decrements.
	putAutoRenew429Remaining atomic.Int32
}

func newRateLimitMockServer(autoRenew bool) *rateLimitMockServer {
	return &rateLimitMockServer{autoRenew: autoRenew}
}

func (m *rateLimitMockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/domains/"+testDomain:
		m.handleGetDomainInfo(w)
	case r.Method == http.MethodGet && r.URL.Path == "/domains":
		m.handleGetDomainList(w)
	case r.Method == http.MethodPut && r.URL.Path == "/domains/"+testDomain+"/autorenew":
		m.handlePutAutoRenew(w, r)
	default:
		http.Error(w, fmt.Sprintf("mock: unhandled %s %s", r.Method, r.URL.Path), http.StatusNotFound)
	}
}

func (m *rateLimitMockServer) handleGetDomainInfo(w http.ResponseWriter) {
	m.getDomainInfoCalls.Add(1)

	if m.rateLimitGetDomain.Load() {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	m.writeCurrentDomainInfo(w)
}

func (m *rateLimitMockServer) handleGetDomainList(w http.ResponseWriter) {
	m.getDomainListCalls.Add(1)

	if m.rateLimitGetDomainList.Load() {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	m.mu.Lock()
	info := m.domainInfo()
	m.mu.Unlock()

	resp := client.DomainList{
		Items: []client.DomainInfo{info},
		Total: 1,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (m *rateLimitMockServer) handlePutAutoRenew(w http.ResponseWriter, r *http.Request) {
	m.putAutoRenewCalls.Add(1)

	if remaining := m.putAutoRenew429Remaining.Load(); remaining > 0 {
		m.putAutoRenew429Remaining.Add(-1)
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	var payload struct {
		IsEnabled bool `json:"isEnabled"`
	}
	body, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(body, &payload)

	m.mu.Lock()
	m.autoRenew = payload.IsEnabled
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(client.AutoRenewalResponse{IsEnabled: payload.IsEnabled})
}

func (m *rateLimitMockServer) writeCurrentDomainInfo(w http.ResponseWriter) {
	m.mu.Lock()
	info := m.domainInfo()
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

// domainInfo returns a full DomainInfo snapshot. Caller must hold m.mu.
func (m *rateLimitMockServer) domainInfo() client.DomainInfo {
	return client.DomainInfo{
		Name:               testDomain,
		UnicodeName:        testDomain,
		IsPremium:          false,
		AutoRenew:          m.autoRenew,
		RegistrationDate:   "2024-01-15T00:00:00Z",
		ExpirationDate:     "2025-01-15T00:00:00Z",
		LifecycleStatus:    "registered",
		VerificationStatus: "success",
		EPPStatuses:        []string{"clientTransferProhibited"},
		Suspensions:        []client.ReasonCode{},
		PrivacyProtection: client.PrivacyProtection{
			ContactForm: true,
			Level:       "high",
		},
		Nameservers: client.Nameservers{
			Provider: "basic",
			Hosts:    []string{"launch1.spaceship.net", "launch2.spaceship.net"},
		},
		Contacts: client.Contacts{
			Registrant: "SP-12345",
			Admin:      "SP-12345",
			Tech:       "SP-12345",
			Billing:    "SP-12345",
			Attributes: []string{},
		},
	}
}

func (m *rateLimitMockServer) resetCounters() {
	m.getDomainInfoCalls.Store(0)
	m.getDomainListCalls.Store(0)
	m.putAutoRenewCalls.Store(0)
}

func testDomainConfig(domain string, autoRenew string) string {
	if autoRenew == "" {
		return fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = %q
}
`, domain)
	}
	return fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain     = %q
	auto_renew = %s
}
`, domain, autoRenew)
}

// TestDomain_autoRenewalWithRateLimiting verifies that the full Terraform
// lifecycle (plan/apply/refresh) succeeds when the API returns 429 responses.
// It exercises both rate-limit code paths:
//   - doJSONWithRetry retry loop (PUT /domains/{name}/autorenew)
//   - GetDomainInfo 429 -> GetDomainList fallback (GET /domains/{name})
//
// Uses a mock HTTP server with Retry-After: 1 so retries take ~1s instead of 60s.
func TestDomain_autoRenewalWithRateLimiting(t *testing.T) {
	mock := newRateLimitMockServer(true) // initial auto_renew = true
	server := httptest.NewServer(mock)
	defer server.Close()

	t.Setenv("SPACESHIP_API_BASE_URL", server.URL)
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Adopt domain into state. No rate limiting — establish baseline.
			{
				Config: testDomainConfig(testDomain, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "name", testDomain),
					resource.TestCheckResourceAttr(domainResourceFullName, "auto_renew", "true"),
				),
			},
			// Step 2: Toggle auto_renew to false. Enable 429 on both paths.
			{
				PreConfig: func() {
					mock.resetCounters()
					mock.rateLimitGetDomain.Store(true)
					mock.putAutoRenew429Remaining.Store(1) // one 429 then success
				},
				Config: testDomainConfig(testDomain, "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "auto_renew", "false"),
					verifyListFallbackUsed(mock),
				),
			},
			// Step 3: Toggle auto_renew to true. Rate limiting again.
			{
				PreConfig: func() {
					mock.resetCounters()
					mock.rateLimitGetDomain.Store(true)
					mock.putAutoRenew429Remaining.Store(1)
				},
				Config: testDomainConfig(testDomain, "true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "auto_renew", "true"),
					verifyListFallbackUsed(mock),
				),
			},
			// Step 4: Toggle auto_renew to false. No rate limiting — clean path works.
			{
				PreConfig: func() {
					mock.resetCounters()
					mock.rateLimitGetDomain.Store(false)
					mock.putAutoRenew429Remaining.Store(0)
				},
				Config: testDomainConfig(testDomain, "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(domainResourceFullName, "auto_renew", "false"),
				),
			},
		},
	})
}

// verifyListFallbackUsed checks that the GetDomainList fallback was called,
// proving that GET /domains/{name} returned 429 and the client fell back.
func verifyListFallbackUsed(mock *rateLimitMockServer) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if mock.getDomainListCalls.Load() == 0 {
			return fmt.Errorf("expected GetDomainList fallback to be called (GET /domains), but it was not")
		}
		return nil
	}
}

// TestDomain_timeoutCancellation verifies that when both API endpoints return
// 429 and the Terraform create timeout expires, the operation surfaces a clear
// "API Rate Limit Timeout" diagnostic instead of hanging indefinitely.
func TestDomain_timeoutCancellation(t *testing.T) {
	mock := newRateLimitMockServer(true)
	server := httptest.NewServer(mock)
	defer server.Close()

	// Rate-limit both endpoints so the client gets stuck in doJSONWithRetry.
	mock.rateLimitGetDomain.Store(true)
	mock.rateLimitGetDomainList.Store(true)

	t.Setenv("SPACESHIP_API_BASE_URL", server.URL)
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "spaceship" {}

resource "spaceship_domain" "this" {
	domain = %q
	timeouts = {
		create = "1s"
	}
}`, testDomain),
				ExpectError: regexp.MustCompile(`API Rate Limit Timeout`),
			},
		},
	})
}

// testDomainConfigResource returns the full resource address for assertions.
// Uses the same constants as the acceptance tests for consistency.
func init() {
	// Verify testDomain doesn't contain characters that would break URL routing.
	if strings.ContainsAny(testDomain, " \t\n/") {
		panic("testDomain contains invalid characters for URL path routing")
	}
}
