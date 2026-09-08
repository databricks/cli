package mutator

import (
	"context"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
)

type defineDefaultWorkspacePaths struct{}

// DefineDefaultWorkspacePaths sets workspace paths if they aren't already set.
func DefineDefaultWorkspacePaths() bundle.Mutator {
	return &defineDefaultWorkspacePaths{}
}

func (m *defineDefaultWorkspacePaths) Name() string {
	return "DefaultWorkspacePaths"
}

func (m *defineDefaultWorkspacePaths) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	root := b.Config.Workspace.RootPath
	if root == "" {
		return diag.Errorf("unable to define default workspace paths: workspace root not defined")
	}

	if b.Config.Workspace.FilePath == "" {
		if b.IsImmutableFolder() {
			b.Config.Workspace.FilePath = resources.SnapshotFullPathRef + "/files"
		} else {
			b.Config.Workspace.FilePath = path.Join(root, "files")
		}
	}

	if b.Config.Workspace.ResourcePath == "" {
		b.Config.Workspace.ResourcePath = path.Join(root, "resources")
	}

	if b.Config.Workspace.ArtifactPath == "" {
		if b.IsImmutableFolder() {
			b.Config.Workspace.ArtifactPath = resources.SnapshotFullPathRef + "/artifacts"
		} else {
			b.Config.Workspace.ArtifactPath = path.Join(root, "artifacts")
		}
	}

	if b.Config.Workspace.StatePath == "" {
		b.Config.Workspace.StatePath = path.Join(root, "state")
	}

	return nil
}
