package provider

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
)

// Both domain data sources declare a timeouts block so users can bound how
// long reads wait out API throttling.
func TestDomainInfoDataSourceSchema_HasTimeoutsBlock(t *testing.T) {
	resp := &fwdatasource.SchemaResponse{}
	(&domainInfoDataSource{}).Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Blocks["timeouts"]; !ok {
		t.Fatal("expected a timeouts block in the spaceship_domain_info schema")
	}
}

func TestDomainListDataSourceSchema_HasTimeoutsBlock(t *testing.T) {
	resp := &fwdatasource.SchemaResponse{}
	(&domainListDataSource{}).Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Blocks["timeouts"]; !ok {
		t.Fatal("expected a timeouts block in the spaceship_domain_list schema")
	}
}
