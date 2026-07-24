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

	// idempotency_token is computed in DoCreate, so a user value has no effect.
	// Reject it and point at the supported knob (`rerun_token`) for forcing a run.
	for name, jr := range b.Config.Resources.JobRuns {
		// An empty `job_runs.<name>:` entry loads as a present key with a nil
		// pointer value, so guard before dereferencing.
		if jr == nil || jr.IdempotencyToken == "" {
			continue
		}
		path := "resources.job_runs." + name + ".idempotency_token"
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "idempotency_token is computed automatically and must not be set in bundle configuration; set `rerun_token` to force a new run",
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
