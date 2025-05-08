package libraries

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
						JobSettings: jobs.JobSettings{
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

	diags := bundle.ApplySeq(context.Background(), b, ExpandGlobReferences(), UploadWithClient(mockFiler))
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
						JobSettings: jobs.JobSettings{
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

	diags := bundle.ApplySeq(context.Background(), b, ExpandGlobReferences(), UploadWithClient(mockFiler))
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

	diags := bundle.ApplySeq(context.Background(), b, ExpandGlobReferences(), UploadWithClient(mockFiler))
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
						JobSettings: jobs.JobSettings{
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

	diags := bundle.ApplySeq(context.Background(), b, ExpandGlobReferences(), UploadWithClient(mockFiler))
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
