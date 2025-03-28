package paths

import (
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
)

func CollectUniqueWorkspacePathPrefixes(workspace config.Workspace) []string {
	rootPath := workspace.RootPath
	var paths []string
	if !libraries.IsVolumesPath(rootPath) {
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
		if libraries.IsVolumesPath(p) {
			continue
		}

		if strings.HasPrefix(p, rootPath) {
			continue
		}

		paths = append(paths, p)
	}

	return paths
}
