package bundle_test

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
	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func touchEmptyFile(t *testing.T, path string) {
	err := os.MkdirAll(filepath.Dir(path), 0o700)
	require.NoError(t, err)
	f, err := os.Create(path)
	require.NoError(t, err)
	f.Close()
}

func TestUploadArtifactFileToCorrectRemotePath(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	dir := t.TempDir()
	whlPath := filepath.Join(dir, "dist", "test.whl")
	touchEmptyFile(t, whlPath)

	wsDir := acc.TemporaryWorkspaceDir(wt, "artifact-")

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
		path.Join(regexp.QuoteMeta(wsDir), `.internal/test\.whl`),
		b.Config.Artifacts["test"].Files[0].RemotePath,
	)

	// The task library path should have been updated to the remote path.
	require.Regexp(t,
		path.Join("/Workspace", regexp.QuoteMeta(wsDir), `.internal/test\.whl`),
		b.Config.Resources.Jobs["test"].JobSettings.Tasks[0].Libraries[0].Whl,
	)
}

func TestUploadArtifactFileToCorrectRemotePathWithEnvironments(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	dir := t.TempDir()
	whlPath := filepath.Join(dir, "dist", "test.whl")
	touchEmptyFile(t, whlPath)

	wsDir := acc.TemporaryWorkspaceDir(wt, "artifact-")

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
		path.Join(regexp.QuoteMeta(wsDir), `.internal/test\.whl`),
		b.Config.Artifacts["test"].Files[0].RemotePath,
	)

	// The job environment deps path should have been updated to the remote path.
	require.Regexp(t,
		path.Join("/Workspace", regexp.QuoteMeta(wsDir), `.internal/test\.whl`),
		b.Config.Resources.Jobs["test"].JobSettings.Environments[0].Spec.Dependencies[0],
	)
}

func TestUploadArtifactFileToCorrectRemotePathForVolumes(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	if os.Getenv("TEST_METASTORE_ID") == "" {
		t.Skip("Skipping tests that require a UC Volume when metastore id is not set.")
	}

	volumePath := acc.TemporaryVolume(wt)

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
		path.Join(regexp.QuoteMeta(volumePath), `.internal/test\.whl`),
		b.Config.Artifacts["test"].Files[0].RemotePath,
	)

	// The task library path should have been updated to the remote path.
	require.Regexp(t,
		path.Join(regexp.QuoteMeta(volumePath), `.internal/test\.whl`),
		b.Config.Resources.Jobs["test"].JobSettings.Tasks[0].Libraries[0].Whl,
	)
}

func TestUploadArtifactFileToVolumeThatDoesNotExist(t *testing.T) {
	ctx, wt := acc.UcWorkspaceTest(t)
	w := wt.W

	schemaName := testutil.RandomName("schema-")

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

	bundleRoot := initTestTemplate(t, ctx, "artifact_path_with_volume", map[string]any{
		"unique_id":   uuid.New().String(),
		"schema_name": schemaName,
		"volume_name": "doesnotexist",
	})

	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	stdout, stderr, err := testcli.RequireErrorRun(t, ctx, "bundle", "deploy")

	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf(`Error: volume main.%s.doesnotexist does not exist
  at workspace.artifact_path
  in databricks.yml:6:18

`, schemaName), stdout.String())
	assert.Equal(t, "", stderr.String())
}

func TestUploadArtifactToVolumeNotYetDeployed(t *testing.T) {
	ctx, wt := acc.UcWorkspaceTest(t)
	w := wt.W

	schemaName := testutil.RandomName("schema-")

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

	bundleRoot := initTestTemplate(t, ctx, "artifact_path_with_volume", map[string]any{
		"unique_id":   uuid.New().String(),
		"schema_name": schemaName,
		"volume_name": "my_volume",
	})

	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	stdout, stderr, err := testcli.RequireErrorRun(t, ctx, "bundle", "deploy")

	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf(`Error: volume main.%s.my_volume does not exist
  at workspace.artifact_path
     resources.volumes.foo
  in databricks.yml:6:18
     databricks.yml:11:7

You are using a volume in your artifact_path that is managed by
this bundle but which has not been deployed yet. Please first deploy
the volume using 'bundle deploy' and then switch over to using it in
the artifact_path.

`, schemaName), stdout.String())
	assert.Equal(t, "", stderr.String())
}
