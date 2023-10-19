package bundle

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/internal"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/require"
)

func touchEmptyFile(t *testing.T, path string) {
	err := os.MkdirAll(filepath.Dir(path), 0700)
	require.NoError(t, err)
	f, err := os.Create(path)
	require.NoError(t, err)
	f.Close()
}

func TestAccUploadArtifactFileToCorrectRemotePath(t *testing.T) {
	t.Log(internal.GetEnvOrSkipTest(t, "CLOUD_ENV"))

	dir := t.TempDir()
	whlPath := filepath.Join(dir, "dist", "test.whl")
	touchEmptyFile(t, whlPath)

	artifact := &config.Artifact{
		Type: "whl",
		Files: []config.ArtifactFile{
			{
				Source: whlPath,
				Libraries: []*compute.Library{
					{Whl: "dist\\test.whl"},
				},
			},
		},
	}

	w := databricks.Must(databricks.NewWorkspaceClient())
	wsDir := internal.TemporaryWorkspaceDir(t, w)

	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Bundle: config.Bundle{
				Target: "whatever",
			},
			Workspace: config.Workspace{
				ArtifactsPath: wsDir,
			},
			Artifacts: config.Artifacts{
				"test": artifact,
			},
		},
	}

	err := bundle.Apply(context.Background(), b, artifacts.BasicUpload("test"))
	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile(path.Join(regexp.QuoteMeta(wsDir), `.internal/[a-z0-9]+/test\.whl`)), artifact.Files[0].RemotePath)
	require.Regexp(t, regexp.MustCompile(path.Join("/Workspace", regexp.QuoteMeta(wsDir), `.internal/[a-z0-9]+/test\.whl`)), artifact.Files[0].Libraries[0].Whl)
}
