package provider

import (
	"context"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStringValueOrNull_NonEmpty(t *testing.T) {
	result := stringValueOrNull("hello")
	if result.IsNull() {
		t.Error("expected non-null for non-empty string")
	}
	if result.ValueString() != "hello" {
		t.Errorf("expected %q, got %q", "hello", result.ValueString())
	}
}

func TestStringValueOrNull_Empty(t *testing.T) {
	result := stringValueOrNull("")
	if !result.IsNull() {
		t.Error("expected null for empty string")
	}
}

func TestFlattenSuspensions_Empty(t *testing.T) {
	result := flattenSuspensions(nil)
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestFlattenSuspensions_WithValues(t *testing.T) {
	result := flattenSuspensions([]client.ReasonCode{
		{ReasonCode: "abuse"},
		{ReasonCode: "fraud"},
	})
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0].ReasonCode.ValueString() != "abuse" {
		t.Errorf("expected %q, got %q", "abuse", result[0].ReasonCode.ValueString())
	}
	if result[1].ReasonCode.ValueString() != "fraud" {
		t.Errorf("expected %q, got %q", "fraud", result[1].ReasonCode.ValueString())
	}
}

func TestFlattenSuspensions_EmptyReasonCode(t *testing.T) {
	result := flattenSuspensions([]client.ReasonCode{{ReasonCode: ""}})
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if !result[0].ReasonCode.IsNull() {
		t.Error("expected null for empty reason code")
	}
}

func TestFlattenPrivacyProtection(t *testing.T) {
	result := flattenPrivacyProtection(client.PrivacyProtection{
		ContactForm: true,
		Level:       "high",
	})
	if result.ContactForm.ValueBool() != true {
		t.Error("expected ContactForm to be true")
	}
	if result.Level.ValueString() != "high" {
		t.Errorf("expected level %q, got %q", "high", result.Level.ValueString())
	}
}

func TestFlattenNameservers(t *testing.T) {
	ctx := context.Background()
	result, diags := flattenNameservers(ctx, client.Nameservers{
		Provider: "custom",
		Hosts:    []string{"ns1.example.com", "ns2.example.com"},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result.Provider.ValueString() != "custom" {
		t.Errorf("expected provider %q, got %q", "custom", result.Provider.ValueString())
	}
	if result.Hosts.IsNull() {
		t.Error("expected non-null hosts")
	}
}

func TestFlattenContacts(t *testing.T) {
	ctx := context.Background()
	result, diags := flattenContacts(ctx, client.Contacts{
		Registrant: "reg-handle",
		Admin:      "admin-handle",
		Tech:       "",
		Billing:    "billing-handle",
		Attributes: []string{"attr1"},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result.Registrant.ValueString() != "reg-handle" {
		t.Errorf("expected registrant %q, got %q", "reg-handle", result.Registrant.ValueString())
	}
	if result.Admin.ValueString() != "admin-handle" {
		t.Errorf("expected admin %q, got %q", "admin-handle", result.Admin.ValueString())
	}
	if !result.Tech.IsNull() {
		t.Error("expected null Tech for empty string")
	}
}

func TestBuildDomainModel(t *testing.T) {
	ctx := context.Background()
	info := client.DomainInfo{
		Name:               "example.com",
		UnicodeName:        "example.com",
		IsPremium:          false,
		AutoRenew:          true,
		RegistrationDate:   "2024-01-01",
		ExpirationDate:     "2025-01-01",
		LifecycleStatus:    "registered",
		VerificationStatus: "success",
		EPPStatuses:        []string{"clientTransferProhibited"},
		Suspensions:        nil,
		PrivacyProtection:  client.PrivacyProtection{Level: "high", ContactForm: true},
		Nameservers:        client.Nameservers{Provider: "basic", Hosts: []string{"launch1.spaceship.net"}},
		Contacts:           client.Contacts{Registrant: "reg", Attributes: []string{}},
	}

	model, diags := buildDomainModel(ctx, info)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if model.Name.ValueString() != "example.com" {
		t.Errorf("expected name %q, got %q", "example.com", model.Name.ValueString())
	}
	if model.AutoRenew.ValueBool() != true {
		t.Error("expected AutoRenew to be true")
	}
	if model.LifecycleStatus.ValueString() != "registered" {
		t.Errorf("expected lifecycle %q, got %q", "registered", model.LifecycleStatus.ValueString())
	}
	if model.VerificationStatus.ValueString() != "success" {
		t.Errorf("expected verification %q, got %q", "success", model.VerificationStatus.ValueString())
	}
}

func TestBuildDomainModel_NullVerificationStatus(t *testing.T) {
	ctx := context.Background()
	info := client.DomainInfo{
		Name:              "example.com",
		EPPStatuses:       []string{},
		Nameservers:       client.Nameservers{Provider: "basic", Hosts: []string{}},
		Contacts:          client.Contacts{Registrant: "reg", Attributes: []string{}},
		PrivacyProtection: client.PrivacyProtection{Level: "public"},
	}

	model, diags := buildDomainModel(ctx, info)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !model.VerificationStatus.IsNull() {
		t.Error("expected null VerificationStatus for empty string")
	}
}

func TestResolveString_FromValue(t *testing.T) {
	result := resolveString(types.StringValue("inline"), "NONEXISTENT_ENV_VAR")
	if result != "inline" {
		t.Errorf("expected %q, got %q", "inline", result)
	}
}

func TestResolveString_NullFallsToEnv(t *testing.T) {
	t.Setenv("TEST_RESOLVE_VAR", "from-env")
	result := resolveString(types.StringNull(), "TEST_RESOLVE_VAR")
	if result != "from-env" {
		t.Errorf("expected %q, got %q", "from-env", result)
	}
}

func TestResolveString_UnknownFallsToEnv(t *testing.T) {
	t.Setenv("TEST_RESOLVE_VAR2", "env-val")
	result := resolveString(types.StringUnknown(), "TEST_RESOLVE_VAR2")
	if result != "env-val" {
		t.Errorf("expected %q, got %q", "env-val", result)
	}
}
