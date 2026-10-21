package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type prependWorkspacePrefix struct{}

// PrependWorkspacePrefix prepends the workspace root path to all paths in the bundle.
func PrependWorkspacePrefix() bundle.Mutator {
	return &prependWorkspacePrefix{}
}

func (m *prependWorkspacePrefix) Name() string {
	return "PrependWorkspacePrefix"
}

var skipPrefixes = []string{
	"/Workspace/",
	"/Volumes/",
}

func (m *prependWorkspacePrefix) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	w := &b.Config.Workspace
	for _, path := range []*string{
		&w.RootPath,
		&w.FilePath,
		&w.ArtifactPath,
		&w.StatePath,
		&w.ResourcePath,
	} {
		*path = prependWorkspacePrefixTo(*path)
	}

	return nil
}

func prependWorkspacePrefixTo(path string) string {
	// Skip prefixing if the path does not start with /, it might be variable reference or smth else.
	if !strings.HasPrefix(path, "/") {
		return path
	}

	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return path
		}
	}

	return "/Workspace" + path
}
