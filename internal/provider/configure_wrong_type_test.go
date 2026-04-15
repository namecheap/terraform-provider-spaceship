package provider

import (
	"context"
	"testing"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestDNSRecordsResource_ConfigureWrongType verifies that Configure adds a
// diagnostic error when ProviderData is not *client.Client.
func TestDNSRecordsResource_ConfigureWrongType(t *testing.T) {
	r := &dnsRecordsResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "wrong-type",
	}, resp)

	assertHasError(t, resp.Diagnostics, "Unexpected provider data type")
}

// TestDomainResource_ConfigureWrongType verifies the same for domain resource.
func TestDomainResource_ConfigureWrongType(t *testing.T) {
	r := &domainResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "wrong-type",
	}, resp)

	assertHasError(t, resp.Diagnostics, "Unexpected provider data type")
}

// TestDomainInfoDataSource_ConfigureWrongType verifies the same for
// domain info data source.
func TestDomainInfoDataSource_ConfigureWrongType(t *testing.T) {
	ds := NewDomainInfoDataSource()
	resp := &datasource.ConfigureResponse{}

	ds.(datasource.DataSourceWithConfigure).Configure(
		context.Background(),
		datasource.ConfigureRequest{ProviderData: 42},
		resp,
	)

	assertHasError(t, resp.Diagnostics, "Unexpected provider data type")
}

// TestDomainListDataSource_ConfigureWrongType verifies the same for
// domain list data source.
func TestDomainListDataSource_ConfigureWrongType(t *testing.T) {
	ds := NewDomainListDataSource()
	resp := &datasource.ConfigureResponse{}

	ds.(datasource.DataSourceWithConfigure).Configure(
		context.Background(),
		datasource.ConfigureRequest{ProviderData: 42},
		resp,
	)

	assertHasError(t, resp.Diagnostics, "Unexpected provider data type")
}

// TestDNSRecordsResource_SchemaIsValid ensures the schema can be retrieved
// without error.
func TestDNSRecordsResource_SchemaIsValid(t *testing.T) {
	r := NewDNSRecordsResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	if resp.Schema.Attributes == nil {
		t.Fatal("expected non-nil schema attributes")
	}
}

// TestDomainResource_SchemaIsValid ensures the schema can be retrieved.
func TestDomainResource_SchemaIsValid(t *testing.T) {
	r := NewDomainResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	if resp.Schema.Attributes == nil {
		t.Fatal("expected non-nil schema attributes")
	}
}

// TestDomainInfoDataSource_SchemaIsValid ensures the data source schema
// can be retrieved.
func TestDomainInfoDataSource_SchemaIsValid(t *testing.T) {
	ds := NewDomainInfoDataSource()
	resp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Schema.Attributes == nil {
		t.Fatal("expected non-nil schema attributes")
	}
}

// TestDomainListDataSource_SchemaIsValid ensures the data source schema
// can be retrieved.
func TestDomainListDataSource_SchemaIsValid(t *testing.T) {
	ds := NewDomainListDataSource()
	resp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Schema.Attributes == nil {
		t.Fatal("expected non-nil schema attributes")
	}
}

// assertHasError is a test helper that checks diagnostics contain an error
// with the specified summary.
func assertHasError(t *testing.T, diags diag.Diagnostics, expectedSummary string) {
	t.Helper()
	if !diags.HasError() {
		t.Fatalf("expected error with summary %q, got no errors", expectedSummary)
	}
	for _, d := range diags {
		if d.Summary() == expectedSummary {
			return
		}
	}
	t.Errorf("expected error with summary %q, got: %v", expectedSummary, diags)
}

// Ensure schemas implement expected types (compile-time check).
var _ resource.Resource = &dnsRecordsResource{}
var _ resource.Resource = &domainResource{}

// Verify Schema methods return proper attribute types.
func TestDNSRecordsResource_SchemaAttributes(t *testing.T) {
	r := NewDNSRecordsResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	required := []string{"domain", "records"}
	for _, attr := range required {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing required attribute %q", attr)
		}
	}
}

func TestDomainResource_SchemaAttributes(t *testing.T) {
	r := NewDomainResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	required := []string{"domain", "auto_renew", "nameservers"}
	for _, attr := range required {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing required attribute %q", attr)
		}
	}
}

// Suppress "unused" for imported schema packages used only for interface assertions.
var (
	_ resourceschema.Schema    = resourceschema.Schema{}
	_ datasourceschema.Schema  = datasourceschema.Schema{}
)
