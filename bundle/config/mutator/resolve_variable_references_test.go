package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/require"
)

func TestResolveVariableReferencesWithSourceLinkedDeployment(t *testing.T) {
	testCases := []struct {
		enabled bool
		assert  func(t *testing.T, b *bundle.Bundle)
	}{
		{
			true,
			func(t *testing.T, b *bundle.Bundle) {
				// Variables that use workspace file path should have SyncRootValue during resolution phase
				require.Equal(t, "sync/root/path", b.Config.Resources.Pipelines["pipeline1"].Configuration["source"])

				// The file path itself should remain the same
				require.Equal(t, "file/path", b.Config.Workspace.FilePath)
			},
		},
		{
			false,
			func(t *testing.T, b *bundle.Bundle) {
				require.Equal(t, "file/path", b.Config.Resources.Pipelines["pipeline1"].Configuration["source"])
				require.Equal(t, "file/path", b.Config.Workspace.FilePath)
			},
		},
	}

	for _, testCase := range testCases {
		b := &bundle.Bundle{
			SyncRootPath: "sync/root/path",
			Config: config.Root{
				Presets: config.Presets{
					SourceLinkedDeployment: &testCase.enabled,
				},
				Workspace: config.Workspace{
					FilePath: "file/path",
				},
				Resources: config.Resources{
					Pipelines: map[string]*resources.Pipeline{
						"pipeline1": {
							CreatePipeline: pipelines.CreatePipeline{
								Configuration: map[string]string{
									"source": "${workspace.file_path}",
								},
							},
						},
					},
				},
			},
		}

		diags := bundle.Apply(t.Context(), b, ResolveVariableReferencesOnlyResources("workspace"))
		require.NoError(t, diags.Error())
		testCase.assert(t, b)
	}
}

func TestResolveVolumePathReferencesOnlyResources(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"foo": {
						VolumePath: "/Volumes/main/myschema/myvolume",
					},
					"volumePathRef": {},
					"otherRef":      {},
				},
			},
		},
	}

	b.Config.Resources.Volumes["volumePathRef"].Comment = "${resources.volumes.foo.volume_path}"
	b.Config.Resources.Volumes["otherRef"].Comment = "${resources.volumes.foo.name}"

	diags := bundle.Apply(t.Context(), b, ResolveVolumePathReferencesOnlyResources())
	require.NoError(t, diags.Error())
	require.Equal(t, "/Volumes/main/myschema/myvolume", b.Config.Resources.Volumes["volumePathRef"].Comment)
	require.Equal(t, "${resources.volumes.foo.name}", b.Config.Resources.Volumes["otherRef"].Comment)
}

func TestResolveVolumePathReferencesOnlyResources_MissingTarget(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"foo": {},
				},
			},
		},
	}
	b.Config.Resources.Volumes["foo"].Comment = "${resources.volumes.missing.volume_path}"

	diags := bundle.Apply(t.Context(), b, ResolveVolumePathReferencesOnlyResources())
	require.ErrorContains(t, diags.Error(), "reference does not exist: ${resources.volumes.missing.volume_path}")
}

func TestResolveVolumePathReferencesOnlyResources_RejectsUnprefixedSchemaReferences(t *testing.T) {
	root, loadDiags := config.LoadFromBytes("databricks.yml", []byte(`resources:
  schemas:
    test:
      name: myschema
      catalog_name: main
  volumes:
    archive:
      name: archive
      schema_name: ${schemas.test.name}
      catalog_name: ${schemas.test.catalog_name}
    ingest:
      name: ingest
      schema_name: ${schemas.test.name}
      catalog_name: ${schemas.test.catalog_name}
`))
	require.NoError(t, loadDiags.Error())
	b := &bundle.Bundle{Config: *root}

	diags := bundle.Apply(t.Context(), b, ResolveVolumePathReferencesOnlyResources())
	require.Len(t, diags, 4)
	require.Error(t, diags.Error())
	for i, want := range []struct {
		volume, field, reference, suggestion string
		line                                 int
	}{
		{"archive", "schema_name", "schemas.test.name", "resources.schemas.test.name", 9},
		{"archive", "catalog_name", "schemas.test.catalog_name", "resources.schemas.test.catalog_name", 10},
		{"ingest", "schema_name", "schemas.test.name", "resources.schemas.test.name", 13},
		{"ingest", "catalog_name", "schemas.test.catalog_name", "resources.schemas.test.catalog_name", 14},
	} {
		d := diags[i]
		require.Equal(t, "volume \""+want.volume+"\" field \""+want.field+"\": reference does not exist: ${"+want.reference+"}", d.Summary)
		require.Equal(t, "did you mean:\n  ${"+want.suggestion+"}", d.Detail)
		require.Len(t, d.Paths, 1)
		require.Equal(t, "resources.volumes."+want.volume+"."+want.field, d.Paths[0].String())
		require.Len(t, d.Locations, 1)
		require.Equal(t, "databricks.yml", d.Locations[0].File)
		require.Equal(t, want.line, d.Locations[0].Line)
	}

	b = &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{"test": {}},
				Volumes: map[string]*resources.Volume{"foo": {}},
			},
		},
	}
	schema := b.Config.Resources.Schemas["test"]
	schema.Name = "myschema"
	schema.CatalogName = "main"
	volume := b.Config.Resources.Volumes["foo"]
	volume.SchemaName = "${resources.schemas.test.name}"
	volume.CatalogName = "${resources.schemas.test.catalog_name}"

	diags = bundle.Apply(t.Context(), b, ResolveVolumePathReferencesOnlyResources())
	require.NoError(t, diags.Error())
	diags = bundle.Apply(t.Context(), b, ResolveVariableReferencesOnlyResources("resources"))
	require.NoError(t, diags.Error())
	require.Equal(t, "myschema", b.Config.Resources.Volumes["foo"].SchemaName)
	require.Equal(t, "main", b.Config.Resources.Volumes["foo"].CatalogName)
}

func TestIsVolumePathReferencePath(t *testing.T) {
	require.True(t, isVolumePathReferencePath(dyn.MustPathFromString("resources.volumes.foo.volume_path")))
	require.False(t, isVolumePathReferencePath(dyn.MustPathFromString("resources.volumes.foo.name")))
	require.False(t, isVolumePathReferencePath(dyn.MustPathFromString("resources.jobs.foo.name")))
}
