package mutator

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
)

type snapshotJobRunValueTriggers struct{}

// SnapshotJobRunValueTriggers records each job_run's on_value_change expressions in
// b.JobRunValueTriggers before variable resolution, so that ResolveJobRunValueTriggers
// can key the resolved values by the expression. A job_run that is already recorded is
// skipped: when a Python mutator updates it, its triggers are already resolved.
func SnapshotJobRunValueTriggers() bundle.Mutator {
	return &snapshotJobRunValueTriggers{}
}

func (*snapshotJobRunValueTriggers) Name() string {
	return "SnapshotJobRunValueTriggers"
}

func (*snapshotJobRunValueTriggers) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	// Sorted so diagnostics from several job_runs come out in a stable order.
	for _, name := range slices.Sorted(maps.Keys(b.Config.Resources.JobRuns)) {
		jr := b.Config.Resources.JobRuns[name]
		if jr == nil || jr.Lifecycle == nil || b.JobRunValueTriggers[name] != nil {
			continue
		}

		exprs := make([]string, len(jr.Lifecycle.Triggers))
		seen := make(map[string]bool)
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
			if seen[expr] {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers.on_value_change expressions must be unique",
					Locations: b.Config.GetLocations(path),
				})
				continue
			}
			seen[expr] = true
			exprs[i] = expr
		}

		if len(seen) > 0 {
			if b.JobRunValueTriggers == nil {
				b.JobRunValueTriggers = make(map[string][]string)
			}
			b.JobRunValueTriggers[name] = exprs
		}
	}
	return diags
}

type resolveJobRunValueTriggers struct{}

// ResolveJobRunValueTriggers stores each job_run's resolved on_value_change values in
// lifecycle.triggers_state, keyed by the expressions SnapshotJobRunValueTriggers
// recorded. The values are paired with the expressions by trigger position, so a
// Python mutator must not add or remove triggers of a job_run that sets on_value_change.
func ResolveJobRunValueTriggers() bundle.Mutator {
	return &resolveJobRunValueTriggers{}
}

func (*resolveJobRunValueTriggers) Name() string {
	return "ResolveJobRunValueTriggers"
}

func (*resolveJobRunValueTriggers) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, name := range slices.Sorted(maps.Keys(b.JobRunValueTriggers)) {
		exprs := b.JobRunValueTriggers[name]
		jr := b.Config.Resources.JobRuns[name]
		path := fmt.Sprintf("resources.job_runs.%s.lifecycle.triggers", name)
		if jr.Lifecycle == nil || len(jr.Lifecycle.Triggers) != len(exprs) {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "Python mutators cannot add or remove lifecycle.triggers of a job_run that sets on_value_change",
				Locations: b.Config.GetLocations(path),
			})
			continue
		}

		values := make(map[string]string)
		for i, expr := range exprs {
			if expr == "" {
				continue
			}
			value := jr.Lifecycle.Triggers[i].OnValueChange
			if value == nil {
				// Normalization dropped a value that is a map or a list, with a warning.
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.triggers.on_value_change must resolve to a string, number or boolean",
					Locations: b.Config.GetLocations(fmt.Sprintf("%s[%d]", path, i)),
				})
				continue
			}
			// Trimmed like the expression, so padding the expression does not re-fire.
			values[expr] = strings.TrimSpace(*value)
		}

		if jr.Lifecycle.TriggersState == nil {
			jr.Lifecycle.TriggersState = &resources.JobRunTriggersState{}
		}
		jr.Lifecycle.TriggersState.OnValueChange = values
	}
	return diags
}
