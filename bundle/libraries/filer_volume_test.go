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
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFindVolumeInBundle(t *testing.T) {
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

	bundletest.SetLocation(b, "resources.volumes.foo", []dyn.Location{
		{
			File:   "volume.yml",
			Line:   1,
			Column: 2,
		},
	})

	// volume is in DAB.
	path, locations, ok := findVolumeInBundle(b, "main", "my_schema", "my_volume")
	assert.True(t, ok)
	assert.Equal(t, []dyn.Location{{
		File:   "volume.yml",
		Line:   1,
		Column: 2,
	}}, locations)
	assert.Equal(t, dyn.MustPathFromString("resources.volumes.foo"), path)

	// wrong volume name
	_, _, ok = findVolumeInBundle(b, "main", "my_schema", "doesnotexist")
	assert.False(t, ok)

	// wrong schema name
	_, _, ok = findVolumeInBundle(b, "main", "doesnotexist", "my_volume")
	assert.False(t, ok)

	// wrong catalog name
	_, _, ok = findVolumeInBundle(b, "doesnotexist", "my_schema", "my_volume")
	assert.False(t, ok)

	// schema name is interpolated.
	b.Config.Resources.Volumes["foo"].SchemaName = "${resources.schemas.my_schema}"
	path, locations, ok = findVolumeInBundle(b, "main", "valuedoesnotmatter", "my_volume")
	assert.True(t, ok)
	assert.Equal(t, []dyn.Location{{
		File:   "volume.yml",
		Line:   1,
		Column: 2,
	}}, locations)
	assert.Equal(t, dyn.MustPathFromString("resources.volumes.foo"), path)
}

func TestFilerForVolumeNotInBundle(t *testing.T) {
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

	_, _, diags := filerForVolume(context.Background(), b)
	assert.EqualError(t, diags.Error(), "failed to fetch metadata for the UC volume /Volumes/main/my_schema/doesnotexist that is configured in the artifact_path: error from API")
	assert.Len(t, diags, 1)
}

func TestFilerForVolumeInBundle(t *testing.T) {
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

	bundletest.SetLocation(b, "resources.volumes.foo", []dyn.Location{
		{
			File:   "volume.yml",
			Line:   1,
			Column: 2,
		},
	})

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &sdkconfig.Config{}
	m.GetMockFilesAPI().EXPECT().GetDirectoryMetadataByDirectoryPath(mock.Anything, "/Volumes/main/my_schema/my_volume").Return(fmt.Errorf("error from API"))
	b.SetWorkpaceClient(m.WorkspaceClient)

	_, _, diags := GetFilerForLibraries(context.Background(), b)
	assert.EqualError(t, diags.Error(), "failed to fetch metadata for the UC volume /Volumes/main/my_schema/my_volume that is configured in the artifact_path: error from API")
	assert.Contains(t, diags, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "You might be using a UC volume in your artifact_path that is managed by this bundle but which has not been deployed yet. Please deploy the UC volume in a separate bundle deploy before using it in the artifact_path.",
		Locations: []dyn.Location{{
			File:   "volume.yml",
			Line:   1,
			Column: 2,
		}},
		Paths: []dyn.Path{dyn.MustPathFromString("resources.volumes.foo")},
	})
}

func TestFilerForVolumeWithInvalidVolumePaths(t *testing.T) {
	invalidPaths := []string{
		"/Volumes/",
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
		require.EqualError(t, diags.Error(), fmt.Sprintf("expected UC volume path to be in the format /Volumes/<catalog>/<schema>/<volume>/..., got %s", path.Join(p, ".internal")))
	}
}

func TestFilerForVolumeWithValidlVolumePaths(t *testing.T) {
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

		client, uploadPath, diags := filerForVolume(context.Background(), b)
		require.NoError(t, diags.Error())
		assert.Equal(t, path.Join(p, ".internal"), uploadPath)
		assert.IsType(t, &filer.FilesClient{}, client)
	}
}
