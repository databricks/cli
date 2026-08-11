package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type validateJobRunTriggers struct{}

// ValidateJobRunTriggers checks lifecycle.triggers on job_runs.
func ValidateJobRunTriggers() bundle.Mutator {
	return &validateJobRunTriggers{}
}

func (*validateJobRunTriggers) Name() string {
	return "ValidateJobRunTriggers"
}

func (*validateJobRunTriggers) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	for name, jr := range b.Config.Resources.JobRuns {
		if jr == nil || jr.Lifecycle == nil {
			continue
		}
		// Recreate-every-deploy cannot coexist with prevent_destroy.
		if jr.HasOnBundleDeploy() && jr.Lifecycle.PreventDestroy {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "lifecycle.triggers.on_bundle_deploy is incompatible with lifecycle.prevent_destroy",
				Locations: b.Config.GetLocations(fmt.Sprintf("resources.job_runs.%s.lifecycle", name)),
			})
		}
		for i, t := range jr.Lifecycle.Triggers {
			path := fmt.Sprintf("resources.job_runs.%s.lifecycle.triggers[%d]", name, i)
			if t.OnBundleDeploy == nil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers entry must set on_bundle_deploy: true",
					Locations: b.Config.GetLocations(path),
				})
				continue
			}
			if !*t.OnBundleDeploy {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers.on_bundle_deploy must be true when set",
					Locations: b.Config.GetLocations(path + ".on_bundle_deploy"),
				})
			}
		}
	}
	return diags
}
