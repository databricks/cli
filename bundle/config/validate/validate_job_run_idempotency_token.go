package validate

import (
	"context"
	"maps"
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
	return "validate:job_run_idempotency_token"
}

func (v *validateJobRunIdempotencyToken) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// Map iteration order is randomized; sort by name for stable output.
	for _, name := range slices.Sorted(maps.Keys(b.Config.Resources.JobRuns)) {
		jobRun := b.Config.Resources.JobRuns[name]
		// An empty `job_runs.<name>:` entry loads as a present key with a nil value.
		if jobRun == nil || jobRun.IdempotencyToken == "" {
			continue
		}
		// The CLI sets a token of its own on every run-now, so a configured one would
		// be overwritten. It would also outlive the run it triggered: the Jobs API
		// keeps a token reserved once its run is deleted, which recreate and destroy
		// both do, and rejects the next run-now that carries it.
		path := "resources.job_runs." + name + ".idempotency_token"
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "idempotency_token must not be set in bundle configuration; the CLI sets it on each run-now request",
			Paths:     []dyn.Path{dyn.MustPathFromString(path)},
			Locations: b.Config.GetLocations(path),
		})
	}

	return diags
}
