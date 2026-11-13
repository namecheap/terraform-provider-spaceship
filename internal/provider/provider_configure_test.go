package provider

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The API key and secret must go only to the compiled-in endpoint. Someone who
// can set the provider's environment but not read the secret must not be able
// to redirect it with SPACESHIP_BASE_URL.
func TestConfigure_IgnoresBaseURLEnv(t *testing.T) {
	var redirected atomic.Int32
	trap := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirected.Add(1)
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(trap.Close)

	server := mockDomainAPIReadOnly(t, baseDomainInfo())
	setTestBaseURL(t, server.URL+"/v1")
	t.Setenv("SPACESHIP_BASE_URL", trap.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
resource "spaceship_domain" "test" {
  domain = "example.com"
}
`,
			},
		},
	})

	if n := redirected.Load(); n != 0 {
		t.Errorf("SPACESHIP_BASE_URL received %d request(s); the provider must ignore it", n)
	}
}

func TestRedactURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "no userinfo", in: defaultBaseURL, want: defaultBaseURL},
		{name: "password masked", in: "https://user:hunter2@example.test/v1", want: "https://user:xxxxx@example.test/v1"},
		{name: "unparseable withheld", in: "https://user:hunter2@[::1/v1", want: "<invalid URL>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactURL(tt.in); got != tt.want {
				t.Errorf("redactURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestConfigure_MissingAPIKey(t *testing.T) {
	t.Setenv("SPACESHIP_API_KEY", "")
	t.Setenv("SPACESHIP_API_SECRET", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      `data "spaceship_domain_list" "test" {}`,
				ExpectError: regexp.MustCompile("Missing Spaceship API key"),
			},
		},
	})
}

func TestConfigure_MissingAPISecret(t *testing.T) {
	t.Setenv("SPACESHIP_API_KEY", "some-key")
	t.Setenv("SPACESHIP_API_SECRET", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      `data "spaceship_domain_list" "test" {}`,
				ExpectError: regexp.MustCompile("Missing Spaceship API secret"),
			},
		},
	})
}

// Credentials taken from a value known only after apply arrive in Configure as
// unknown. They must not fall back to the environment: plan would run with the
// env account while apply uses the configured one.
func TestConfigure_UnknownCredentialsDoNotFallBackToEnv(t *testing.T) {
	t.Setenv("SPACESHIP_API_KEY", "env-key")
	t.Setenv("SPACESHIP_API_SECRET", "env-secret")
	// Any request that slips through fails fast instead of reaching the real API.
	setTestBaseURL(t, "http://127.0.0.1:1")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
resource "terraform_data" "credentials" {
  input = "configured"
}

provider "spaceship" {
  api_key    = terraform_data.credentials.output
  api_secret = terraform_data.credentials.output
}

resource "spaceship_dns_record" "test" {
  domain  = "example.com"
  type    = "A"
  name    = "www"
  address = "192.0.2.1"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ExpectError:        regexp.MustCompile("Unknown Spaceship API key"),
			},
		},
	})
}

func TestConfigure_MissingBothCredentials(t *testing.T) {
	t.Setenv("SPACESHIP_API_KEY", "")
	t.Setenv("SPACESHIP_API_SECRET", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      `data "spaceship_domain_list" "test" {}`,
				ExpectError: regexp.MustCompile("Missing Spaceship API"),
			},
		},
	})
}
