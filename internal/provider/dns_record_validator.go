package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// singularRecordValidator adapts a validator.Object — designed to run against
// each element of a nested record list — to operate on the flat singular
// dns_record resource. It reads the resource config into a dnsRecordModel,
// synthesizes a types.Object whose attributes match what the inner validator
// expects, and dispatches to the inner ValidateObject. Diagnostic paths use
// path.Empty() as the base, so an error on `exchange` produces a path of
// `exchange` (not `records[0].exchange`).
type singularRecordValidator struct {
	inner validator.Object
}

func (a singularRecordValidator) Description(ctx context.Context) string {
	return a.inner.Description(ctx)
}

func (a singularRecordValidator) MarkdownDescription(ctx context.Context) string {
	return a.inner.MarkdownDescription(ctx)
}

func (a singularRecordValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model dnsRecordResourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	obj, objDiags := types.ObjectValueFrom(ctx, dnsRecordObjectType.AttrTypes, model.dnsRecordModel)
	resp.Diagnostics.Append(objDiags...)
	if objDiags.HasError() {
		return
	}

	objReq := validator.ObjectRequest{
		Path:        path.Empty(),
		Config:      req.Config,
		ConfigValue: obj,
	}
	objResp := &validator.ObjectResponse{}
	a.inner.ValidateObject(ctx, objReq, objResp)
	resp.Diagnostics.Append(objResp.Diagnostics...)
}
