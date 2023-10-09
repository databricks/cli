package libraries

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

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
		if task.PythonWheelTask != nil && IsTaskWithLocalLibraries(task) {
			wheelTasks = append(wheelTasks, task)
		}
	}

	return wheelTasks
}

func IsTaskWithLocalLibraries(task *jobs.Task) bool {
	for _, l := range task.Libraries {
		if isLocalLibrary(&l) {
			return true
		}
	}

	return false
}

func IsTaskWithWorkspaceLibraries(task *jobs.Task) bool {
	for _, l := range task.Libraries {
		path := libPath(&l)
		if isWorkspacePath(path) {
			return true
		}
	}

	return false
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

	if len(matches) == 0 && isLocalLibrary(lib) {
		return fmt.Errorf("file %s is referenced in libraries section but doesn't exist on the local file system", libPath(lib))
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

func isLocalLibrary(library *compute.Library) bool {
	path := libPath(library)
	if path == "" {
		return false
	}

	return IsLocalPath(path)
}

func IsLocalPath(path string) bool {
	if isExplicitFileScheme(path) {
		return true
	}

	if isRemoteStorageScheme(path) {
		return false
	}

	if isAbsoluteRemotePath(path) {
		return false
	}

	return !isWorkspacePath(path) && !isReposPath(path)
}

func isExplicitFileScheme(path string) bool {
	return strings.HasPrefix(path, "file://")
}

func isRemoteStorageScheme(path string) bool {
	url, err := url.Parse(path)
	if err != nil {
		return false
	}

	if url.Scheme == "" {
		return false
	}

	// If the path starts with scheme:/ format, it's a correct remote storage scheme
	return strings.HasPrefix(path, url.Scheme+":/")

}

func isWorkspacePath(path string) bool {
	return strings.HasPrefix(path, "/Workspace/") ||
		strings.HasPrefix(path, "/Users/") ||
		strings.HasPrefix(path, "/Shared/")
}

func isReposPath(path string) bool {
	return strings.HasPrefix(path, "/Repos/")
}

func isAbsoluteRemotePath(p string) bool {
	// If path for library starts with /, it's a remote absolute path
	return path.IsAbs(p)
}
