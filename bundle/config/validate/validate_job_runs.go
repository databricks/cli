package validate

import (
	"context"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

func ValidateJobRuns() bundle.ReadOnlyMutator {
	return &validateJobRuns{}
}

type validateJobRuns struct{ bundle.RO }

func (v *validateJobRuns) Name() string {
	return "validate:job_runs"
}

func (v *validateJobRuns) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// Sorted so the reported order does not depend on map iteration order.
	for _, name := range slices.Sorted(maps.Keys(b.Config.Resources.JobRuns)) {
		jr := b.Config.Resources.JobRuns[name]
		// An empty `job_runs.<name>:` entry loads as a present key with a nil value.
		if jr == nil || jr.IdempotencyToken == "" {
			continue
		}
		// DoCreate computes the token so a retried deploy rejoins the run it already
		// triggered; a user-set one would break that.
		path := "resources.job_runs." + name + ".idempotency_token"
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "idempotency_token is computed automatically and must not be set in bundle configuration; set `rerun_token` to force a new run",
			Paths:     []dyn.Path{dyn.MustPathFromString(path)},
			Locations: b.Config.GetLocations(path),
		})
	}

	return diags
}
