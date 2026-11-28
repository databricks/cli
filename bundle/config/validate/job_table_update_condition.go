package validate

import (
	"context"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

// JobTableUpdateCondition errors when a job's table_update trigger monitors more than
// one table without a condition. The Jobs API rejects such a job with "Invalid condition:
// it must be one of ANY_UPDATED, ALL_UPDATED when more than one table is defined", which
// otherwise only surfaces during deploy, after the bundle files are uploaded.
func JobTableUpdateCondition() bundle.ReadOnlyMutator {
	return &jobTableUpdateCondition{}
}

type jobTableUpdateCondition struct{ bundle.RO }

func (v *jobTableUpdateCondition) Name() string {
	return "validate:job_table_update_condition"
}

func (v *jobTableUpdateCondition) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// Map iteration order is randomized; sort by key for stable output.
	for _, key := range slices.Sorted(maps.Keys(b.Config.Resources.Jobs)) {
		trigger := b.Config.Resources.Jobs[key].Trigger
		if trigger == nil || trigger.TableUpdate == nil {
			continue
		}

		// A ${resources.*} reference is still unresolved here and is not flagged: as
		// table_names it leaves the typed slice empty, as condition it is a non-empty string.
		if len(trigger.TableUpdate.TableNames) <= 1 || trigger.TableUpdate.Condition != "" {
			continue
		}

		path := structpath.NewPath(nil, "resources", "jobs", key, "trigger", "table_update")
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "trigger.table_update.condition is required when table_names has more than one table",
			Detail:    "Set condition to ANY_UPDATED or ALL_UPDATED.",
			Locations: b.Config.GetLocationsOf(path),
			Paths:     []*structpath.PathNode{path},
		})
	}

	return diags
}
