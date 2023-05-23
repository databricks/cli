package wheel

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/apierr"
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
	return fmt.Sprintf("wheel.Upload(%s)", m.name)
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	a, ok := b.Config.Artifacts[m.name]
	if !ok {
		return nil, fmt.Errorf("artifact doesn't exist: %s", m.name)
	}

	cmdio.LogString(ctx, fmt.Sprintf("Starting upload of Python wheel artifact: %s", m.name))

	artifact := a.PythonPackage
	raw, err := os.ReadFile(artifact.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s: %w", m.name, errors.Unwrap(err))
	}

	// Make sure target directory exists.
	err = b.WorkspaceClient().Workspace.MkdirsByPath(ctx, path.Dir(artifact.RemotePath))
	if err != nil {
		var apiErr *apierr.APIError
		if !errors.As(err, &apiErr) || apiErr.ErrorCode != "RESOURCE_ALREADY_EXISTS" {
			return nil, fmt.Errorf("unable to create directory for %s: %w", path.Dir(artifact.RemotePath), err)
		}
	}

	// Import to workspace.
	err = b.WorkspaceClient().Workspace.Import(ctx, workspace.Import{
		Path:      artifact.RemotePath,
		Overwrite: true,
		Format:    workspace.ExportFormatSource,
		Language:  workspace.LanguagePython,
		Content:   base64.StdEncoding.EncodeToString(raw),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to import %s: %w", m.name, err)
	}

	cmdio.LogString(ctx, fmt.Sprintf("Uploaded Python wheel artifact: %s. Artifact path: %s", m.name, artifact.RemotePath))
	return nil, nil
}
