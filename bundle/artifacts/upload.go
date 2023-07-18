package artifacts

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
)

func UploadAll() bundle.Mutator {
	return &all{
		name: "Upload",
		fn:   uploadArtifactByName,
	}
}

type upload struct {
	name string
}

func uploadArtifactByName(name string) (bundle.Mutator, error) {
	return &upload{name}, nil
}

func (m *upload) Name() string {
	return fmt.Sprintf("artifacts.Upload(%s)", m.name)
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) error {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return fmt.Errorf("artifact doesn't exist: %s", m.name)
	}

	if artifact.File == "" {
		return fmt.Errorf("artifact source is not configured: %s", m.name)
	}

	return bundle.Apply(ctx, b, getUploadMutator(artifact.Type, m.name))
}
