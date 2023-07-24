package artifacts

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func UploadAll() bundle.Mutator {
	return &all{
		name: "Upload",
		fn:   uploadArtifactByName,
	}
}

func CleanUp() bundle.Mutator {
	return &cleanUp{}
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

	if len(artifact.Files) == 0 {
		return fmt.Errorf("artifact source is not configured: %s", m.name)
	}

	return bundle.Apply(ctx, b, getUploadMutator(artifact.Type, m.name))
}

type cleanUp struct{}

func (m *cleanUp) Name() string {
	return "artifacts.CleanUp"
}

func (m *cleanUp) Apply(ctx context.Context, b *bundle.Bundle) error {
	artifactPath := b.Config.Workspace.ArtifactsPath
	b.WorkspaceClient().Workspace.Delete(ctx, workspace.Delete{
		Path:      artifactPath,
		Recursive: true,
	})

	err := b.WorkspaceClient().Workspace.MkdirsByPath(ctx, artifactPath)
	if err != nil {
		return fmt.Errorf("unable to create directory for %s: %w", artifactPath, err)
	}

	return nil
}
