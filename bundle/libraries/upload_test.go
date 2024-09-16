package libraries

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	mockfiler "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/filer"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestArtifactUploadForWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	whlFolder := filepath.Join(tmpDir, "whl")
	testutil.Touch(t, whlFolder, "source.whl")
	whlLocalPath := filepath.Join(whlFolder, "source.whl")

	b := &bundle.Bundle{
		SyncRootPath: tmpDir,
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
								{
									ForEachTask: &jobs.ForEachTask{
										Task: jobs.Task{
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

	mockFiler := mockfiler.NewMockFiler(t)
	mockFiler.EXPECT().Write(
		mock.Anything,
		filepath.Join("source.whl"),
		mock.AnythingOfType("*os.File"),
		filer.OverwriteIfExists,
		filer.CreateParentDirectories,
	).Return(nil)

	diags := bundle.Apply(context.Background(), b, bundle.Seq(ExpandGlobReferences(), UploadWithClient(mockFiler)))
	require.NoError(t, diags.Error())

	// Test that libraries path is updated
	require.Equal(t, "/Workspace/foo/bar/artifacts/.internal/source.whl", b.Config.Resources.Jobs["job"].JobSettings.Tasks[0].Libraries[0].Whl)
	require.Equal(t, "/Workspace/Users/foo@bar.com/mywheel.whl", b.Config.Resources.Jobs["job"].JobSettings.Tasks[0].Libraries[1].Whl)
	require.Equal(t, "/Workspace/foo/bar/artifacts/.internal/source.whl", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[0])
	require.Equal(t, "/Workspace/Users/foo@bar.com/mywheel.whl", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[1])
	require.Equal(t, "/Workspace/foo/bar/artifacts/.internal/source.whl", b.Config.Resources.Jobs["job"].JobSettings.Tasks[1].ForEachTask.Task.Libraries[0].Whl)
	require.Equal(t, "/Workspace/Users/foo@bar.com/mywheel.whl", b.Config.Resources.Jobs["job"].JobSettings.Tasks[1].ForEachTask.Task.Libraries[1].Whl)
}

func TestArtifactUploadForVolumes(t *testing.T) {
	tmpDir := t.TempDir()
	whlFolder := filepath.Join(tmpDir, "whl")
	testutil.Touch(t, whlFolder, "source.whl")
	whlLocalPath := filepath.Join(whlFolder, "source.whl")

	b := &bundle.Bundle{
		SyncRootPath: tmpDir,
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Volumes/foo/bar/artifacts",
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
											Whl: "/Volumes/some/path/mywheel.whl",
										},
									},
								},
								{
									ForEachTask: &jobs.ForEachTask{
										Task: jobs.Task{
											Libraries: []compute.Library{
												{
													Whl: filepath.Join("whl", "*.whl"),
												},
												{
													Whl: "/Volumes/some/path/mywheel.whl",
												},
											},
										},
									},
								},
							},
							Environments: []jobs.JobEnvironment{
								{
									Spec: &compute.Environment{
										Dependencies: []string{
											filepath.Join("whl", "source.whl"),
											"/Volumes/some/path/mywheel.whl",
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

	mockFiler := mockfiler.NewMockFiler(t)
	mockFiler.EXPECT().Write(
		mock.Anything,
		filepath.Join("source.whl"),
		mock.AnythingOfType("*os.File"),
		filer.OverwriteIfExists,
		filer.CreateParentDirectories,
	).Return(nil)

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &sdkconfig.Config{}
	m.GetMockFilesAPI().EXPECT().GetDirectoryMetadataByDirectoryPath(mock.Anything, "/Volumes/foo/bar/artifacts").Return(nil)
	b.SetWorkpaceClient(m.WorkspaceClient)

	diags := bundle.Apply(context.Background(), b, bundle.Seq(ExpandGlobReferences(), UploadWithClient(mockFiler)))
	require.NoError(t, diags.Error())

	// Test that libraries path is updated
	require.Equal(t, "/Volumes/foo/bar/artifacts/.internal/source.whl", b.Config.Resources.Jobs["job"].JobSettings.Tasks[0].Libraries[0].Whl)
	require.Equal(t, "/Volumes/some/path/mywheel.whl", b.Config.Resources.Jobs["job"].JobSettings.Tasks[0].Libraries[1].Whl)
	require.Equal(t, "/Volumes/foo/bar/artifacts/.internal/source.whl", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[0])
	require.Equal(t, "/Volumes/some/path/mywheel.whl", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[1])
	require.Equal(t, "/Volumes/foo/bar/artifacts/.internal/source.whl", b.Config.Resources.Jobs["job"].JobSettings.Tasks[1].ForEachTask.Task.Libraries[0].Whl)
	require.Equal(t, "/Volumes/some/path/mywheel.whl", b.Config.Resources.Jobs["job"].JobSettings.Tasks[1].ForEachTask.Task.Libraries[1].Whl)
}

func TestArtifactUploadWithNoLibraryReference(t *testing.T) {
	tmpDir := t.TempDir()
	whlFolder := filepath.Join(tmpDir, "whl")
	testutil.Touch(t, whlFolder, "source.whl")
	whlLocalPath := filepath.Join(whlFolder, "source.whl")

	b := &bundle.Bundle{
		SyncRootPath: tmpDir,
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Workspace/foo/bar/artifacts",
			},
			Artifacts: config.Artifacts{
				"whl": {
					Type: config.ArtifactPythonWheel,
					Files: []config.ArtifactFile{
						{Source: whlLocalPath},
					},
				},
			},
		},
	}

	mockFiler := mockfiler.NewMockFiler(t)
	mockFiler.EXPECT().Write(
		mock.Anything,
		filepath.Join("source.whl"),
		mock.AnythingOfType("*os.File"),
		filer.OverwriteIfExists,
		filer.CreateParentDirectories,
	).Return(nil)

	diags := bundle.Apply(context.Background(), b, bundle.Seq(ExpandGlobReferences(), UploadWithClient(mockFiler)))
	require.NoError(t, diags.Error())

	require.Equal(t, "/Workspace/foo/bar/artifacts/.internal/source.whl", b.Config.Artifacts["whl"].Files[0].RemotePath)
}

func TestUploadMultipleLibraries(t *testing.T) {
	tmpDir := t.TempDir()
	whlFolder := filepath.Join(tmpDir, "whl")
	testutil.Touch(t, whlFolder, "source1.whl")
	testutil.Touch(t, whlFolder, "source2.whl")
	testutil.Touch(t, whlFolder, "source3.whl")
	testutil.Touch(t, whlFolder, "source4.whl")

	b := &bundle.Bundle{
		SyncRootPath: tmpDir,
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/foo/bar/artifacts",
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
											filepath.Join("whl", "*.whl"),
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

	mockFiler := mockfiler.NewMockFiler(t)
	mockFiler.EXPECT().Write(
		mock.Anything,
		filepath.Join("source1.whl"),
		mock.AnythingOfType("*os.File"),
		filer.OverwriteIfExists,
		filer.CreateParentDirectories,
	).Return(nil).Once()

	mockFiler.EXPECT().Write(
		mock.Anything,
		filepath.Join("source2.whl"),
		mock.AnythingOfType("*os.File"),
		filer.OverwriteIfExists,
		filer.CreateParentDirectories,
	).Return(nil).Once()

	mockFiler.EXPECT().Write(
		mock.Anything,
		filepath.Join("source3.whl"),
		mock.AnythingOfType("*os.File"),
		filer.OverwriteIfExists,
		filer.CreateParentDirectories,
	).Return(nil).Once()

	mockFiler.EXPECT().Write(
		mock.Anything,
		filepath.Join("source4.whl"),
		mock.AnythingOfType("*os.File"),
		filer.OverwriteIfExists,
		filer.CreateParentDirectories,
	).Return(nil).Once()

	diags := bundle.Apply(context.Background(), b, bundle.Seq(ExpandGlobReferences(), UploadWithClient(mockFiler)))
	require.NoError(t, diags.Error())

	// Test that libraries path is updated
	require.Len(t, b.Config.Resources.Jobs["job"].JobSettings.Tasks[0].Libraries, 5)
	require.Contains(t, b.Config.Resources.Jobs["job"].JobSettings.Tasks[0].Libraries, compute.Library{Whl: "/Workspace/foo/bar/artifacts/.internal/source1.whl"})
	require.Contains(t, b.Config.Resources.Jobs["job"].JobSettings.Tasks[0].Libraries, compute.Library{Whl: "/Workspace/foo/bar/artifacts/.internal/source2.whl"})
	require.Contains(t, b.Config.Resources.Jobs["job"].JobSettings.Tasks[0].Libraries, compute.Library{Whl: "/Workspace/foo/bar/artifacts/.internal/source3.whl"})
	require.Contains(t, b.Config.Resources.Jobs["job"].JobSettings.Tasks[0].Libraries, compute.Library{Whl: "/Workspace/foo/bar/artifacts/.internal/source4.whl"})
	require.Contains(t, b.Config.Resources.Jobs["job"].JobSettings.Tasks[0].Libraries, compute.Library{Whl: "/Workspace/Users/foo@bar.com/mywheel.whl"})

	require.Len(t, b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies, 5)
	require.Contains(t, b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies, "/Workspace/foo/bar/artifacts/.internal/source1.whl")
	require.Contains(t, b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies, "/Workspace/foo/bar/artifacts/.internal/source2.whl")
	require.Contains(t, b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies, "/Workspace/foo/bar/artifacts/.internal/source3.whl")
	require.Contains(t, b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies, "/Workspace/foo/bar/artifacts/.internal/source4.whl")
	require.Contains(t, b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies, "/Workspace/Users/foo@bar.com/mywheel.whl")
}

func TestMatchVolumeInBundle(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"foo": {
						CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
							CatalogName: "main",
							Name:        "my_volume",
							SchemaName:  "my_schema",
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.volumes.foo", "volume.yml")

	// volume is in DAB.
	path, locations, ok := matchVolumeInBundle(b, "main", "my_schema", "my_volume")
	assert.True(t, ok)
	assert.Equal(t, []dyn.Location{{
		File: "volume.yml",
	}}, locations)
	assert.Equal(t, dyn.MustPathFromString("resources.volumes.foo"), path)

	// wrong volume name
	_, _, ok = matchVolumeInBundle(b, "main", "my_schema", "doesnotexist")
	assert.False(t, ok)

	// wrong schema name
	_, _, ok = matchVolumeInBundle(b, "main", "doesnotexist", "my_volume")
	assert.False(t, ok)

	// schema name is interpolated.
	b.Config.Resources.Volumes["foo"].SchemaName = "${resources.schemas.my_schema}"
	path, locations, ok = matchVolumeInBundle(b, "main", "valuedoesnotmatter", "my_volume")
	assert.True(t, ok)
	assert.Equal(t, []dyn.Location{{
		File: "volume.yml",
	}}, locations)
	assert.Equal(t, dyn.MustPathFromString("resources.volumes.foo"), path)
}

func TestGetFilerForLibraries(t *testing.T) {
	t.Run("valid wsfs", func(t *testing.T) {
		b := &bundle.Bundle{
			Config: config.Root{
				Workspace: config.Workspace{
					ArtifactPath: "/foo/bar/artifacts",
				},
			},
		}

		client, uploadPath, diags := GetFilerForLibraries(context.Background(), b)
		require.NoError(t, diags.Error())
		assert.Equal(t, "/foo/bar/artifacts/.internal", uploadPath)

		assert.IsType(t, &filer.WorkspaceFilesClient{}, client)
	})

	t.Run("valid uc volume", func(t *testing.T) {
		b := &bundle.Bundle{
			Config: config.Root{
				Workspace: config.Workspace{
					ArtifactPath: "/Volumes/main/my_schema/my_volume",
				},
			},
		}

		m := mocks.NewMockWorkspaceClient(t)
		m.WorkspaceClient.Config = &sdkconfig.Config{}
		m.GetMockFilesAPI().EXPECT().GetDirectoryMetadataByDirectoryPath(mock.Anything, "/Volumes/main/my_schema/my_volume").Return(nil)
		b.SetWorkpaceClient(m.WorkspaceClient)

		client, uploadPath, diags := GetFilerForLibraries(context.Background(), b)
		require.NoError(t, diags.Error())
		assert.Equal(t, "/Volumes/main/my_schema/my_volume/.internal", uploadPath)

		assert.IsType(t, &filer.FilesClient{}, client)
	})

	t.Run("volume not in DAB", func(t *testing.T) {
		b := &bundle.Bundle{
			Config: config.Root{
				Workspace: config.Workspace{
					ArtifactPath: "/Volumes/main/my_schema/doesnotexist",
				},
			},
		}

		m := mocks.NewMockWorkspaceClient(t)
		m.WorkspaceClient.Config = &sdkconfig.Config{}
		m.GetMockFilesAPI().EXPECT().GetDirectoryMetadataByDirectoryPath(mock.Anything, "/Volumes/main/my_schema/doesnotexist").Return(fmt.Errorf("error from API"))
		b.SetWorkpaceClient(m.WorkspaceClient)

		_, _, diags := GetFilerForLibraries(context.Background(), b)
		require.EqualError(t, diags.Error(), "failed to fetch metadata for the UC volume /Volumes/main/my_schema/doesnotexist that is configured in the artifact_path: error from API")
	})

	t.Run("volume in DAB config", func(t *testing.T) {
		b := &bundle.Bundle{
			Config: config.Root{
				Workspace: config.Workspace{
					ArtifactPath: "/Volumes/main/my_schema/my_volume",
				},
				Resources: config.Resources{
					Volumes: map[string]*resources.Volume{
						"foo": {
							CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
								CatalogName: "main",
								Name:        "my_volume",
								VolumeType:  "MANAGED",
								SchemaName:  "my_schema",
							},
						},
					},
				},
			},
		}

		bundletest.SetLocation(b, "resources.volumes.foo", "volume.yml")

		m := mocks.NewMockWorkspaceClient(t)
		m.WorkspaceClient.Config = &sdkconfig.Config{}
		m.GetMockFilesAPI().EXPECT().GetDirectoryMetadataByDirectoryPath(mock.Anything, "/Volumes/main/my_schema/my_volume").Return(fmt.Errorf("error from API"))
		b.SetWorkpaceClient(m.WorkspaceClient)

		_, _, diags := GetFilerForLibraries(context.Background(), b)
		assert.EqualError(t, diags.Error(), "failed to fetch metadata for the UC volume /Volumes/main/my_schema/my_volume that is configured in the artifact_path: error from API")
		assert.Contains(t, diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "the UC volume that is likely being used in the artifact_path has not been deployed yet. Please deploy the UC volume in a separate bundle deploy before using it in the artifact_path.",
			Locations: []dyn.Location{{
				File: "volume.yml",
			}},
			Paths: []dyn.Path{dyn.MustPathFromString("resources.volumes.foo")},
		})
	})

	t.Run("remote path is not set", func(t *testing.T) {
		b := &bundle.Bundle{}

		_, _, diags := GetFilerForLibraries(context.Background(), b)
		require.EqualError(t, diags.Error(), "remote artifact path not configured")
	})

	t.Run("invalid volume paths", func(t *testing.T) {
		invalidPaths := []string{
			"/Volumes",
			"/Volumes/main",
			"/Volumes/main/",
			"/Volumes/main//",
			"/Volumes/main//my_schema",
			"/Volumes/main/my_schema",
			"/Volumes/main/my_schema/",
			"/Volumes/main/my_schema//",
			"/Volumes//my_schema/my_volume",
		}

		for _, p := range invalidPaths {
			b := &bundle.Bundle{
				Config: config.Root{
					Workspace: config.Workspace{
						ArtifactPath: p,
					},
				},
			}

			_, _, diags := GetFilerForLibraries(context.Background(), b)
			require.EqualError(t, diags.Error(), fmt.Sprintf("expected UC volume path to be in the format /Volumes/<catalog>/<schema>/<path>, got %s", path.Join(p, ".internal")))
		}
	})
}
