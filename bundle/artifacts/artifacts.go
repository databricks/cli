package artifacts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts/whl"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
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
	return fmt.Sprintf("artifacts.Upload(%s)", m.name)
}

func (m *basicUpload) Apply(ctx context.Context, b *bundle.Bundle) error {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return fmt.Errorf("artifact doesn't exist: %s", m.name)
	}

	if len(artifact.Files) == 0 {
		return fmt.Errorf("artifact source is not configured: %s", m.name)
	}

	uploadPath, err := getUploadBasePath(b)
	if err != nil {
		return err
	}

	client, err := filer.NewWorkspaceFilesClientWithProgressLogging(b.WorkspaceClient(), uploadPath)
	if err != nil {
		return err
	}

	err = uploadArtifact(ctx, artifact, uploadPath, client)
	if err != nil {
		return fmt.Errorf("artifacts.Upload(%s): %w", m.name, err)
	}

	return nil
}

func uploadArtifact(ctx context.Context, a *config.Artifact, uploadPath string, client filer.Filer) error {
	for i := range a.Files {
		f := &a.Files[i]
		if f.NeedsUpload() {
			filename := filepath.Base(f.Source)
			cmdio.LogString(ctx, fmt.Sprintf("artifacts.Upload(%s): Uploading...", filename))

			remotePath, err := uploadArtifactFile(ctx, f.Source, uploadPath, client)
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
func uploadArtifactFile(ctx context.Context, file string, uploadPath string, client filer.Filer) (string, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("unable to read %s: %w", file, errors.Unwrap(err))
	}

	fileHash := sha256.Sum256(raw)
	relPath := path.Join(fmt.Sprintf("%x", fileHash), filepath.Base(file))
	remotePath := path.Join(uploadPath, relPath)

	err = client.Mkdir(ctx, path.Dir(relPath))
	if err != nil {
		return "", fmt.Errorf("unable to import %s: %w", remotePath, err)
	}

	err = client.Write(ctx, relPath, bytes.NewReader(raw), int64(len(raw)), filer.OverwriteIfExists, filer.CreateParentDirectories)
	if err != nil {
		return "", fmt.Errorf("unable to import %s: %w", remotePath, err)
	}

	return remotePath, nil
}

func getUploadBasePath(b *bundle.Bundle) (string, error) {
	artifactPath := b.Config.Workspace.ArtifactPath
	if artifactPath == "" {
		return "", fmt.Errorf("remote artifact path not configured")
	}

	return path.Join(artifactPath, ".internal"), nil
}
