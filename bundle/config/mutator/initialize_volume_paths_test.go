package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/require"
)

func TestInitializeVolumePaths(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"my": {
						CreateSchema: catalog.CreateSchema{
							CatalogName: "main",
							Name:        "myschema",
						},
					},
				},
				Volumes: map[string]*resources.Volume{
					"bar": {
						CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
							CatalogName: "main",
							SchemaName:  "myschema",
							Name:        "volbar",
						},
					},
					// foo references the schema's name; InitializeVolumePaths resolves
					// it locally to compute the path without rewriting schema_name.
					"foo": {
						CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
							CatalogName: "main",
							SchemaName:  "${resources.schemas.my.name}",
							Name:        "volfoo",
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, InitializeVolumePaths())
	require.NoError(t, diags.Error())

	require.Equal(t, "/Volumes/main/myschema/volbar", b.Config.Resources.Volumes["bar"].VolumePath)

	foo := b.Config.Resources.Volumes["foo"]
	require.Equal(t, "/Volumes/main/myschema/volfoo", foo.VolumePath)
	// The schema_name reference must be preserved, not replaced with the resolved value.
	require.Equal(t, "${resources.schemas.my.name}", foo.SchemaName)
}

func TestInitializeVolumePaths_UnresolvedReference(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					// The referenced schema does not exist, so the path is left unset.
					"foo": {
						CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
							CatalogName: "main",
							SchemaName:  "${resources.schemas.missing.name}",
							Name:        "volfoo",
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, InitializeVolumePaths())
	require.NoError(t, diags.Error())
	require.Empty(t, b.Config.Resources.Volumes["foo"].VolumePath)
}

func TestInitializeVolumePaths_MalformedReference(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					// A malformed reference must not leak into the computed path.
					"foo": {
						CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
							CatalogName: "${resources.volumes.bar.bad..syntax}",
							SchemaName:  "myschema",
							Name:        "volfoo",
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, InitializeVolumePaths())
	require.NoError(t, diags.Error())
	require.Empty(t, b.Config.Resources.Volumes["foo"].VolumePath)
}

func TestVolumePathPipeline_ResolvesCrossVolumeReference(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"bar": {
						CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
							CatalogName: "main",
							SchemaName:  "myschema",
							Name:        "volbar",
						},
					},
					"foo": {
						CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
							CatalogName: "main",
							SchemaName:  "myschema",
							Name:        "volfoo",
							Comment:     "${resources.volumes.bar.volume_path}",
						},
					},
				},
			},
		},
	}

	diags := bundle.ApplySeq(
		t.Context(),
		b,
		InitializeVolumePaths(),
		ResolveVolumePathReferencesOnlyResources(),
	)
	require.NoError(t, diags.Error())
	require.Equal(t, "/Volumes/main/myschema/volbar", b.Config.Resources.Volumes["bar"].VolumePath)
	require.Equal(t, "/Volumes/main/myschema/volbar", b.Config.Resources.Volumes["foo"].Comment)
}
