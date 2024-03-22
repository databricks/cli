package mutator

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
	case dyn.KindNil:
		return ""
	case dyn.KindString:
		return v.MustString()
	default:
		panic("task key must be a string")
	}
}

func (m *mergeJobTasks) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	return b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		if v == dyn.NilValue {
			return v, nil
		}

		return dyn.Map(v, "resources.jobs", dyn.Foreach(func(_ dyn.Path, job dyn.Value) (dyn.Value, error) {
			return dyn.Map(job, "tasks", merge.ElementsByKey("task_key", m.taskKeyString))
		}))
	})
}
