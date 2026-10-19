package validate

import (
	"context"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

func JobTableUpdateCondition() bundle.ReadOnlyMutator {
	return &jobTableUpdateCondition{}
}

type jobTableUpdateCondition struct{ bundle.RO }

func (v *jobTableUpdateCondition) Name() string {
	return "validate:job_table_update_condition"
}

func (v *jobTableUpdateCondition) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// Map iteration order is randomized; sort by name for stable output.
	for _, name := range slices.Sorted(maps.Keys(b.Config.Resources.Jobs)) {
		job := b.Config.Resources.Jobs[name]
		// An empty `jobs.<name>:` entry loads as a present key with a nil value.
		if job == nil || job.Trigger == nil || job.Trigger.TableUpdate == nil {
			continue
		}
		tableUpdate := job.Trigger.TableUpdate
		// The Jobs API rejects the trigger when more than one table is listed
		// without a condition, so fail validation instead of failing at deploy.
		if len(tableUpdate.TableNames) <= 1 || tableUpdate.Condition != "" {
			continue
		}
		// The condition field is missing from configuration, so point at the
		// table_update block that holds a location.
		path := structpath.NewPath(nil, "resources", "jobs", name, "trigger", "table_update")
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "trigger.table_update.condition is required when table_names lists more than one table",
			Detail:    "Set condition to ANY_UPDATED or ALL_UPDATED",
			Paths:     []*structpath.PathNode{path},
			Locations: b.Config.GetLocationsOf(path),
		})
	}

	return diags
}
