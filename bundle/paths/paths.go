package paths

import (
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
)

func CollectUniquePaths(workspace config.Workspace) []string {
	rootPath := workspace.RootPath
	paths := []string{}
	if !libraries.IsVolumesPath(rootPath) && !libraries.IsWorkspaceSharedPath(rootPath) {
		paths = append(paths, rootPath)
	}

	if !strings.HasSuffix(rootPath, "/") {
		rootPath += "/"
	}

	for _, p := range []string{
		workspace.ArtifactPath,
		workspace.FilePath,
		workspace.StatePath,
		workspace.ResourcePath,
	} {
		if libraries.IsWorkspaceSharedPath(p) || libraries.IsVolumesPath(p) {
			continue
		}

		if strings.HasPrefix(p, rootPath) {
			continue
		}

		paths = append(paths, p)
	}

	return paths
}
