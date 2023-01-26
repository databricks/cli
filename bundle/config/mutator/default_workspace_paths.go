package mutator

import (
	"context"
	"fmt"
	"path"

	"github.com/databricks/bricks/bundle"
)

type defineDefaultWorkspacePaths struct{}

// DefineDefaultWorkspacePaths sets workspace paths if they aren't already set.
func DefineDefaultWorkspacePaths() bundle.Mutator {
	return &defineDefaultWorkspacePaths{}
}

func (m *defineDefaultWorkspacePaths) Name() string {
	return "DefaultWorkspacePaths"
}

func (m *defineDefaultWorkspacePaths) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	root := b.Config.Workspace.Root
	if root == "" {
		return nil, fmt.Errorf("unable to define default workspace paths: workspace root not defined")
	}

	if !b.Config.Workspace.FilePath.IsSet() {
		b.Config.Workspace.FilePath.Workspace = path.Join(root, "files")
	}

	if !b.Config.Workspace.ArtifactPath.IsSet() {
		b.Config.Workspace.ArtifactPath.Workspace = path.Join(root, "artifacts")
	}

	if !b.Config.Workspace.StatePath.IsSet() {
		b.Config.Workspace.StatePath.Workspace = path.Join(root, "state")
	}

	return nil, nil
}
