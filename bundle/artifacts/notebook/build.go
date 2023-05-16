package notebook

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type build struct {
	name string
}

func Build(name string) bundle.Mutator {
	return &build{
		name: name,
	}
}

func (m *build) Name() string {
	return fmt.Sprintf("notebook.Build(%s)", m.name)
}

func (m *build) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	a, ok := b.Config.Artifacts[m.name]
	if !ok {
		return nil, fmt.Errorf("artifact doesn't exist: %s", m.name)
	}

	artifact := a.Notebook

	// Check if the filetype is supported.
	switch ext := strings.ToLower(filepath.Ext(artifact.Path)); ext {
	case ".py":
		artifact.Language = workspace.LanguagePython
	case ".scala":
		artifact.Language = workspace.LanguageScala
	case ".sql":
		artifact.Language = workspace.LanguageSql
	default:
		return nil, fmt.Errorf("invalid notebook extension: %s", ext)
	}

	// Open underlying file.
	f, err := os.Open(filepath.Join(b.Config.Path, artifact.Path))
	if err != nil {
		return nil, fmt.Errorf("unable to open artifact file %s: %w", artifact.Path, errors.Unwrap(err))
	}
	defer f.Close()

	// Check that the file contains the notebook marker on its first line.
	ok, err = hasMarker(artifact.Language, f)
	if err != nil {
		return nil, fmt.Errorf("unable to read artifact file %s: %s", artifact.Path, errors.Unwrap(err))
	}
	if !ok {
		return nil, fmt.Errorf("notebook marker not found in %s", artifact.Path)
	}

	// Check that an artifact path is defined.
	remotePath := b.Config.Workspace.ArtifactsPath
	if remotePath == "" {
		return nil, fmt.Errorf("remote artifact path not configured")
	}

	// Store absolute paths.
	artifact.LocalPath = filepath.Join(b.Config.Path, artifact.Path)
	artifact.RemotePath = path.Join(remotePath, stripExtension(artifact.Path))
	return nil, nil
}

func stripExtension(path string) string {
	ext := filepath.Ext(path)
	return path[0 : len(path)-len(ext)]
}
