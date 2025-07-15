package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
)

type mergeJobTasks struct{}

func MergeJobTasks() bundle.Mutator {
	return &mergeJobTasks{}
}

func (m *mergeJobTasks) Name() string {
	return "MergeJobTasks"
}

func (m *mergeJobTasks) taskKeyString(v dyn.Value) string {
	switch v.Kind() {
	case dyn.KindInvalid, dyn.KindNil:
		return ""
	case dyn.KindString:
		return v.MustString()
	default:
		panic("task key must be a string")
	}
}

func (m *mergeJobTasks) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		if v.Kind() == dyn.KindNil {
			return v, nil
		}

		return dyn.Map(v, "resources.jobs", dyn.Foreach(func(_ dyn.Path, job dyn.Value) (dyn.Value, error) {
			// Sorting keys here since it'll be sorted by TF anyway
			// https://github.com/databricks/terraform-provider-databricks/blob/0a932c2/jobs/resource_job.go#L343
			// However, if we don't sort we have a difference between direct and TF and between configs in
			// "bundle validate" and configs sent to backend.
			return dyn.Map(job, "tasks", merge.ElementsBySortedKey("task_key", m.taskKeyString))
		}))
	})

	return diag.FromErr(err)
}
