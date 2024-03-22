package artifacts

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
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

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return diag.Errorf("artifact doesn't exist: %s", m.name)
	}

	if len(artifact.Files) == 0 {
		return diag.Errorf("artifact source is not configured: %s", m.name)
	}

	// Check if source paths are absolute, if not, make them absolute
	for k := range artifact.Files {
		f := &artifact.Files[k]
		if !filepath.IsAbs(f.Source) {
			dirPath := filepath.Dir(artifact.ConfigFilePath)
			f.Source = filepath.Join(dirPath, f.Source)
		}
	}

	// Expand any glob reference in files source path
	files := make([]config.ArtifactFile, 0, len(artifact.Files))
	for _, f := range artifact.Files {
		matches, err := filepath.Glob(f.Source)
		if err != nil {
			return diag.Errorf("unable to find files for %s: %v", f.Source, err)
		}

		if len(matches) == 0 {
			return diag.Errorf("no files found for %s", f.Source)
		}

		for _, match := range matches {
			files = append(files, config.ArtifactFile{
				Source: match,
			})
		}
	}

	artifact.Files = files
	return bundle.Apply(ctx, b, getUploadMutator(artifact.Type, m.name))
}

type cleanUp struct{}

func (m *cleanUp) Name() string {
	return "artifacts.CleanUp"
}

func (m *cleanUp) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	uploadPath, err := getUploadBasePath(b)
	if err != nil {
		return diag.FromErr(err)
	}

	b.WorkspaceClient().Workspace.Delete(ctx, workspace.Delete{
		Path:      uploadPath,
		Recursive: true,
	})

	err = b.WorkspaceClient().Workspace.MkdirsByPath(ctx, uploadPath)
	if err != nil {
		return diag.Errorf("unable to create directory for %s: %v", uploadPath, err)
	}

	return nil
}
