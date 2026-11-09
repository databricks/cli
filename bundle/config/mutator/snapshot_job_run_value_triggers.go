package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type snapshotJobRunValueTriggers struct{}

// SnapshotJobRunValueTriggers copies each on_value_change expression into
// lifecycle.triggers_state, keyed by the trimmed expression, for
// ResolveJobRunValueTriggerReferences to resolve. The expression itself is never
// resolved in place (see keepUnresolved), so the key stays stable when the value
// changes.
func SnapshotJobRunValueTriggers() bundle.Mutator {
	return &snapshotJobRunValueTriggers{}
}

func (*snapshotJobRunValueTriggers) Name() string {
	return "SnapshotJobRunValueTriggers"
}

func (*snapshotJobRunValueTriggers) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		for name, jr := range b.Config.Resources.JobRuns {
			if jr == nil || jr.Lifecycle == nil {
				continue
			}

			lifecyclePath := dyn.NewPath(dyn.Key("resources"), dyn.Key("job_runs"), dyn.Key(name), dyn.Key("lifecycle"))
			watches := make(map[string]dyn.Value)
			for i, trigger := range jr.Lifecycle.Triggers {
				if trigger.OnValueChange == nil {
					continue
				}

				path := lifecyclePath.Append(dyn.Key("triggers"), dyn.Index(i), dyn.Key("on_value_change"))
				expr := strings.TrimSpace(*trigger.OnValueChange)
				if expr == "" {
					diags = diags.Append(diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   "lifecycle.triggers.on_value_change must be non-empty when set",
						Locations: b.Config.GetLocations(path.String()),
					})
					continue
				}
				if _, ok := watches[expr]; ok {
					diags = diags.Append(diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   "lifecycle.triggers.on_value_change expressions must be unique",
						Locations: b.Config.GetLocations(path.String()),
					})
					continue
				}

				// The copy keeps the expression's location, so a diagnostic about its
				// resolved value points at the trigger.
				v, err := dyn.GetByPath(root, path)
				if err != nil {
					return dyn.InvalidValue, err
				}
				watches[expr] = dyn.NewValue(expr, v.Locations())
			}
			if len(watches) == 0 {
				continue
			}

			statePath := lifecyclePath.Append(dyn.Key("triggers_state"))
			state, err := dyn.GetByPath(root, statePath)
			if dyn.IsNoSuchKeyError(err) {
				state, err = dyn.V(map[string]dyn.Value{}), nil
			}
			if err != nil {
				return dyn.InvalidValue, err
			}
			state, err = dyn.Set(state, "on_value_change", dyn.V(watches))
			if err != nil {
				return dyn.InvalidValue, err
			}
			root, err = dyn.SetByPath(root, statePath, state)
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("snapshot on_value_change of job_runs.%s: %w", name, err)
			}
		}
		return root, nil
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}
	return diags
}
