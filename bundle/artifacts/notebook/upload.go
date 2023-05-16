package notebook

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type upload struct {
	name string
}

func Upload(name string) bundle.Mutator {
	return &upload{
		name: name,
	}
}

func (m *upload) Name() string {
	return fmt.Sprintf("notebook.Upload(%s)", m.name)
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	a, ok := b.Config.Artifacts[m.name]
	if !ok {
		return nil, fmt.Errorf("artifact doesn't exist: %s", m.name)
	}

	artifact := a.Notebook
	raw, err := os.ReadFile(artifact.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s: %w", m.name, errors.Unwrap(err))
	}

	// Make sure target directory exists.
	err = b.WorkspaceClient().Workspace.MkdirsByPath(ctx, path.Dir(artifact.RemotePath))
	if err != nil {
		return nil, fmt.Errorf("unable to create directory for %s: %w", m.name, err)
	}

	// Import to workspace.
	err = b.WorkspaceClient().Workspace.Import(ctx, workspace.Import{
		Path:      artifact.RemotePath,
		Overwrite: true,
		Format:    workspace.ExportFormatSource,
		Language:  artifact.Language,
		Content:   base64.StdEncoding.EncodeToString(raw),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to import %s: %w", m.name, err)
	}

	return nil, nil
}
