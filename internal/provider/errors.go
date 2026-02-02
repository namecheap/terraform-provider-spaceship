package provider

import (
	"errors"
	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// addErrorDiag appends an error diagnostic to diags. For rate-limit timeout
// errors it produces a Terraform-specific summary and actionable guidance;
// all other errors fall through with the supplied summary and err.Error().
func addErrorDiag(diags *diag.Diagnostics, summary string, err error) {
	var rlErr *client.RateLimitTimeoutError
	if errors.As(err, &rlErr) {
		diags.AddError(
			"API Rate Limit Timeout",
			"The operation timed out while waiting for the Spaceship API rate-limit "+
				"window to clear. Multiple concurrent Terraform operations may be "+
				"exhausting your API quota.\n\n"+
				"Try one of the following:\n"+
				"  • Increase the timeout in the resource's 'timeouts' block\n"+
				"  • Wait for the rate-limit window to pass and run again\n"+
				"  • Reduce parallelism:  terraform apply -parallelism=1",
		)
		return
	}
	diags.AddError(summary, err.Error())
}
