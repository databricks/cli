package validate

import (
	"cmp"
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

func ValidateJobRunIdempotencyToken() bundle.ReadOnlyMutator {
	return &validateJobRunIdempotencyToken{}
}

type validateJobRunIdempotencyToken struct{ bundle.RO }

func (v *validateJobRunIdempotencyToken) Name() string {
	return "validate:validate_job_run_idempotency_token"
}

func (v *validateJobRunIdempotencyToken) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// The idempotency_token is derived automatically in DoCreate so a retried
	// deploy reuses the existing run instead of triggering a duplicate. A value
	// set here would have no effect, so reject it up front with a clear error.
	for name, jr := range b.Config.Resources.JobRuns {
		// An empty `job_runs.<name>:` entry unmarshals to a nil pointer
		// (convert.ToTyped), so guard before dereferencing.
		if jr == nil || jr.IdempotencyToken == "" {
			continue
		}
		path := "resources.job_runs." + name + ".idempotency_token"
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "idempotency_token is computed automatically and must not be set in bundle configuration",
			Paths:     []dyn.Path{dyn.MustPathFromString(path)},
			Locations: b.Config.GetLocations(path),
		})
	}

	// Map iteration order is randomized; sort by path for stable output.
	slices.SortFunc(diags, func(x, y diag.Diagnostic) int {
		return cmp.Compare(x.Paths[0].String(), y.Paths[0].String())
	})

	return diags
}
