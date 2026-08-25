package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type validateJobRunTriggers struct{}

// ValidateJobRunTriggers rejects invalid lifecycle.triggers on job_runs.
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
		if jr.Lifecycle.PreventDestroy {
			if summary := preventDestroyError(jr.HasOnBundleDeploy(), jr.HasOnFileChange()); summary != "" {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   summary,
					Locations: b.Config.GetLocations(fmt.Sprintf("resources.job_runs.%s.lifecycle", name)),
				})
			}
		}
		for i, t := range jr.Lifecycle.Triggers {
			path := fmt.Sprintf("resources.job_runs.%s.lifecycle.triggers[%d]", name, i)
			if t.OnBundleDeploy == nil && t.OnFileChange == nil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers entry must set on_bundle_deploy or on_file_change",
					Locations: b.Config.GetLocations(path),
				})
				continue
			}
			if t.OnBundleDeploy != nil && t.OnFileChange != nil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers entry must set only one of on_bundle_deploy or on_file_change",
					Locations: b.Config.GetLocations(path),
				})
				continue
			}
			if t.OnBundleDeploy != nil && !*t.OnBundleDeploy {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers.on_bundle_deploy must be true when set",
					Locations: b.Config.GetLocations(path + ".on_bundle_deploy"),
				})
			}
			if t.OnFileChange != nil && strings.TrimSpace(*t.OnFileChange) == "" {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers.on_file_change must be non-empty when set",
					Locations: b.Config.GetLocations(path + ".on_file_change"),
				})
			}
		}
	}
	return diags
}

func preventDestroyError(onBundleDeploy, onFileChange bool) string {
	switch {
	case onBundleDeploy && onFileChange:
		return "lifecycle.triggers.on_bundle_deploy and on_file_change are incompatible with lifecycle.prevent_destroy"
	case onBundleDeploy:
		return "lifecycle.triggers.on_bundle_deploy is incompatible with lifecycle.prevent_destroy"
	case onFileChange:
		return "lifecycle.triggers.on_file_change is incompatible with lifecycle.prevent_destroy"
	default:
		return ""
	}
}
