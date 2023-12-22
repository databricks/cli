package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
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

// mergeJobTasks merges tasks with the same key.
// The tasks field is a slice, and as such, overrides are appended to it.
// We can identify a task by its task key, however, so we can use this key
// to figure out which definitions are actually overrides and merge them.
func (m *mergeJobTasks) mergeJobTasks(v dyn.Value) (dyn.Value, error) {
	// We know the type of this value is a sequence.
	// For additional defence, return self if it is not.
	tasks, ok := v.AsSequence()
	if !ok {
		return v, nil
	}

	seen := make(map[string]dyn.Value, len(tasks))
	keys := make([]string, 0, len(tasks))

	// Target overrides are always appended, so we can iterate in natural order to
	// first find the base definition, and merge instances we encounter later.
	for i := range tasks {
		var key string

		// Get task key if present.
		kv := tasks[i].Get("task_key")
		if kv.Kind() == dyn.KindString {
			key = kv.MustString()
		}

		// Register task with key if not yet seen before.
		ref, ok := seen[key]
		if !ok {
			keys = append(keys, key)
			seen[key] = tasks[i]
			continue
		}

		// Merge this instance into the reference.
		nv, err := merge.Merge(ref, tasks[i])
		if err != nil {
			return v, err
		}

		// Overwrite reference.
		seen[key] = nv
	}

	// Gather resulting clusters in natural order.
	out := make([]dyn.Value, 0, len(keys))
	for _, key := range keys {
		out = append(out, seen[key])
	}

	return dyn.NewValue(out, v.Location()), nil

}

func (m *mergeJobTasks) foreachJob(v dyn.Value) (dyn.Value, error) {
	jobs, ok := v.AsMap()
	if !ok {
		return v, nil
	}

	out := make(map[string]dyn.Value)
	for key, job := range jobs {
		var err error
		out[key], err = job.Transform("tasks", m.mergeJobTasks)
		if err != nil {
			return v, err
		}
	}

	return dyn.NewValue(out, v.Location()), nil
}

func (m *mergeJobTasks) Apply(ctx context.Context, b *bundle.Bundle) error {
	return b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		if v == dyn.NilValue {
			return v, nil
		}

		nv, err := v.Transform("resources.jobs", m.foreachJob)

		// It is not a problem if the pipelines key is not set.
		if dyn.IsNoSuchKeyError(err) {
			return v, nil
		}

		if err != nil {
			return v, err
		}

		return nv, nil
	})
}
