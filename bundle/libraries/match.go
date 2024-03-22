package libraries

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type match struct {
}

func MatchWithArtifacts() bundle.Mutator {
	return &match{}
}

func (a *match) Name() string {
	return "libraries.MatchWithArtifacts"
}

func (a *match) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	tasks := findAllTasks(b)
	for _, task := range tasks {
		if isMissingRequiredLibraries(task) {
			return diag.Errorf("task '%s' is missing required libraries. Please include your package code in task libraries block", task.TaskKey)
		}
		for j := range task.Libraries {
			lib := &task.Libraries[j]
			_, err := findArtifactFiles(ctx, lib, b)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}
	return nil
}

func isMissingRequiredLibraries(task *jobs.Task) bool {
	if task.Libraries != nil {
		return false
	}

	return task.PythonWheelTask != nil || task.SparkJarTask != nil
}
