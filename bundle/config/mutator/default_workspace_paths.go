package mutator

import (
	"context"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
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

	// Reject an explicit workspace.file_path: immutable bundles set it automatically to
	// the content-addressed snapshot location. A user-supplied value would be silently
	// discarded during path translation, so we error early. The value is not empty only
	// if the user set it: the default is set below.
	if b.IsImmutableFolder() && b.Config.Workspace.FilePath != "" {
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   "workspace.file_path cannot be configured when experimental.immutable_folder is enabled",
			Locations: b.Config.GetLocationsOf(structpath.NewPath(nil, "workspace", "file_path")),
		}}
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
