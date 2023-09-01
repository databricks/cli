package artifacts

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/require"
)

func touchEmptyFile(t *testing.T, path string) {
	err := os.MkdirAll(filepath.Dir(path), 0700)
	require.NoError(t, err)
	f, err := os.Create(path)
	require.NoError(t, err)
	f.Close()
}

type MockWorkspaceService struct {
}

// Delete implements workspace.WorkspaceService.
func (MockWorkspaceService) Delete(ctx context.Context, request workspace.Delete) error {
	panic("unimplemented")
}

// Export implements workspace.WorkspaceService.
func (MockWorkspaceService) Export(ctx context.Context, request workspace.ExportRequest) (*workspace.ExportResponse, error) {
	panic("unimplemented")
}

// GetStatus implements workspace.WorkspaceService.
func (MockWorkspaceService) GetStatus(ctx context.Context, request workspace.GetStatusRequest) (*workspace.ObjectInfo, error) {
	panic("unimplemented")
}

// Import implements workspace.WorkspaceService.
func (MockWorkspaceService) Import(ctx context.Context, request workspace.Import) error {
	return nil
}

// List implements workspace.WorkspaceService.
func (MockWorkspaceService) List(ctx context.Context, request workspace.ListWorkspaceRequest) (*workspace.ListResponse, error) {
	panic("unimplemented")
}

// Mkdirs implements workspace.WorkspaceService.
func (MockWorkspaceService) Mkdirs(ctx context.Context, request workspace.Mkdirs) error {
	return nil
}

func TestUploadArtifactFileToCorrectRemotePath(t *testing.T) {
	dir := t.TempDir()
	whlPath := filepath.Join(dir, "dist", "test.whl")
	touchEmptyFile(t, whlPath)
	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Bundle: config.Bundle{
				Target: "whatever",
			},
			Workspace: config.Workspace{
				ArtifactsPath: "/Users/test@databricks.com/whatever",
			},
		},
	}

	b.WorkspaceClient().Workspace.WithImpl(MockWorkspaceService{})
	artifact := &config.Artifact{
		Files: []config.ArtifactFile{
			{
				Source: whlPath,
				Libraries: []*compute.Library{
					{Whl: "dist\\test.whl"},
				},
			},
		},
	}

	err := uploadArtifact(context.Background(), artifact, b)
	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile("/Users/test@databricks.com/whatever/.internal/[a-z0-9]+/test.whl"), artifact.Files[0].RemotePath)
}
