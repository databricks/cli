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
			if summary := preventDestroyError(jr.ArmedTriggerNames()); summary != "" {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   summary,
					Locations: b.Config.GetLocations(fmt.Sprintf("resources.job_runs.%s.lifecycle", name)),
				})
			}
		}
		seenValueChange := map[string]struct{}{}
		for i, t := range jr.Lifecycle.Triggers {
			path := fmt.Sprintf("resources.job_runs.%s.lifecycle.triggers[%d]", name, i)
			switch t.ArmedCount() {
			case 0:
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers entry must set on_bundle_deploy, on_file_change, or on_value_change",
					Locations: b.Config.GetLocations(path),
				})
				continue
			case 1:
				// Exactly one key: valid.
			default:
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers entry must set only one of on_bundle_deploy, on_file_change, or on_value_change",
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
			if t.OnValueChange != nil {
				expr := strings.TrimSpace(*t.OnValueChange)
				if expr == "" {
					diags = diags.Append(diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   "lifecycle.triggers.on_value_change must be non-empty when set",
						Locations: b.Config.GetLocations(path + ".on_value_change"),
					})
					continue
				}
				if _, dup := seenValueChange[expr]; dup {
					diags = diags.Append(diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   "lifecycle.triggers.on_value_change expressions must be unique",
						Locations: b.Config.GetLocations(path + ".on_value_change"),
					})
					continue
				}
				seenValueChange[expr] = struct{}{}
			}
		}
	}
	return diags
}

// preventDestroyError names the armed triggers that conflict with prevent_destroy,
// or returns an empty string when none are armed.
func preventDestroyError(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("lifecycle.triggers.%s is incompatible with lifecycle.prevent_destroy", names[0])
	default:
		last := len(names) - 1
		return fmt.Sprintf("lifecycle.triggers.%s and %s are incompatible with lifecycle.prevent_destroy", strings.Join(names[:last], ", "), names[last])
	}
}
