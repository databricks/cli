package validate

import (
	"context"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
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
		if job == nil {
			continue
		}
		if job.Trigger != nil && job.Trigger.TableUpdate != nil {
			path := structpath.NewPath(nil, "resources", "jobs", name, "trigger", "table_update")
			diags = diags.Extend(checkTableUpdateCondition(b, job.Trigger.TableUpdate, path))
		}
		// The Jobs API applies the same condition requirement to each entry
		// of the plural `triggers` list.
		for i := range job.Triggers {
			if job.Triggers[i].TableUpdate == nil {
				continue
			}
			path := structpath.NewPath(nil, "resources", "jobs", name, "triggers", i, "table_update")
			diags = diags.Extend(checkTableUpdateCondition(b, job.Triggers[i].TableUpdate, path))
		}
	}

	return diags
}

func checkTableUpdateCondition(b *bundle.Bundle, tableUpdate *jobs.TableUpdateTriggerConfiguration, path *structpath.PathNode) diag.Diagnostics {
	// The Jobs API rejects the trigger when more than one table is listed
	// without a condition, so fail validation instead of failing at deploy.
	if len(tableUpdate.TableNames) <= 1 || tableUpdate.Condition != "" {
		return nil
	}
	// The condition field is missing from configuration, so point at the
	// table_update block that holds a location.
	return diag.Diagnostics{{
		Severity:  diag.Error,
		Summary:   "trigger.table_update.condition is required when table_names lists more than one table",
		Detail:    "Set condition to ANY_UPDATED or ALL_UPDATED",
		Paths:     []*structpath.PathNode{path},
		Locations: b.Config.GetLocationsOf(path),
	}}
}
