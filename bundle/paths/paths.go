package paths

import (
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
)

// WorkspacePaths holds the unique workspace folders that bundle permissions are applied
// to. A logical bundle path (artifact, file, state, resource) nested under another is
// represented only by its enclosing folder, so permissions are applied once and the
// nested paths inherit them.
type WorkspacePaths struct {
	Paths []string
}

func CollectUniqueWorkspacePathPrefixes(workspace config.Workspace) WorkspacePaths {
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

	return WorkspacePaths{Paths: paths}
}

// Governing returns the path whose ACL governs the given path: the path itself when it
// is one of the collected paths, or the enclosing path when it is nested under one (e.g.
// the state path nested under the root path, which is deduplicated out of Paths).
// Returns an empty string when no path governs it, e.g. a Volumes path that permissions
// don't apply to. Paths are /Workspace-normalized by PrependWorkspacePrefix.
func (w WorkspacePaths) Governing(path string) string {
	for _, p := range w.Paths {
		trimmed := strings.TrimSuffix(p, "/")
		if path == trimmed || strings.HasPrefix(path, trimmed+"/") {
			return p
		}
	}
	return ""
}
