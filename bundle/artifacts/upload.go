package artifacts

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts/notebook"
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

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return nil, fmt.Errorf("artifact doesn't exist: %s", m.name)
	}

	if artifact.Notebook != nil {
		return []bundle.Mutator{notebook.Upload(m.name)}, nil
	}

	return nil, nil
}
