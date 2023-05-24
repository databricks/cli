package artifacts

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts/notebook"
	"github.com/databricks/cli/bundle/artifacts/wheel"
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

	if artifact.Notebook != nil {
		return bundle.Apply(ctx, b, notebook.Upload(m.name))
	}

	if artifact.PythonPackage != nil {
		return bundle.Apply(ctx, b, wheel.Upload(m.name))
	}

	return nil
}
