package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
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
		for i, t := range jr.Lifecycle.Triggers {
			path := structpath.NewPath(nil, "resources", "job_runs", name, "lifecycle", "triggers", i)
			if t.OnBundleDeploy == nil && t.OnFileChange == nil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers entry must set on_bundle_deploy or on_file_change",
					Locations: b.Config.GetLocationsOf(path),
				})
				continue
			}
			if t.OnBundleDeploy != nil && t.OnFileChange != nil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers entry must set only one of on_bundle_deploy or on_file_change",
					Locations: b.Config.GetLocationsOf(path),
				})
				continue
			}
			if t.OnBundleDeploy != nil && !*t.OnBundleDeploy {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers.on_bundle_deploy must be true when set",
					Locations: b.Config.GetLocationsOf(structpath.NewPath(path, "on_bundle_deploy")),
				})
			}
			if t.OnFileChange != nil {
				onFileChange := structpath.NewPath(path, "on_file_change")
				if strings.TrimSpace(*t.OnFileChange) == "" {
					diags = diags.Append(diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   "lifecycle.triggers.on_file_change must be non-empty when set",
						Locations: b.Config.GetLocationsOf(onFileChange),
					})
					continue
				}
				// Report bad patterns at validate time; hashing only runs on deploy.
				_, patternDiags := validateFileTriggerPattern(b, onFileChange, *t.OnFileChange)
				diags = diags.Extend(patternDiags)
			}
		}
	}
	return diags
}
