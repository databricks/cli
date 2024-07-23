package whl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
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

	dir := artifact.Path

	distPath := filepath.Join(dir, "dist")
	os.RemoveAll(distPath)
	python.CleanupWheelFolder(dir)

	return nil
}
