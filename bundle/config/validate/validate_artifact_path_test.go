package validate

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateArtifactPathWithVolumeInBundle(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Volumes/catalogN/schemaN/volumeN/abc",
			},
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"foo": {
						CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
							CatalogName: "catalogN",
							Name:        "volumeN",
							SchemaName:  "schemaN",
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "workspace.artifact_path", []dyn.Location{{File: "file", Line: 1, Column: 1}})
	bundletest.SetLocation(b, "resources.volumes.foo", []dyn.Location{{File: "file", Line: 2, Column: 2}})

	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockVolumesAPI()
	api.EXPECT().ReadByName(mock.Anything, "catalogN.schemaN.volumeN").Return(nil, &apierr.APIError{
		StatusCode: 404,
	})
	b.SetWorkpaceClient(m.WorkspaceClient)

	diags := ValidateArtifactPath().Apply(ctx, b)
	assert.Equal(t, diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  "volume catalogN.schemaN.volumeN does not exist",
		Locations: []dyn.Location{
			{File: "file", Line: 1, Column: 1},
			{File: "file", Line: 2, Column: 2},
		},
		Paths: []dyn.Path{
			dyn.MustPathFromString("workspace.artifact_path"),
			dyn.MustPathFromString("resources.volumes.foo"),
		},
		Detail: `You are using a volume in your artifact_path that is managed by
this bundle but which has not been deployed yet. Please first deploy
the volume using 'bundle deploy' and then switch over to using it in
the artifact_path.`,
	}}, diags)
}

func TestValidateArtifactPath(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Volumes/catalogN/schemaN/volumeN/abc",
			},
		},
	}

	bundletest.SetLocation(b, "workspace.artifact_path", []dyn.Location{{File: "file", Line: 1, Column: 1}})
	assertDiags := func(t *testing.T, diags diag.Diagnostics, expected string) {
		assert.Len(t, diags, 1)
		assert.Equal(t, diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   expected,
			Locations: []dyn.Location{{File: "file", Line: 1, Column: 1}},
			Paths:     []dyn.Path{dyn.MustPathFromString("workspace.artifact_path")},
		}}, diags)
	}

	ctx := context.Background()

	tcases := []struct {
		err             error
		expectedSummary string
	}{
		{
			err: &apierr.APIError{
				StatusCode: 403,
				Message:    "User does not have USE SCHEMA on Schema 'catalogN.schemaN'",
			},
			expectedSummary: "cannot access volume catalogN.schemaN.volumeN: User does not have USE SCHEMA on Schema 'catalogN.schemaN'",
		},
		{
			err: &apierr.APIError{
				StatusCode: 404,
			},
			expectedSummary: "volume catalogN.schemaN.volumeN does not exist",
		},
		{
			err: &apierr.APIError{
				StatusCode: 500,
				Message:    "Internal Server Error",
			},
			expectedSummary: "cannot read volume catalogN.schemaN.volumeN: Internal Server Error",
		},
	}

	for _, tc := range tcases {
		m := mocks.NewMockWorkspaceClient(t)
		api := m.GetMockVolumesAPI()
		api.EXPECT().ReadByName(mock.Anything, "catalogN.schemaN.volumeN").Return(nil, tc.err)
		b.SetWorkpaceClient(m.WorkspaceClient)

		diags := ValidateArtifactPath().Apply(ctx, b)
		assertDiags(t, diags, tc.expectedSummary)
	}
}

func invalidVolumePaths() []string {
	return []string{
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
}

func TestExtractVolumeFromPath(t *testing.T) {
	catalogName, schemaName, volumeName, err := extractVolumeFromPath("/Volumes/main/my_schema/my_volume")
	require.NoError(t, err)
	assert.Equal(t, "main", catalogName)
	assert.Equal(t, "my_schema", schemaName)
	assert.Equal(t, "my_volume", volumeName)

	for _, p := range invalidVolumePaths() {
		_, _, _, err := extractVolumeFromPath(p)
		assert.EqualError(t, err, "expected UC volume path to be in the format /Volumes/<catalog>/<schema>/<volume>/..., got "+p)
	}
}

func TestValidateArtifactPathWithInvalidPaths(t *testing.T) {
	for _, p := range invalidVolumePaths() {
		b := &bundle.Bundle{
			Config: config.Root{
				Workspace: config.Workspace{
					ArtifactPath: p,
				},
			},
		}

		bundletest.SetLocation(b, "workspace.artifact_path", []dyn.Location{{File: "config.yml", Line: 1, Column: 2}})

		diags := ValidateArtifactPath().Apply(context.Background(), b)
		require.Equal(t, diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   "expected UC volume path to be in the format /Volumes/<catalog>/<schema>/<volume>/..., got " + p,
			Locations: []dyn.Location{{File: "config.yml", Line: 1, Column: 2}},
			Paths:     []dyn.Path{dyn.MustPathFromString("workspace.artifact_path")},
		}}, diags)
	}
}

func TestFindVolumeInBundle(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"foo": {
						CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
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
	path, locations, ok := findVolumeInBundle(b.Config, "main", "my_schema", "my_volume")
	assert.True(t, ok)
	assert.Equal(t, []dyn.Location{{
		File:   "volume.yml",
		Line:   1,
		Column: 2,
	}}, locations)
	assert.Equal(t, dyn.MustPathFromString("resources.volumes.foo"), path)

	// wrong volume name
	_, _, ok = findVolumeInBundle(b.Config, "main", "my_schema", "doesnotexist")
	assert.False(t, ok)

	// wrong schema name
	_, _, ok = findVolumeInBundle(b.Config, "main", "doesnotexist", "my_volume")
	assert.False(t, ok)

	// wrong catalog name
	_, _, ok = findVolumeInBundle(b.Config, "doesnotexist", "my_schema", "my_volume")
	assert.False(t, ok)

	// schema name is interpolated but does not have the right prefix. In this case
	// we should not match the volume.
	b.Config.Resources.Volumes["foo"].SchemaName = "${foo.bar.baz}"
	_, _, ok = findVolumeInBundle(b.Config, "main", "my_schema", "my_volume")
	assert.False(t, ok)

	// schema name is interpolated.
	b.Config.Resources.Volumes["foo"].SchemaName = "${resources.schemas.my_schema.name}"
	path, locations, ok = findVolumeInBundle(b.Config, "main", "valuedoesnotmatter", "my_volume")
	assert.True(t, ok)
	assert.Equal(t, []dyn.Location{{
		File:   "volume.yml",
		Line:   1,
		Column: 2,
	}}, locations)
	assert.Equal(t, dyn.MustPathFromString("resources.volumes.foo"), path)
}
