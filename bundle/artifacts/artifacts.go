package artifacts

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts/whl"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type mutatorFactory = func(name string) bundle.Mutator

var buildMutators map[config.ArtifactType]mutatorFactory = map[config.ArtifactType]mutatorFactory{
	config.ArtifactPythonWheel: whl.Build,
}

var uploadMutators map[config.ArtifactType]mutatorFactory = map[config.ArtifactType]mutatorFactory{}

func getBuildMutator(t config.ArtifactType, name string) bundle.Mutator {
	mutatorFactory, ok := buildMutators[t]
	if !ok {
		mutatorFactory = BasicBuild
	}

	return mutatorFactory(name)
}

func getUploadMutator(t config.ArtifactType, name string) bundle.Mutator {
	mutatorFactory, ok := uploadMutators[t]
	if !ok {
		mutatorFactory = BasicUpload
	}

	return mutatorFactory(name)
}

// Basic Build defines a general build mutator which builds artifact based on artifact.BuildCommand
type basicBuild struct {
	name string
}

func BasicBuild(name string) bundle.Mutator {
	return &basicBuild{name: name}
}

func (m *basicBuild) Name() string {
	return fmt.Sprintf("artifacts.Build(%s)", m.name)
}

func (m *basicBuild) Apply(ctx context.Context, b *bundle.Bundle) error {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return fmt.Errorf("artifact doesn't exist: %s", m.name)
	}

	cmdio.LogString(ctx, fmt.Sprintf("artifacts.Build(%s): Building...", m.name))

	out, err := artifact.Build(ctx)
	if err != nil {
		return fmt.Errorf("artifacts.Build(%s): %w, output: %s", m.name, err, out)
	}
	cmdio.LogString(ctx, fmt.Sprintf("artifacts.Build(%s): Build succeeded", m.name))

	return nil
}

// Basic Upload defines a general upload mutator which uploads artifact as a library to workspace
type basicUpload struct {
	name string
}

func BasicUpload(name string) bundle.Mutator {
	return &basicUpload{name: name}
}

func (m *basicUpload) Name() string {
	return fmt.Sprintf("artifacts.Build(%s)", m.name)
}

func (m *basicUpload) Apply(ctx context.Context, b *bundle.Bundle) error {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return fmt.Errorf("artifact doesn't exist: %s", m.name)
	}

	if len(artifact.Files) == 0 {
		return fmt.Errorf("artifact source is not configured: %s", m.name)
	}

	err := uploadArtifact(ctx, artifact, b)
	if err != nil {
		return fmt.Errorf("artifacts.Upload(%s): %w", m.name, err)
	}

	return nil
}

func uploadArtifact(ctx context.Context, a *config.Artifact, b *bundle.Bundle) error {
	for i := range a.Files {
		f := &a.Files[i]
		if f.NeedsUpload() {
			filename := filepath.Base(f.Source)
			cmdio.LogString(ctx, fmt.Sprintf("artifacts.Upload(%s): Uploading...", filename))
			remotePath, err := uploadArtifactFile(ctx, f.Source, b)
			if err != nil {
				return err
			}
			cmdio.LogString(ctx, fmt.Sprintf("artifacts.Upload(%s): Upload succeeded", filename))

			f.RemotePath = remotePath
		}
	}

	a.NormalisePaths()
	return nil
}

// Function to upload artifact file to Workspace
func uploadArtifactFile(ctx context.Context, file string, b *bundle.Bundle) (string, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("unable to read %s: %w", file, errors.Unwrap(err))
	}

	uploadPath, err := getUploadBasePath(b)
	if err != nil {
		return "", err
	}

	fileHash := sha256.Sum256(raw)
	remotePath := path.Join(uploadPath, fmt.Sprintf("%x", fileHash), filepath.Base(file))
	// Make sure target directory exists.
	err = b.WorkspaceClient().Workspace.MkdirsByPath(ctx, path.Dir(remotePath))
	if err != nil {
		return "", fmt.Errorf("unable to create directory for %s: %w", remotePath, err)
	}

	// Import to workspace.
	err = b.WorkspaceClient().Workspace.Import(ctx, workspace.Import{
		Path:      remotePath,
		Overwrite: true,
		Format:    workspace.ImportFormatAuto,
		Content:   base64.StdEncoding.EncodeToString(raw),
	})
	if err != nil {
		return "", fmt.Errorf("unable to import %s: %w", remotePath, err)
	}

	return remotePath, nil
}

func getUploadBasePath(b *bundle.Bundle) (string, error) {
	artifactPath := b.Config.Workspace.ArtifactsPath
	if artifactPath == "" {
		return "", fmt.Errorf("remote artifact path not configured")
	}

	return path.Join(artifactPath, ".internal"), nil
}
