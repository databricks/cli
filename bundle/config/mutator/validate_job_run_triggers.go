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
		for i, t := range jr.Lifecycle.Triggers {
			path := fmt.Sprintf("resources.job_runs.%s.lifecycle.triggers[%d]", name, i)
			if t.FieldCount() != 1 {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers entry must set exactly one trigger mode",
					Locations: b.Config.GetLocations(path),
				})
			}
			if t.OnBundleDeploy != nil && !*t.OnBundleDeploy {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers.on_bundle_deploy must be true when set",
					Locations: b.Config.GetLocations(path + ".on_bundle_deploy"),
				})
			}
			if t.OnFileChange != "" {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers.on_file_change is not supported yet",
					Locations: b.Config.GetLocations(path + ".on_file_change"),
				})
			}
			if t.OnValueChange != "" {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers.on_value_change is not supported yet",
					Locations: b.Config.GetLocations(path + ".on_value_change"),
				})
			}
		}
	}
	return diags
}
