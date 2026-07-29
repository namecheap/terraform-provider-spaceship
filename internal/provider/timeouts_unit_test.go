package provider

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

// Every resource and data source declares a timeouts block so users can bound
// how long operations wait out API throttling.
func TestResourceSchemas_HaveTimeoutsBlock(t *testing.T) {
	resources := map[string]fwresource.Resource{
		"spaceship_domain":              &domainResource{},
		"spaceship_dns_record":          &dnsRecordResource{},
		"spaceship_dns_records":         &dnsRecordsResource{},
		"spaceship_personal_nameserver": &personalNameserverResource{},
	}
	for name, r := range resources {
		resp := &fwresource.SchemaResponse{}
		r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s schema diagnostics: %v", name, resp.Diagnostics)
		}
		if _, ok := resp.Schema.Blocks["timeouts"]; !ok {
			t.Errorf("expected a timeouts block in the %s schema", name)
		}
	}
}

func TestDataSourceSchemas_HaveTimeoutsBlock(t *testing.T) {
	dataSources := map[string]fwdatasource.DataSource{
		"spaceship_domain_info": &domainInfoDataSource{},
		"spaceship_domain_list": &domainListDataSource{},
	}
	for name, d := range dataSources {
		resp := &fwdatasource.SchemaResponse{}
		d.Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s schema diagnostics: %v", name, resp.Diagnostics)
		}
		if _, ok := resp.Schema.Blocks["timeouts"]; !ok {
			t.Errorf("expected a timeouts block in the %s schema", name)
		}
	}
}
