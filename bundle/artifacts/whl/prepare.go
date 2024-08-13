package whl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/python"
)

type prepare struct {
	name string
}

func Prepare(name string) bundle.Mutator {
	return &prepare{
		name: name,
	}
}

func (m *prepare) Name() string {
	return fmt.Sprintf("artifacts.whl.Prepare(%s)", m.name)
}

func (m *prepare) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return diag.Errorf("artifact doesn't exist: %s", m.name)
	}

	// If there is no build command for the artifact, we don't need to cleanup the dist folder before
	if artifact.BuildCommand == "" {
		return nil
	}

	dir := artifact.Path

	distPath := filepath.Join(dir, "dist")

	// If we have multiple artifacts con figured, prepare will be called multiple times
	// The first time we will remove the folders, other times will be no-op.
	err := os.RemoveAll(distPath)
	if err != nil {
		log.Infof(ctx, "Failed to remove dist folder: %v", err)
	}
	python.CleanupWheelFolder(dir)

	return nil
}
