package libraries

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries/utils"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/compute"
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
			err := findArtifactsAndMarkForUpload(ctx, lib, b)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func findAllTasks(b *bundle.Bundle) []*jobs.Task {
	r := b.Config.Resources
	result := make([]*jobs.Task, 0)
	for k := range b.Config.Resources.Jobs {
		tasks := r.Jobs[k].JobSettings.Tasks
		for i := range tasks {
			task := &tasks[i]
			result = append(result, task)
		}
	}

	return result
}

func FindAllWheelTasksWithLocalLibraries(b *bundle.Bundle) []*jobs.Task {
	tasks := findAllTasks(b)
	wheelTasks := make([]*jobs.Task, 0)
	for _, task := range tasks {
		if task.PythonWheelTask != nil && utils.IsTaskWithLocalLibraries(task) {
			wheelTasks = append(wheelTasks, task)
		}
	}

	return wheelTasks
}

func isMissingRequiredLibraries(task *jobs.Task) bool {
	if task.Libraries != nil {
		return false
	}

	return task.PythonWheelTask != nil || task.SparkJarTask != nil
}

func findLibraryMatches(lib *compute.Library, b *bundle.Bundle) ([]string, error) {
	path := utils.LibPath(lib)
	if path == nil {
		return nil, nil
	}

	fullPath := filepath.Join(b.Config.Path, *path)
	return filepath.Glob(fullPath)
}

func findArtifactsAndMarkForUpload(ctx context.Context, lib *compute.Library, b *bundle.Bundle) error {
	matches, err := findLibraryMatches(lib, b)
	if err != nil {
		return err
	}

	if len(matches) == 0 && utils.IsLocalLibrary(lib) {
		return fmt.Errorf("file %s is referenced in libraries section but doesn't exist on the local file system", utils.LibPath(lib))
	}

	for _, match := range matches {
		af, err := findArtifactFileByLocalPath(match, b)
		if err != nil {
			cmdio.LogString(ctx, fmt.Sprintf("%s. Skipping uploading. In order to use the define 'artifacts' section", err.Error()))
		} else {
			af.Libraries = append(af.Libraries, lib)
		}
	}

	return nil
}

func findArtifactFileByLocalPath(path string, b *bundle.Bundle) (*config.ArtifactFile, error) {
	for _, a := range b.Config.Artifacts {
		for k := range a.Files {
			if a.Files[k].Source == path {
				return &a.Files[k], nil
			}
		}
	}

	return nil, fmt.Errorf("artifact section is not defined for file at %s", path)
}
