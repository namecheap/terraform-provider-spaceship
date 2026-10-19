package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// A custom host taken from a resource that does not exist yet stays unknown
// at plan time. ModifyPlan must leave the planned hosts as they are instead of
// failing to decode the unknown element into a Go string.
func TestDomainResource_ModifyPlan_UnknownCustomHost(t *testing.T) {
	ctx := context.Background()
	r := &domainResource{}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	nsType, ok := s.Attributes["nameservers"].GetType().(types.ObjectType)
	if !ok {
		t.Fatalf("expected nameservers to be an object, got %T", s.Attributes["nameservers"].GetType())
	}
	hosts := types.SetValueMust(types.StringType, []attr.Value{
		types.StringValue("ns1.example.com"),
		types.StringUnknown(),
	})
	ns := types.ObjectValueMust(nsType.AttrTypes, map[string]attr.Value{
		"provider": types.StringValue("custom"),
		"hosts":    hosts,
	})

	plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
	if diags := plan.SetAttribute(ctx, path.Root("domain"), "example.com"); diags.HasError() {
		t.Fatalf("set domain: %s", diags)
	}
	if diags := plan.SetAttribute(ctx, path.Root("nameservers"), ns); diags.HasError() {
		t.Fatalf("set nameservers: %s", diags)
	}

	resp := &resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors, got: %s", resp.Diagnostics)
	}
	var got types.Set
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("nameservers").AtName("hosts"), &got)...)
	if !got.Equal(hosts) {
		t.Errorf("expected planned hosts to stay %s, got %s", hosts, got)
	}
}
