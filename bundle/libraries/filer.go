package libraries

import (
	"context"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
)

// We upload artifacts to the workspace in a directory named ".internal" to have
// a well defined location for artifacts that have been uploaded by the DABs.
const InternalDirName = ".internal"

// This function returns a filer for uploading artifacts to the configured location.
// Supported locations:
// 1. WSFS
// 2. UC volumes
func GetFilerForLibraries(ctx context.Context, b *bundle.Bundle) (filer.Filer, string, diag.Diagnostics) {
	artifactPath := b.Config.Workspace.ArtifactPath
	if artifactPath == "" {
		return nil, "", diag.Errorf("remote artifact path not configured")
	}

	uploadPath := path.Join(b.Config.Workspace.ArtifactPath, InternalDirName)

	switch {
	case IsVolumesPath(artifactPath):
		return filerForVolume(b, uploadPath)

	default:
		return filerForWorkspace(b, uploadPath)
	}
}

func GetFilerForLibrariesCleanup(ctx context.Context, b *bundle.Bundle) (filer.Filer, string, diag.Diagnostics) {
	artifactPath := b.Config.Workspace.ArtifactPath
	if artifactPath == "" {
		return nil, "", diag.Errorf("remote artifact path not configured")
	}

	switch {
	case IsVolumesPath(artifactPath):
		return filerForVolume(b, b.Config.Workspace.ArtifactPath)

	default:
		return filerForWorkspace(b, b.Config.Workspace.ArtifactPath)
	}
}
