package mutator

import (
	"context"
	"fmt"
	"path"

	"github.com/databricks/cli/bundle"
)

type defineDefaultWorkspacePaths struct{}

// DefineDefaultWorkspacePaths sets workspace paths if they aren't already set.
func DefineDefaultWorkspacePaths() bundle.Mutator {
	return &defineDefaultWorkspacePaths{}
}

func (m *defineDefaultWorkspacePaths) Name() string {
	return "DefaultWorkspacePaths"
}

func (m *defineDefaultWorkspacePaths) Apply(ctx context.Context, b *bundle.Bundle) error {
	root := b.Config.Workspace.RootPath
	if root == "" {
		return fmt.Errorf("unable to define default workspace paths: workspace root not defined")
	}

	if b.Config.Workspace.FilePath == "" {
		b.Config.Workspace.FilePath = path.Join(root, "files")
	}

	if b.Config.Workspace.ArtifactPath == "" {
		b.Config.Workspace.ArtifactPath = path.Join(root, "artifacts")
	}

	if b.Config.Workspace.StatePath == "" {
		b.Config.Workspace.StatePath = path.Join(root, "state")
	}

	return nil
}
