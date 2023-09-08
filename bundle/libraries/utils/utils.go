package utils

import (
	"net/url"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func ReplacePath(lib *compute.Library, newPath string) {
	path := LibPath(lib)
	if path == nil {
		return
	}

	*path = newPath
}

func LibPath(library *compute.Library) *string {
	if library.Whl != "" {
		return &library.Whl
	}
	if library.Jar != "" {
		return &library.Jar
	}
	if library.Egg != "" {
		return &library.Egg
	}

	return nil
}

func IsTaskWithWorkspaceLibraries(task *jobs.Task) bool {
	for _, l := range task.Libraries {
		path := LibPath(&l)
		if path == nil {
			continue
		}
		if isWorkspacePath(*path) {
			return true
		}
	}

	return false
}

func IsTaskWithLocalLibraries(task *jobs.Task) bool {
	for _, l := range task.Libraries {
		if IsLocalLibrary(&l) {
			return true
		}
	}
	return false
}

func isWorkspacePath(path string) bool {
	return strings.HasPrefix(path, "/Workspace/") ||
		strings.HasPrefix(path, "/Users/") ||
		strings.HasPrefix(path, "/Shared/")
}

func IsLocalLibrary(library *compute.Library) bool {
	path := LibPath(library)
	if path == nil {
		return false
	}

	if isExplicitFileScheme(*path) {
		return true
	}

	if isRemoteStorageScheme(*path) {
		return false
	}

	return !isWorkspacePath(*path)
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
