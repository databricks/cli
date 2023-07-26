package artifacts

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
)

func BuildAll() bundle.Mutator {
	return &all{
		name: "Build",
		fn:   buildArtifactByName,
	}
}

type build struct {
	name string
}

func buildArtifactByName(name string) (bundle.Mutator, error) {
	return &build{name}, nil
}

func (m *build) Name() string {
	return fmt.Sprintf("artifacts.Build(%s)", m.name)
}

func (m *build) Apply(ctx context.Context, b *bundle.Bundle) error {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return fmt.Errorf("artifact doesn't exist: %s", m.name)
	}

	// Skip building if build command is not specified or infered
	if artifact.BuildCommand == "" {
		// If no build command was specified or infered and there is no
		// artifact output files specified, artifact is misconfigured
		if len(artifact.Files) == 0 {
			return fmt.Errorf("misconfigured artifact: please specify 'build' or 'files' property")
		}
		return nil
	}

	// If artifact path is not provided, use bundle root dir
	if artifact.Path == "" {
		artifact.Path = b.Config.Path
	}

	if !filepath.IsAbs(artifact.Path) {
		artifact.Path = filepath.Join(b.Config.Path, artifact.Path)
	}

	return bundle.Apply(ctx, b, getBuildMutator(artifact.Type, m.name))
}
