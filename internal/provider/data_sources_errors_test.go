package provider

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mockDataSourceAPIWithError creates a mock that returns 500 for all domain
// API requests, exercising the Read error paths in data sources.
func mockDataSourceAPIWithError(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "api error"}`))
	}))
	t.Cleanup(server.Close)
	return server
}

// TestDomainInfoDataSource_ReadError verifies that the domain info data
// source returns an error when GetDomainInfo fails.
func TestDomainInfoDataSource_ReadError(t *testing.T) {
	server := mockDataSourceAPIWithError(t)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "spaceship" {}

data "spaceship_domain_info" "test" {
  domain = "example.com"
}
`,
				ExpectError: regexp.MustCompile(`Unable to read domain info`),
			},
		},
	})
}

// TestDomainListDataSource_ReadError verifies that the domain list data
// source returns an error when GetDomainList fails.
func TestDomainListDataSource_ReadError(t *testing.T) {
	server := mockDataSourceAPIWithError(t)

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "spaceship" {}

data "spaceship_domain_list" "test" {}
`,
				ExpectError: regexp.MustCompile(`Unable to read domain list`),
			},
		},
	})
}
