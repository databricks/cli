package artifacts

import (
	"context"
	"fmt"
	"os"

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

	if artifact.File == "" && artifact.BuildCommand == "" {
		return fmt.Errorf("artifact %s misconfigured: 'file' or 'build' property is requried", m.name)
	}

	// If artifact file is explicitly defined, skip building the artifact
	if artifact.File != "" {
		return nil
	}

	// If artifact path is not provided, use current dir
	if artifact.Path == "" {
		path, err := os.Getwd()
		if err != nil {
			return nil
		}
		artifact.Path = path
	}

	return bundle.Apply(ctx, b, getBuildMutator(artifact.Type, m.name))
}
