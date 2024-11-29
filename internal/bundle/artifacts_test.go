package bundle

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
		BundleRootPath: dir,
		SyncRootPath:   dir,
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

	diags := bundle.Apply(ctx, b, bundle.Seq(libraries.ExpandGlobReferences(), libraries.Upload()))
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

func TestAccUploadArtifactFileToCorrectRemotePathWithEnvironments(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W
	dir := t.TempDir()
	whlPath := filepath.Join(dir, "dist", "test.whl")
	touchEmptyFile(t, whlPath)

	wsDir := internal.TemporaryWorkspaceDir(t, w)

	b := &bundle.Bundle{
		BundleRootPath: dir,
		SyncRootPath:   dir,
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
							Environments: []jobs.JobEnvironment{
								{
									Spec: &compute.Environment{
										Dependencies: []string{
											"dist/test.whl",
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

	diags := bundle.Apply(ctx, b, bundle.Seq(libraries.ExpandGlobReferences(), libraries.Upload()))
	require.NoError(t, diags.Error())

	// The remote path attribute on the artifact file should have been set.
	require.Regexp(t,
		regexp.MustCompile(path.Join(regexp.QuoteMeta(wsDir), `.internal/test\.whl`)),
		b.Config.Artifacts["test"].Files[0].RemotePath,
	)

	// The job environment deps path should have been updated to the remote path.
	require.Regexp(t,
		regexp.MustCompile(path.Join("/Workspace", regexp.QuoteMeta(wsDir), `.internal/test\.whl`)),
		b.Config.Resources.Jobs["test"].JobSettings.Environments[0].Spec.Dependencies[0],
	)
}

func TestAccUploadArtifactFileToCorrectRemotePathForVolumes(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	if os.Getenv("TEST_METASTORE_ID") == "" {
		t.Skip("Skipping tests that require a UC Volume when metastore id is not set.")
	}

	volumePath := internal.TemporaryUcVolume(t, w)

	dir := t.TempDir()
	whlPath := filepath.Join(dir, "dist", "test.whl")
	touchEmptyFile(t, whlPath)

	b := &bundle.Bundle{
		BundleRootPath: dir,
		SyncRootPath:   dir,
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "whatever",
			},
			Workspace: config.Workspace{
				ArtifactPath: volumePath,
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

	diags := bundle.Apply(ctx, b, bundle.Seq(libraries.ExpandGlobReferences(), libraries.Upload()))
	require.NoError(t, diags.Error())

	// The remote path attribute on the artifact file should have been set.
	require.Regexp(t,
		regexp.MustCompile(path.Join(regexp.QuoteMeta(volumePath), `.internal/test\.whl`)),
		b.Config.Artifacts["test"].Files[0].RemotePath,
	)

	// The task library path should have been updated to the remote path.
	require.Regexp(t,
		regexp.MustCompile(path.Join(regexp.QuoteMeta(volumePath), `.internal/test\.whl`)),
		b.Config.Resources.Jobs["test"].JobSettings.Tasks[0].Libraries[0].Whl,
	)
}

func TestAccUploadArtifactFileToVolumeThatDoesNotExist(t *testing.T) {
	ctx, wt := acc.UcWorkspaceTest(t)
	w := wt.W

	schemaName := internal.RandomName("schema-")

	_, err := w.Schemas.Create(ctx, catalog.CreateSchema{
		CatalogName: "main",
		Comment:     "test schema",
		Name:        schemaName,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = w.Schemas.DeleteByFullName(ctx, "main."+schemaName)
		require.NoError(t, err)
	})

	bundleRoot, err := initTestTemplate(t, ctx, "artifact_path_with_volume", map[string]any{
		"unique_id":   uuid.New().String(),
		"schema_name": schemaName,
		"volume_name": "doesnotexist",
	})
	require.NoError(t, err)

	t.Setenv("BUNDLE_ROOT", bundleRoot)
	stdout, stderr, err := internal.RequireErrorRun(t, "bundle", "deploy")

	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf(`Error: failed to fetch metadata for /Volumes/main/%s/doesnotexist: Not Found
  at workspace.artifact_path
  in databricks.yml:6:18

`, schemaName), stdout.String())
	assert.Equal(t, "", stderr.String())
}

func TestAccUploadArtifactToVolumeNotYetDeployed(t *testing.T) {
	ctx, wt := acc.UcWorkspaceTest(t)
	w := wt.W

	schemaName := internal.RandomName("schema-")

	_, err := w.Schemas.Create(ctx, catalog.CreateSchema{
		CatalogName: "main",
		Comment:     "test schema",
		Name:        schemaName,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = w.Schemas.DeleteByFullName(ctx, "main."+schemaName)
		require.NoError(t, err)
	})

	bundleRoot, err := initTestTemplate(t, ctx, "artifact_path_with_volume", map[string]any{
		"unique_id":   uuid.New().String(),
		"schema_name": schemaName,
		"volume_name": "my_volume",
	})
	require.NoError(t, err)

	t.Setenv("BUNDLE_ROOT", bundleRoot)
	stdout, stderr, err := internal.RequireErrorRun(t, "bundle", "deploy")

	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf(`Error: failed to fetch metadata for /Volumes/main/%s/my_volume: Not Found
  at workspace.artifact_path
     resources.volumes.foo
  in databricks.yml:6:18
     databricks.yml:11:7

You are using a UC volume in your artifact_path that is managed by
this bundle but which has not been deployed yet. Please first deploy
the UC volume using 'bundle deploy' and then switch over to using it in
the artifact_path.

`, schemaName), stdout.String())
	assert.Equal(t, "", stderr.String())
}
