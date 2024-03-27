package bundle

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
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
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W
	dir := t.TempDir()
	whlPath := filepath.Join(dir, "dist", "test.whl")
	touchEmptyFile(t, whlPath)

	wsDir := internal.TemporaryWorkspaceDir(t, w)

	b := &bundle.Bundle{
		RootPath: dir,
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "whatever",
			},
			Workspace: config.Workspace{
				ArtifactPath: wsDir,
			},
			Artifacts: config.Artifacts{
				"test": &config.Artifact{
					Type: "whl",
					Files: []config.ArtifactFile{
						{
							Source: whlPath,
						},
					},
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									Libraries: []compute.Library{
										{
											Whl: "dist/test.whl",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(ctx, b, artifacts.BasicUpload("test"))
	require.NoError(t, diags.Error())

	// The remote path attribute on the artifact file should have been set.
	require.Regexp(t,
		regexp.MustCompile(path.Join(regexp.QuoteMeta(wsDir), `.internal/test\.whl`)),
		b.Config.Artifacts["test"].Files[0].RemotePath,
	)

	// The task library path should have been updated to the remote path.
	require.Regexp(t,
		regexp.MustCompile(path.Join("/Workspace", regexp.QuoteMeta(wsDir), `.internal/test\.whl`)),
		b.Config.Resources.Jobs["test"].JobSettings.Tasks[0].Libraries[0].Whl,
	)
}
