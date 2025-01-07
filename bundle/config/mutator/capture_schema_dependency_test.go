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

func TestResolveSchemaDependencyForVolume(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"schema1": {
						CreateSchema: &catalog.CreateSchema{
							CatalogName: "catalog1",
							Name:        "foobar",
						},
					},
					"schema2": {
						CreateSchema: &catalog.CreateSchema{
							CatalogName: "catalog2",
							Name:        "foobar",
						},
					},
					"schema3": {
						CreateSchema: &catalog.CreateSchema{
							CatalogName: "catalog1",
							Name:        "barfoo",
						},
					},
					"nilschema": {},
					"emptyschema": {
						CreateSchema: &catalog.CreateSchema{},
					},
				},
				Volumes: map[string]*resources.Volume{
					"volume1": {
						CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
							CatalogName: "catalog1",
							SchemaName:  "foobar",
						},
					},
					"volume2": {
						CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
							CatalogName: "catalog2",
							SchemaName:  "foobar",
						},
					},
					"volume3": {
						CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
							CatalogName: "catalog1",
							SchemaName:  "barfoo",
						},
					},
					"volume4": {
						CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
							CatalogName: "catalogX",
							SchemaName:  "foobar",
						},
					},
					"volume5": {
						CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
							CatalogName: "catalog1",
							SchemaName:  "schemaX",
						},
					},
				},
			},
		},
	}

	d := bundle.Apply(context.Background(), b, ResolveSchemaDependency())
	require.Nil(t, d)
	assert.Equal(t, b.Config.Resources.Volumes["volume1"].CreateVolumeRequestContent.SchemaName, "${resources.schemas.schema1.name}")
	assert.Equal(t, b.Config.Resources.Volumes["volume2"].CreateVolumeRequestContent.SchemaName, "${resources.schemas.schema2.name}")
	assert.Equal(t, b.Config.Resources.Volumes["volume3"].CreateVolumeRequestContent.SchemaName, "${resources.schemas.schema3.name}")
	assert.Equal(t, b.Config.Resources.Volumes["volume4"].CreateVolumeRequestContent.SchemaName, "foobar")
	assert.Equal(t, b.Config.Resources.Volumes["volume5"].CreateVolumeRequestContent.SchemaName, "schemaX")
}

func TestResolveSchemaDependencyForPipelinesWithTarget(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"schema1": {
						CreateSchema: &catalog.CreateSchema{
							CatalogName: "catalog1",
							Name:        "foobar",
						},
					},
					"schema2": {
						CreateSchema: &catalog.CreateSchema{
							CatalogName: "catalog2",
							Name:        "foobar",
						},
					},
					"schema3": {
						CreateSchema: &catalog.CreateSchema{
							CatalogName: "catalog1",
							Name:        "barfoo",
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline1": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "catalog1",
							Schema:  "foobar",
						},
					},
					"pipeline2": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "catalog2",
							Schema:  "foobar",
						},
					},
					"pipeline3": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "catalog1",
							Schema:  "barfoo",
						},
					},
					"pipeline4": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "catalogX",
							Schema:  "foobar",
						},
					},
					"pipeline5": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "catalog1",
							Schema:  "schemaX",
						},
					},
					"pipeline6": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "",
							Schema:  "foobar",
						},
					},
					"pipeline7": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "",
							Schema:  "",
							Name:    "whatever",
						},
					},
				},
			},
		},
	}

	d := bundle.Apply(context.Background(), b, ResolveSchemaDependency())
	require.Nil(t, d)
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline1"].Schema, "${resources.schemas.schema1.name}")
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline2"].Schema, "${resources.schemas.schema2.name}")
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline3"].Schema, "${resources.schemas.schema3.name}")
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline4"].Schema, "foobar")
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline5"].Schema, "schemaX")
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline6"].Schema, "foobar")
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline7"].Schema, "")

	for _, k := range []string{"pipeline1", "pipeline2", "pipeline3", "pipeline4", "pipeline5", "pipeline6", "pipeline7"} {
		assert.Empty(t, b.Config.Resources.Pipelines[k].Target)
	}
}

func TestResolveSchemaDependencyForPipelinesWithSchema(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"schema1": {
						CreateSchema: &catalog.CreateSchema{
							CatalogName: "catalog1",
							Name:        "foobar",
						},
					},
					"schema2": {
						CreateSchema: &catalog.CreateSchema{
							CatalogName: "catalog2",
							Name:        "foobar",
						},
					},
					"schema3": {
						CreateSchema: &catalog.CreateSchema{
							CatalogName: "catalog1",
							Name:        "barfoo",
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline1": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "catalog1",
							Target:  "foobar",
						},
					},
					"pipeline2": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "catalog2",
							Target:  "foobar",
						},
					},
					"pipeline3": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "catalog1",
							Target:  "barfoo",
						},
					},
					"pipeline4": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "catalogX",
							Target:  "foobar",
						},
					},
					"pipeline5": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "catalog1",
							Target:  "schemaX",
						},
					},
					"pipeline6": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "",
							Target:  "foobar",
						},
					},
					"pipeline7": {
						PipelineSpec: &pipelines.PipelineSpec{
							Catalog: "",
							Target:  "",
							Name:    "whatever",
						},
					},
				},
			},
		},
	}

	d := bundle.Apply(context.Background(), b, ResolveSchemaDependency())
	require.Nil(t, d)
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline1"].Target, "${resources.schemas.schema1.name}")
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline2"].Target, "${resources.schemas.schema2.name}")
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline3"].Target, "${resources.schemas.schema3.name}")
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline4"].Target, "foobar")
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline5"].Target, "schemaX")
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline6"].Target, "foobar")
	assert.Equal(t, b.Config.Resources.Pipelines["pipeline7"].Target, "")

	for _, k := range []string{"pipeline1", "pipeline2", "pipeline3", "pipeline4", "pipeline5", "pipeline6", "pipeline7"} {
		assert.Empty(t, b.Config.Resources.Pipelines[k].Schema)
	}
}
