package artifacts

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts/notebook"
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

func (m *build) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return nil, fmt.Errorf("artifact doesn't exist: %s", m.name)
	}

	if artifact.Notebook != nil {
		return []bundle.Mutator{notebook.Build(m.name)}, nil
	}

	return nil, nil
}
