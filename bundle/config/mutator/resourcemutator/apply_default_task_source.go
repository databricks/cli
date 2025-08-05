package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type applyDefaultTaskSource struct{}

func ApplyDefaultTaskSource() bundle.Mutator {
	return &applyDefaultTaskSource{}
}

func (a *applyDefaultTaskSource) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("jobs"),
		dyn.AnyKey(),
	)

	taskPattern := dyn.NewPattern(
		dyn.Key("tasks"),
		dyn.AnyIndex(),
	)

	foreachTaskPattern := dyn.NewPattern(
		dyn.Key("for_each_task"),
		dyn.Key("task"),
	)

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(v, pattern, func(p dyn.Path, job dyn.Value) (dyn.Value, error) {
			defaultSource := "WORKSPACE"

			// Check if the job has git_source set and set the default source to GIT if it does
			_, err := dyn.Get(job, "git_source")
			if err == nil {
				defaultSource = "GIT"
			}

			// Then iterate over the tasks and set the source to the default if it's not set
			return dyn.MapByPattern(job, taskPattern, func(p dyn.Path, task dyn.Value) (dyn.Value, error) {
				// Then iterate over the foreach tasks and set the source to the default if it's not set
				task, err = dyn.MapByPattern(task, foreachTaskPattern, func(p dyn.Path, foreachTask dyn.Value) (dyn.Value, error) {
					return setSourceIfNotSet(foreachTask, defaultSource)
				})
				if err != nil {
					return task, err
				}

				return setSourceIfNotSet(task, defaultSource)
			})
		})
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// These are the task types that support the source field
// https://docs.databricks.com/api/workspace/jobs/create
var supportedTypeTasks = []string{
	"db_task",
	"notebook_task",
	"spark_python_task",
}

func setSourceIfNotSet(task dyn.Value, defaultSource string) (dyn.Value, error) {
	for _, taskType := range supportedTypeTasks {
		t, err := dyn.Get(task, taskType)
		if err != nil {
			continue
		}

		_, err = dyn.Get(t, "source")
		if err != nil {
			return dyn.Set(task, taskType+".source", dyn.V(defaultSource))
		}
	}
	return task, nil
}

func (a *applyDefaultTaskSource) Name() string {
	return "ApplyDefaultTaskSource"
}
