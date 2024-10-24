package libraries

import (
	"strings"

	"github.com/databricks/databricks-sdk-go/service/compute"
)

// IsWorkspacePath returns true if the specified path indicates that
// it should be interpreted as a Databricks Workspace path.
//
// The following paths are considered workspace paths:
//
// - /Workspace/Users/jane@doe.com/myfile
// - /Users/jane@doe.com/myfile
// - /Shared/project/myfile
//
// The following paths are not considered workspace paths:
//
// - myfile.txt
// - ./myfile.txt
// - ../myfile.txt
// - /foo/bar/myfile.txt
func IsWorkspacePath(path string) bool {
	return strings.HasPrefix(path, "/Workspace/") ||
		strings.HasPrefix(path, "/Users/") ||
		strings.HasPrefix(path, "/Shared/")
}

// IsWorkspaceLibrary returns true if the specified library refers to a workspace path.
func IsWorkspaceLibrary(library *compute.Library) bool {
	path, err := libraryPath(library)
	if err != nil {
		return false
	}

	return IsWorkspacePath(path)
}

// IsVolumesPath returns true if the specified path indicates that
// it should be interpreted as a Databricks Volumes path.
func IsVolumesPath(path string) bool {
	return strings.HasPrefix(path, "/Volumes/")
}

func IsWorkspaceSharedPath(path string) bool {
	return strings.HasPrefix(path, "/Workspace/Shared/")
}
