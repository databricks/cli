package whl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/python"
)

type build struct {
	name string
}

func Build(name string) bundle.Mutator {
	return &build{
		name: name,
	}
}

func (m *build) Name() string {
	return fmt.Sprintf("artifacts.whl.Build(%s)", m.name)
}

func (m *build) Apply(ctx context.Context, b *bundle.Bundle) error {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return fmt.Errorf("artifact doesn't exist: %s", m.name)
	}

	// TODO: If not set, BuildCommand should be infer prior to this
	// via a mutator so that it can be observable.
	if artifact.BuildCommand == "" {
		return fmt.Errorf("artifacts.whl.Build(%s): missing build property for the artifact", m.name)
	}

	cmdio.LogString(ctx, fmt.Sprintf("artifacts.whl.Build(%s): Building...", m.name))

	dir := artifact.Path

	distPath := filepath.Join(dir, "dist")
	os.RemoveAll(distPath)
	python.CleanupWheelFolder(dir)

	out, err := artifact.Build(ctx)
	if err != nil {
		return fmt.Errorf("artifacts.whl.Build(%s): Failed %w, output: %s", m.name, err, out)
	}
	cmdio.LogString(ctx, fmt.Sprintf("artifacts.whl.Build(%s): Build succeeded", m.name))

	wheels := python.FindFilesWithSuffixInPath(distPath, ".whl")
	if len(wheels) == 0 {
		return fmt.Errorf("artifacts.whl.Build(%s): cannot find built wheel in %s", m.name, dir)
	}
	for _, wheel := range wheels {
		artifact.Files = append(artifact.Files, config.ArtifactFile{
			Source: wheel,
		})
	}

	return nil
}
