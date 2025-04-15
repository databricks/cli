package artifacts

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/python"
	"github.com/databricks/cli/libs/utils"
)

func cleanupPythonDistBuild(ctx context.Context, b *bundle.Bundle) {
	removeFolders := make(map[string]bool, len(b.Config.Artifacts))
	cleanupWheelFolders := make(map[string]bool, len(b.Config.Artifacts))

	for _, artifactName := range utils.SortedKeys(b.Config.Artifacts) {
		artifact := b.Config.Artifacts[artifactName]
		if artifact.Type == "whl" && artifact.BuildCommand != "" {
			dir := artifact.Path
			removeFolders[filepath.Join(dir, "dist")] = true
			cleanupWheelFolders[dir] = true
		}
	}

	for _, dir := range utils.SortedKeys(removeFolders) {
		err := os.RemoveAll(dir)
		if err != nil {
			log.Infof(ctx, "Failed to remove %s: %s", dir, err)
		}
	}

	for _, dir := range utils.SortedKeys(cleanupWheelFolders) {
		log.Infof(ctx, "Cleaning up Python build artifacts in %s", dir)
		python.CleanupWheelFolder(dir)
	}
}
