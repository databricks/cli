package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/testserver"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/sdk-go/core/apierr"
	"github.com/stretchr/testify/assert"
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
						CreateVolumeRequestContent: resources.CreateVolumeRequestContent{
							CatalogName: new("catalogN"),
							Name:        new("volumeN"),
							SchemaName:  new("schemaN"),
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "workspace.artifact_path", []diag.Location{{File: "file", Line: 1, Column: 1}})
	bundletest.SetLocation(b, "resources.volumes.foo", []diag.Location{{File: "file", Line: 2, Column: 2}})

	ctx := t.Context()
	b.SetWorkpaceClient(volumeClients(t, 404, ""))

	diags := ValidateArtifactPath().Apply(ctx, b)
	assert.Equal(t, diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  "volume catalogN.schemaN.volumeN does not exist",
		Locations: []diag.Location{
			{File: "file", Line: 1, Column: 1},
			{File: "file", Line: 2, Column: 2},
		},
		Paths: structpath.MustParsePaths(
			"workspace.artifact_path",
			"resources.volumes.foo",
		),
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

	bundletest.SetLocation(b, "workspace.artifact_path", []diag.Location{{File: "file", Line: 1, Column: 1}})
	assertDiags := func(t *testing.T, diags diag.Diagnostics, expected string) {
		assert.Len(t, diags, 1)
		assert.Equal(t, diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   expected,
			Locations: []diag.Location{{File: "file", Line: 1, Column: 1}},
			Paths:     structpath.NewPathSlice("workspace", "artifact_path"),
		}}, diags)
	}

	ctx := t.Context()

	tcases := []struct {
		statusCode      int
		message         string
		expectedSummary string
	}{
		{
			statusCode:      403,
			message:         "User does not have USE SCHEMA on Schema 'catalogN.schemaN'",
			expectedSummary: "cannot access volume catalogN.schemaN.volumeN: " + apiErrorText("User does not have USE SCHEMA on Schema 'catalogN.schemaN'"),
		},
		{
			statusCode:      404,
			expectedSummary: "volume catalogN.schemaN.volumeN does not exist",
		},
		{
			statusCode:      500,
			message:         "Internal Server Error",
			expectedSummary: "cannot read volume catalogN.schemaN.volumeN: " + apiErrorText("Internal Server Error"),
		},
	}

	for _, tc := range tcases {
		b.SetWorkpaceClient(volumeClients(t, tc.statusCode, tc.message))

		diags := ValidateArtifactPath().Apply(ctx, b)
		assertDiags(t, diags, tc.expectedSummary)
	}
}

// volumeClients returns clients for a workspace that fails every volume GET
// with the given status code and message.
func volumeClients(t *testing.T, statusCode int, message string) *sdkclients.Clients {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.1/unity-catalog/volumes/catalogN.schemaN.volumeN", func(testserver.Request) any {
		return testserver.Response{
			StatusCode: statusCode,
			Body:       map[string]string{"message": message},
		}
	})
	w, err := sdkclients.New(t.Context(), &sdkconfig.Config{
		Host:    server.URL,
		Token:   "testtoken",
		Loaders: []sdkconfig.Loader{sdkconfig.ConfigAttributes},
	})
	require.NoError(t, err)
	return w
}

// apiErrorText is how an API error with the given message renders.
func apiErrorText(message string) string {
	return apierr.FromHTTPError(500, nil, []byte(`{"message":"`+message+`"}`)).Error()
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

		bundletest.SetLocation(b, "workspace.artifact_path", []diag.Location{{File: "config.yml", Line: 1, Column: 2}})

		diags := ValidateArtifactPath().Apply(t.Context(), b)
		require.Equal(t, diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   "expected UC volume path to be in the format /Volumes/<catalog>/<schema>/<volume>/..., got " + p,
			Locations: []diag.Location{{File: "config.yml", Line: 1, Column: 2}},
			Paths:     structpath.NewPathSlice("workspace", "artifact_path"),
		}}, diags)
	}
}

func TestFindVolumeInBundle(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"foo": {
						CreateVolumeRequestContent: resources.CreateVolumeRequestContent{
							CatalogName: new("main"),
							Name:        new("my_volume"),
							SchemaName:  new("my_schema"),
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.volumes.foo", []diag.Location{
		{
			File:   "volume.yml",
			Line:   1,
			Column: 2,
		},
	})

	// volume is in DAB.
	path, locations, ok := findVolumeInBundle(b.Config, "main", "my_schema", "my_volume")
	assert.True(t, ok)
	assert.Equal(t, []diag.Location{{
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
	b.Config.Resources.Volumes["foo"].SchemaName = new("${foo.bar.baz}")
	_, _, ok = findVolumeInBundle(b.Config, "main", "my_schema", "my_volume")
	assert.False(t, ok)

	// schema name is interpolated.
	b.Config.Resources.Volumes["foo"].SchemaName = new("${resources.schemas.my_schema.name}")
	path, locations, ok = findVolumeInBundle(b.Config, "main", "valuedoesnotmatter", "my_volume")
	assert.True(t, ok)
	assert.Equal(t, []diag.Location{{
		File:   "volume.yml",
		Line:   1,
		Column: 2,
	}}, locations)
	assert.Equal(t, dyn.MustPathFromString("resources.volumes.foo"), path)
}
