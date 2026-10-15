package mutator

import (
	"testing"

	"github.com/databricks/cli/libs/utils"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/require"
)

func TestInitializeVolumePaths(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"my": {
						CreateSchema: resources.CreateSchema{
							CatalogName: new("main"),
							Name:        new("myschema"),
						},
					},
				},
				Volumes: map[string]*resources.Volume{
					"bar": {
						CreateVolumeRequestContent: resources.CreateVolumeRequestContent{
							CatalogName: new("main"),
							SchemaName:  new("myschema"),
							Name:        new("volbar"),
						},
					},
					// foo references the schema's name; InitializeVolumePaths resolves
					// it locally to compute the path without rewriting schema_name.
					"foo": {
						CreateVolumeRequestContent: resources.CreateVolumeRequestContent{
							CatalogName: new("main"),
							SchemaName:  new("${resources.schemas.my.name}"),
							Name:        new("volfoo"),
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
	require.Equal(t, "${resources.schemas.my.name}", utils.Deref(foo.SchemaName))
}

func TestInitializeVolumePaths_UnresolvedReference(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					// The reference cannot be resolved locally, so it is embedded into
					// the path verbatim to be resolved later during plan or deploy.
					"foo": {
						CreateVolumeRequestContent: resources.CreateVolumeRequestContent{
							CatalogName: new("main"),
							SchemaName:  new("${resources.schemas.missing.name}"),
							Name:        new("volfoo"),
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, InitializeVolumePaths())
	require.NoError(t, diags.Error())
	require.Equal(t, "/Volumes/main/${resources.schemas.missing.name}/volfoo", b.Config.Resources.Volumes["foo"].VolumePath)
	// The schema_name reference itself must be preserved, not rewritten.
	require.Equal(t, "${resources.schemas.missing.name}", utils.Deref(b.Config.Resources.Volumes["foo"].SchemaName))
}

func TestInitializeVolumePaths_RejectsUserProvidedPath(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"foo": {
						CreateVolumeRequestContent: resources.CreateVolumeRequestContent{
							CatalogName: new("main"),
							SchemaName:  new("myschema"),
							Name:        new("volfoo"),
						},
						VolumePath: "/Volumes/bogus/path/set-by-user",
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, InitializeVolumePaths())
	require.ErrorContains(t, diags.Error(), "volume_path is computed and read-only")
}

func TestVolumePathPipeline_ResolvesCrossVolumeReference(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"bar": {
						CreateVolumeRequestContent: resources.CreateVolumeRequestContent{
							CatalogName: new("main"),
							SchemaName:  new("myschema"),
							Name:        new("volbar"),
						},
					},
					"foo": {
						CreateVolumeRequestContent: resources.CreateVolumeRequestContent{
							CatalogName: new("main"),
							SchemaName:  new("myschema"),
							Name:        new("volfoo"),
							Comment:     new("${resources.volumes.bar.volume_path}"),
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
	require.Equal(t, "/Volumes/main/myschema/volbar", utils.Deref(b.Config.Resources.Volumes["foo"].Comment))
}
