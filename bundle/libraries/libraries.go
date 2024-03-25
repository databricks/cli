package libraries

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

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
		if task.PythonWheelTask != nil && IsTaskWithLocalLibraries(task) {
			wheelTasks = append(wheelTasks, task)
		}
	}

	return wheelTasks
}

func IsTaskWithLocalLibraries(task *jobs.Task) bool {
	for _, l := range task.Libraries {
		if IsLocalLibrary(&l) {
			return true
		}
	}

	return false
}

func IsTaskWithWorkspaceLibraries(task *jobs.Task) bool {
	for _, l := range task.Libraries {
		if IsWorkspaceLibrary(&l) {
			return true
		}
	}

	return false
}

func findLibraryMatches(lib *compute.Library, b *bundle.Bundle) ([]string, error) {
	path := libraryPath(lib)
	if path == "" {
		return nil, nil
	}

	fullPath := filepath.Join(b.Config.Path, path)
	return filepath.Glob(fullPath)
}

func findArtifactFiles(ctx context.Context, lib *compute.Library, b *bundle.Bundle) ([]*config.ArtifactFile, error) {
	matches, err := findLibraryMatches(lib, b)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 && IsLocalLibrary(lib) {
		return nil, fmt.Errorf("file %s is referenced in libraries section but doesn't exist on the local file system", libraryPath(lib))
	}

	var out []*config.ArtifactFile
	for _, match := range matches {
		af, err := findArtifactFileByLocalPath(match, b)
		if err != nil {
			cmdio.LogString(ctx, fmt.Sprintf("%s. Skipping uploading. In order to use the define 'artifacts' section", err.Error()))
		} else {
			out = append(out, af)
		}
	}

	return out, nil
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

func MapFilesToTaskLibraries(ctx context.Context, b *bundle.Bundle) map[string][]*compute.Library {
	tasks := findAllTasks(b)
	out := make(map[string][]*compute.Library)
	for _, task := range tasks {
		for j := range task.Libraries {
			lib := &task.Libraries[j]
			if !IsLocalLibrary(lib) {
				continue
			}

			matches, err := findLibraryMatches(lib, b)
			if err != nil {
				log.Warnf(ctx, "Error matching library to files: %s", err.Error())
				continue
			}

			for _, match := range matches {
				out[match] = append(out[match], lib)
			}
		}
	}

	return out
}
