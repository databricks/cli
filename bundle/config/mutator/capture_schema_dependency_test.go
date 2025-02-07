package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaptureSchemaDependencyForVolumeWithEmptySchemas(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"nilschema":   nil,
					"emptyschema": {},
				},
				Volumes: map[string]*resources.Volume{
					"volume1": {
						CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
							CatalogName: "catalog1",
							SchemaName:  "foobar",
						},
					},
					"nilVolume":   nil,
					"emptyVolume": {},
				},
			},
		},
	}

	d := bundle.Apply(context.Background(), b, CaptureSchemaDependency())
	require.Nil(t, d)

	assert.Equal(t, "foobar", b.Config.Resources.Volumes["volume1"].CreateVolumeRequestContent.SchemaName)
	assert.Nil(t, b.Config.Resources.Volumes["nilVolume"])
	assert.Nil(t, b.Config.Resources.Volumes["emptyVolume"].CreateVolumeRequestContent)
}

func TestCaptureSchemaDependencyForPipelinesWithEmptySchemas(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"nilschema":   nil,
					"emptyschema": {},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline1": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "catalog1",
							Schema:  "foobar",
						},
					},
					"nilPipeline":   nil,
					"emptyPipeline": {},
				},
			},
		},
	}

	d := bundle.Apply(context.Background(), b, CaptureSchemaDependency())
	require.Nil(t, d)

	assert.Equal(t, "foobar", b.Config.Resources.Pipelines["pipeline1"].Schema)
	assert.Nil(t, b.Config.Resources.Pipelines["nilPipeline"])
	assert.Nil(t, b.Config.Resources.Pipelines["emptyPipeline"].PipelineSpec)
}
