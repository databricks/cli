package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// JobTableUpdateTriggerCondition validates that a table update trigger that watches
// more than one table sets a condition. The Jobs API rejects such a trigger on create,
// which would otherwise only surface during deploy.
func JobTableUpdateTriggerCondition() bundle.ReadOnlyMutator {
	return &jobTableUpdateTriggerCondition{}
}

type jobTableUpdateTriggerCondition struct{ bundle.RO }

func (v *jobTableUpdateTriggerCondition) Name() string {
	return "validate:job_table_update_trigger_condition"
}

func (v *jobTableUpdateTriggerCondition) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for k, job := range b.Config.Resources.Jobs {
		if job.Trigger != nil {
			diags = diags.Extend(checkTableUpdateCondition(b, job.Trigger.TableUpdate,
				structpath.NewPath(nil, "resources", "jobs", k, "trigger", "table_update")))
		}
		for i, trigger := range job.Triggers {
			diags = diags.Extend(checkTableUpdateCondition(b, trigger.TableUpdate,
				structpath.NewPath(nil, "resources", "jobs", k, "triggers", i, "table_update")))
		}
	}

	return diags
}

func checkTableUpdateCondition(b *bundle.Bundle, tableUpdate *jobs.TableUpdateTriggerConfiguration, path *structpath.PathNode) diag.Diagnostics {
	if tableUpdate == nil || len(tableUpdate.TableNames) <= 1 || tableUpdate.Condition != "" {
		return nil
	}

	return diag.Diagnostics{{
		Severity:  diag.Error,
		Summary:   "Missing condition for table update trigger with multiple tables",
		Detail:    "A table update trigger that lists more than one table must set condition to ANY_UPDATED or ALL_UPDATED.",
		Locations: b.Config.GetLocationsOf(path),
		Paths:     []*structpath.PathNode{path},
	}}
}
