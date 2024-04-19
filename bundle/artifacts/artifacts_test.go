package artifacts

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	mockfiler "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestArtifactUpload(t *testing.T) {
	tmpDir := t.TempDir()
	whlFolder := filepath.Join(tmpDir, "whl")
	testutil.Touch(t, whlFolder, "source.whl")
	whlLocalPath := filepath.Join(whlFolder, "source.whl")

	b := &bundle.Bundle{
		RootPath: tmpDir,
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/foo/bar/artifacts",
			},
			Artifacts: config.Artifacts{
				"whl": {
					Type: config.ArtifactPythonWheel,
					Files: []config.ArtifactFile{
						{Source: whlLocalPath},
					},
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									Libraries: []compute.Library{
										{
											Whl: filepath.Join("whl", "*.whl"),
										},
										{
											Whl: "/Workspace/Users/foo@bar.com/mywheel.whl",
										},
									},
								},
							},
							Environments: []jobs.JobEnvironment{
								{
									Spec: &compute.Environment{
										Dependencies: []string{
											filepath.Join("whl", "source.whl"),
											"/Workspace/Users/foo@bar.com/mywheel.whl",
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

	artifact := b.Config.Artifacts["whl"]
	mockFiler := mockfiler.NewMockFiler(t)
	mockFiler.EXPECT().Write(
		mock.Anything,
		filepath.Join("source.whl"),
		mock.AnythingOfType("*bytes.Reader"),
		filer.OverwriteIfExists,
		filer.CreateParentDirectories,
	).Return(nil)

	err := uploadArtifact(context.Background(), b, artifact, "/foo/bar/artifacts", mockFiler)
	require.NoError(t, err)

	// Test that libraries path is updated
	require.Equal(t, "/Workspace/foo/bar/artifacts/source.whl", b.Config.Resources.Jobs["job"].JobSettings.Tasks[0].Libraries[0].Whl)
	require.Equal(t, "/Workspace/Users/foo@bar.com/mywheel.whl", b.Config.Resources.Jobs["job"].JobSettings.Tasks[0].Libraries[1].Whl)
	require.Equal(t, "/Workspace/foo/bar/artifacts/source.whl", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[0])
	require.Equal(t, "/Workspace/Users/foo@bar.com/mywheel.whl", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[1])
}
