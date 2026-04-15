package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestResolveString_ExplicitValueTakesPrecedence(t *testing.T) {
	t.Setenv("TEST_RESOLVE_ENV", "from-env")

	got := resolveString(types.StringValue("explicit"), "TEST_RESOLVE_ENV")
	if got != "explicit" {
		t.Errorf("expected %q, got %q", "explicit", got)
	}
}

func TestResolveString_FallsBackToEnvVar(t *testing.T) {
	t.Setenv("TEST_RESOLVE_ENV", "from-env")

	got := resolveString(types.StringNull(), "TEST_RESOLVE_ENV")
	if got != "from-env" {
		t.Errorf("expected %q, got %q", "from-env", got)
	}
}

func TestResolveString_FallsBackToEnvVarWhenUnknown(t *testing.T) {
	t.Setenv("TEST_RESOLVE_ENV", "from-env")

	got := resolveString(types.StringUnknown(), "TEST_RESOLVE_ENV")
	if got != "from-env" {
		t.Errorf("expected %q, got %q", "from-env", got)
	}
}

func TestResolveString_ReturnsEmptyWhenBothEmpty(t *testing.T) {
	t.Setenv("TEST_RESOLVE_ENV", "")

	got := resolveString(types.StringNull(), "TEST_RESOLVE_ENV")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestResolveString_TrimsWhitespaceFromEnvVar(t *testing.T) {
	t.Setenv("TEST_RESOLVE_ENV", "  spaced  ")

	got := resolveString(types.StringNull(), "TEST_RESOLVE_ENV")
	if got != "spaced" {
		t.Errorf("expected %q, got %q", "spaced", got)
	}
}

func TestResolveString_ExplicitEmptyStringIsUsed(t *testing.T) {
	t.Setenv("TEST_RESOLVE_ENV", "from-env")

	got := resolveString(types.StringValue(""), "TEST_RESOLVE_ENV")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
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
