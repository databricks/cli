package libraries

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
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
	r := b.Config.Resources
	for k := range b.Config.Resources.Jobs {
		tasks := r.Jobs[k].JobSettings.Tasks
		for i := range tasks {
			task := &tasks[i]
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
	}
	return nil
}

func isMissingRequiredLibraries(task *jobs.Task) bool {
	if task.Libraries != nil {
		return false
	}

	return task.PythonWheelTask != nil || task.SparkJarTask != nil
}

func findLibraryMatches(lib *compute.Library, b *bundle.Bundle) ([]string, error) {
	path := libPath(lib)
	if path == "" {
		return nil, nil
	}

	fullPath := filepath.Join(b.Config.Path, path)
	return filepath.Glob(fullPath)
}

func findArtifactsAndMarkForUpload(ctx context.Context, lib *compute.Library, b *bundle.Bundle) error {
	matches, err := findLibraryMatches(lib, b)
	if err != nil {
		return err
	}

	for _, match := range matches {
		af, err := findArtifactFileByLocalPath(match, b)
		if err != nil {
			cmdio.LogString(ctx, fmt.Sprintf("%s. Skipping %s. In order to use the library upload it manually", err.Error(), match))
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

	return nil, fmt.Errorf("artifact file is not found for path %s", path)
}

func libPath(library *compute.Library) string {
	if library.Whl != "" {
		return library.Whl
	}
	if library.Jar != "" {
		return library.Jar
	}
	if library.Egg != "" {
		return library.Egg
	}

	return ""
}
