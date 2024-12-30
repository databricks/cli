package libraries

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFilerForVolumeForErrorFromAPI(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Volumes/main/my_schema/my_volume",
			},
		},
	}

	bundletest.SetLocation(b, "workspace.artifact_path", []dyn.Location{{File: "config.yml", Line: 1, Column: 2}})

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &sdkconfig.Config{}
	m.GetMockFilesAPI().EXPECT().GetDirectoryMetadataByDirectoryPath(mock.Anything, "/Volumes/main/my_schema/my_volume").Return(fmt.Errorf("error from API"))
	b.SetWorkpaceClient(m.WorkspaceClient)

	_, _, diags := filerForVolume(b)
	assert.Equal(t, diag.Diagnostics{
		{
			Severity:  diag.Error,
			Summary:   "unable to determine if volume at /Volumes/main/my_schema/my_volume exists: error from API",
			Locations: []dyn.Location{{File: "config.yml", Line: 1, Column: 2}},
			Paths:     []dyn.Path{dyn.MustPathFromString("workspace.artifact_path")},
		},
	}, diags)
}

func TestFilerForVolumeWithVolumeNotFound(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Volumes/main/my_schema/doesnotexist",
			},
		},
	}

	bundletest.SetLocation(b, "workspace.artifact_path", []dyn.Location{{File: "config.yml", Line: 1, Column: 2}})

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &sdkconfig.Config{}
	m.GetMockFilesAPI().EXPECT().GetDirectoryMetadataByDirectoryPath(mock.Anything, "/Volumes/main/my_schema/doesnotexist").Return(apierr.NotFound("some error message"))
	b.SetWorkpaceClient(m.WorkspaceClient)

	_, _, diags := filerForVolume(b)
	assert.Equal(t, diag.Diagnostics{
		{
			Severity:  diag.Error,
			Summary:   "volume /Volumes/main/my_schema/doesnotexist does not exist: some error message",
			Locations: []dyn.Location{{File: "config.yml", Line: 1, Column: 2}},
			Paths:     []dyn.Path{dyn.MustPathFromString("workspace.artifact_path")},
		},
	}, diags)
}

func TestFilerForVolumeNotFoundAndInBundle(t *testing.T) {
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

	bundletest.SetLocation(b, "workspace.artifact_path", []dyn.Location{{File: "config.yml", Line: 1, Column: 2}})
	bundletest.SetLocation(b, "resources.volumes.foo", []dyn.Location{{File: "volume.yml", Line: 1, Column: 2}})

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &sdkconfig.Config{}
	m.GetMockFilesAPI().EXPECT().GetDirectoryMetadataByDirectoryPath(mock.Anything, "/Volumes/main/my_schema/my_volume").Return(apierr.NotFound("error from API"))
	b.SetWorkpaceClient(m.WorkspaceClient)

	_, _, diags := GetFilerForLibraries(context.Background(), b)
	assert.Equal(t, diag.Diagnostics{
		{
			Severity:  diag.Error,
			Summary:   "volume /Volumes/main/my_schema/my_volume does not exist: error from API",
			Locations: []dyn.Location{{File: "config.yml", Line: 1, Column: 2}, {File: "volume.yml", Line: 1, Column: 2}},
			Paths:     []dyn.Path{dyn.MustPathFromString("workspace.artifact_path"), dyn.MustPathFromString("resources.volumes.foo")},
			Detail: `You are using a volume in your artifact_path that is managed by
this bundle but which has not been deployed yet. Please first deploy
the volume using 'bundle deploy' and then switch over to using it in
the artifact_path.`,
		},
	}, diags)
}

func TestFilerForVolumeWithInvalidPrefix(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Volume/main/my_schema/my_volume",
			},
		},
	}

	_, _, diags := filerForVolume(b)
	require.EqualError(t, diags.Error(), "expected artifact_path to start with /Volumes/, got /Volume/main/my_schema/my_volume")
}

func TestFilerForVolumeWithValidVolumePaths(t *testing.T) {
	validPaths := []string{
		"/Volumes/main/my_schema/my_volume",
		"/Volumes/main/my_schema/my_volume/",
		"/Volumes/main/my_schema/my_volume/a/b/c",
		"/Volumes/main/my_schema/my_volume/a/a/a",
	}

	for _, p := range validPaths {
		b := &bundle.Bundle{
			Config: config.Root{
				Workspace: config.Workspace{
					ArtifactPath: p,
				},
			},
		}

		m := mocks.NewMockWorkspaceClient(t)
		m.WorkspaceClient.Config = &sdkconfig.Config{}
		m.GetMockFilesAPI().EXPECT().GetDirectoryMetadataByDirectoryPath(mock.Anything, "/Volumes/main/my_schema/my_volume").Return(nil)
		b.SetWorkpaceClient(m.WorkspaceClient)

		client, uploadPath, diags := filerForVolume(b)
		require.NoError(t, diags.Error())
		assert.Equal(t, path.Join(p, ".internal"), uploadPath)
		assert.IsType(t, &filer.FilesClient{}, client)
	}
}
