package artifacts

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

func PrepareAll() bundle.Mutator {
	return &all{
		name: "Prepare",
		fn:   prepareArtifactByName,
	}
}

type prepare struct {
	name string
}

func prepareArtifactByName(name string) (bundle.Mutator, error) {
	return &prepare{name}, nil
}

func (m *prepare) Name() string {
	return fmt.Sprintf("artifacts.Prepare(%s)", m.name)
}

func (m *prepare) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return diag.Errorf("artifact doesn't exist: %s", m.name)
	}

	l := b.Config.GetLocation("artifacts." + m.name)
	dirPath := filepath.Dir(l.File)

	// Check if source paths are absolute, if not, make them absolute
	for k := range artifact.Files {
		f := &artifact.Files[k]
		if !filepath.IsAbs(f.Source) {
			f.Source = filepath.Join(dirPath, f.Source)
		}
	}

	// If artifact path is not provided, use bundle root dir
	if artifact.Path == "" {
		artifact.Path = b.BundleRootPath
	}

	if !filepath.IsAbs(artifact.Path) {
		artifact.Path = filepath.Join(dirPath, artifact.Path)
	}

	return bundle.Apply(ctx, b, getPrepareMutator(artifact.Type, m.name))
}
