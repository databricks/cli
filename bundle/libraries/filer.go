package libraries

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
)

// This function returns the right filer to use, to upload artifacts to the configured location.
// Supported locations:
// 1. WSFS
// 2. UC volumes
func GetFilerForLibraries(ctx context.Context, b *bundle.Bundle) (filer.Filer, string, diag.Diagnostics) {
	artifactPath := b.Config.Workspace.ArtifactPath
	if artifactPath == "" {
		return nil, "", diag.Errorf("remote artifact path not configured")
	}

	switch {
	case strings.HasPrefix(artifactPath, "/Volumes/"):
		return filerForVolume(ctx, b)

	default:
		return filerForWorkspace(b)
	}
}
