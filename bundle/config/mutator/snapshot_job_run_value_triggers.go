package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type snapshotJobRunValueTriggers struct{}

// SnapshotJobRunValueTriggers records on_value_change expressions before interpolation.
func SnapshotJobRunValueTriggers() bundle.Mutator {
	return &snapshotJobRunValueTriggers{}
}

func (*snapshotJobRunValueTriggers) Name() string {
	return "SnapshotJobRunValueTriggers"
}

func (*snapshotJobRunValueTriggers) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	for name, jr := range b.Config.Resources.JobRuns {
		if jr == nil || jr.Lifecycle == nil || jr.ResolvedValueTriggers != nil {
			continue
		}

		out := make(map[string]string)
		for i, trigger := range jr.Lifecycle.Triggers {
			if trigger.OnValueChange == nil {
				continue
			}

			path := fmt.Sprintf("resources.job_runs.%s.lifecycle.triggers[%d].on_value_change", name, i)
			expr := strings.TrimSpace(*trigger.OnValueChange)
			if expr == "" {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers.on_value_change must be non-empty when set",
					Locations: b.Config.GetLocations(path),
				})
				continue
			}
			if _, ok := out[expr]; ok {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers.on_value_change expressions must be unique",
					Locations: b.Config.GetLocations(path),
				})
				continue
			}
			out[expr] = expr
		}

		if len(out) > 0 {
			jr.ResolvedValueTriggers = out
		}
	}
	return diags
}
