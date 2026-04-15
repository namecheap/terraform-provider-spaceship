package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestSuspensionsToTerraformList_Nil(t *testing.T) {
	list, diags := suspensionsToTerraformList(context.Background(), nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !list.IsNull() {
		t.Fatalf("expected null list for nil suspensions, got %v", list)
	}
}

func TestSuspensionsToTerraformList_Empty(t *testing.T) {
	list, diags := suspensionsToTerraformList(context.Background(), []client.ReasonCode{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !list.IsNull() {
		t.Fatalf("expected null list for empty suspensions, got %v", list)
	}
}

func TestSuspensionsToTerraformList_WithValues(t *testing.T) {
	suspensions := []client.ReasonCode{
		{ReasonCode: "serverHold"},
		{ReasonCode: "clientHold"},
	}

	list, diags := suspensionsToTerraformList(context.Background(), suspensions)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if list.IsNull() || list.IsUnknown() {
		t.Fatalf("expected non-null list, got null=%v unknown=%v", list.IsNull(), list.IsUnknown())
	}

	elements := list.Elements()
	if len(elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elements))
	}
}

func TestContactsToTerraformObject_Full(t *testing.T) {
	contacts := client.Contacts{
		Registrant: "reg-handle",
		Admin:      "admin-handle",
		Tech:       "tech-handle",
		Billing:    "billing-handle",
		Attributes: []string{"attr1", "attr2"},
	}

	obj, diags := contactsToTerraformObject(context.Background(), contacts)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if obj.IsNull() || obj.IsUnknown() {
		t.Fatalf("expected non-null object, got null=%v unknown=%v", obj.IsNull(), obj.IsUnknown())
	}

	attrs := obj.Attributes()

	registrant := attrs["registrant"]
	if registrant.String() != `"reg-handle"` {
		t.Errorf("expected registrant to be reg-handle, got %s", registrant.String())
	}

	admin := attrs["admin"]
	if admin.String() != `"admin-handle"` {
		t.Errorf("expected admin to be admin-handle, got %s", admin.String())
	}

	tech := attrs["tech"]
	if tech.String() != `"tech-handle"` {
		t.Errorf("expected tech to be tech-handle, got %s", tech.String())
	}

	billing := attrs["billing"]
	if billing.String() != `"billing-handle"` {
		t.Errorf("expected billing to be billing-handle, got %s", billing.String())
	}

	attrList := attrs["attributes"]
	if attrList.IsNull() || attrList.IsUnknown() {
		t.Errorf("expected non-null attributes list, got null=%v unknown=%v", attrList.IsNull(), attrList.IsUnknown())
	}
}

func TestContactsToTerraformObject_MinimalWithNilAttributes(t *testing.T) {
	contacts := client.Contacts{
		Registrant: "reg-handle",
		Attributes: nil,
	}

	obj, diags := contactsToTerraformObject(context.Background(), contacts)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if obj.IsNull() || obj.IsUnknown() {
		t.Fatalf("expected non-null object, got null=%v unknown=%v", obj.IsNull(), obj.IsUnknown())
	}

	attrs := obj.Attributes()

	registrant := attrs["registrant"]
	if registrant.String() != `"reg-handle"` {
		t.Errorf("expected registrant to be reg-handle, got %s", registrant.String())
	}

	attrList := attrs["attributes"]
	if !attrList.IsNull() {
		t.Errorf("expected null attributes list when contacts.Attributes is nil, got %v", attrList)
	}
}

func TestLogDiagnostics_NoError(t *testing.T) {
	var d diag.Diagnostics
	// Should return early without error when there are no diagnostics.
	logDiagnostics(context.Background(), "test", d)
}

func TestLogDiagnostics_WithError(t *testing.T) {
	var d diag.Diagnostics
	d.AddError("summary", "detail message")
	// tflog is a no-op without a properly initialized context, so this
	// exercises the error branch without producing log output.
	logDiagnostics(context.Background(), "test", d)
}

func TestDomainResource_DeleteRemovesFromState(t *testing.T) {
	server := mockDomainAPIReadOnly(t, baseDomainInfo())

	t.Setenv("SPACESHIP_BASE_URL", server.URL+"/v1")
	t.Setenv("SPACESHIP_API_KEY", "test-key")
	t.Setenv("SPACESHIP_API_SECRET", "test-secret")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testMockProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create the resource.
			{
				Config: `
provider "spaceship" {}

resource "spaceship_domain" "test" {
  domain = "example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spaceship_domain.test", "domain", "example.com"),
				),
			},
			// Step 2: Destroy — exercises the Delete method
			// (a no-op that just removes the resource from state).
			{
				Config: `
provider "spaceship" {}

resource "spaceship_domain" "test" {
  domain = "example.com"
}
`,
				Destroy: true,
			},
		},
	})
}

// mockDomainAPIReadOnly creates a simple deterministic mock that always
// returns the same domain info for GET requests. No state mutation.
func mockDomainAPIReadOnly(t *testing.T, domain client.DomainInfo) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/domains/") {
			_ = json.NewEncoder(w).Encode(domain)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	return server
}
