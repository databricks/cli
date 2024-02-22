package libraries

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
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

func (a *match) Apply(ctx context.Context, b *bundle.Bundle) error {
	tasks := findAllTasks(b)
	for _, task := range tasks {
		if isMissingRequiredLibraries(task) {
			return fmt.Errorf("task '%s' is missing required libraries. Please include your package code in task libraries block", task.TaskKey)
		}
		for j := range task.Libraries {
			lib := &task.Libraries[j]
			_, err := findArtifactFiles(ctx, lib, b)
			if err != nil {
				return err
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
