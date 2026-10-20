package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPresetsSkipsNilQualityMonitors(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Presets: config.Presets{
				TriggerPauseStatus: config.Paused,
			},
			Resources: config.Resources{
				QualityMonitors: map[string]*resources.QualityMonitor{
					// Python-generated configs can contain null resource entries that
					// bypass AllResourcesHaveValues, so ApplyPresets must skip them.
					"nil_monitor": nil,
					"monitor": {
						TableName: "monitor",
						CreateMonitor: catalog.CreateMonitor{
							OutputSchemaName: "catalog.schema",
							Schedule:         &catalog.MonitorCronSchedule{},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ApplyPresets())
	require.NoError(t, diags.Error())
	assert.Nil(t, b.Config.Resources.QualityMonitors["monitor"].Schedule)
}

func bundleWithSchemaDependents() *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Presets: config.Presets{NamePrefix: "dev_"},
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"my_schema": {CreateSchema: catalog.CreateSchema{CatalogName: "main", Name: "myschema"}},
				},
				Volumes: map[string]*resources.Volume{
					"my_volume": {CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{CatalogName: "main", SchemaName: "myschema"}},
					"other":     {CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{CatalogName: "main", SchemaName: "unrelated"}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my_pipeline": {CreatePipeline: pipelines.CreatePipeline{Catalog: "main", Target: "myschema"}},
				},
				QualityMonitors: map[string]*resources.QualityMonitor{
					"my_monitor": {CreateMonitor: catalog.CreateMonitor{OutputSchemaName: "main.myschema"}},
				},
				VectorSearchIndexes: map[string]*resources.VectorSearchIndex{
					"my_index": {CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{Name: "main.myschema.myindex"}},
				},
				ModelServices: map[string]*resources.ModelService{
					"my_service": {ModelServiceConfig: resources.ModelServiceConfig{Parent: "schemas/main.myschema"}},
				},
			},
		},
	}
}

func TestApplyPresetsPrefixesSchemaReferences(t *testing.T) {
	b := bundleWithSchemaDependents()

	diags := bundle.Apply(t.Context(), b, ApplyPresets())
	require.NoError(t, diags.Error())

	prefixed := normalizePrefix("dev_") + "myschema"
	r := b.Config.Resources
	assert.Equal(t, prefixed, r.Schemas["my_schema"].Name)
	assert.Equal(t, prefixed, r.Volumes["my_volume"].SchemaName)
	assert.Equal(t, "unrelated", r.Volumes["other"].SchemaName)
	assert.Equal(t, prefixed, r.Pipelines["my_pipeline"].Target)
	assert.Empty(t, r.Pipelines["my_pipeline"].Schema)
	assert.Equal(t, "main."+prefixed, r.QualityMonitors["my_monitor"].OutputSchemaName)
	assert.Equal(t, "main."+prefixed+".myindex", r.VectorSearchIndexes["my_index"].Name)
	assert.Equal(t, "schemas/main."+prefixed, r.ModelServices["my_service"].Parent)
}

func TestApplyPresetsSkipNamePrefixForSchemaKeepsReferences(t *testing.T) {
	b := bundleWithSchemaDependents()
	b.Config.Experimental = &config.Experimental{SkipNamePrefixForSchema: true}

	diags := bundle.Apply(t.Context(), b, ApplyPresets())
	require.NoError(t, diags.Error())

	r := b.Config.Resources
	assert.Equal(t, "myschema", r.Schemas["my_schema"].Name)
	assert.Equal(t, "myschema", r.Volumes["my_volume"].SchemaName)
	assert.Equal(t, "main.myschema", r.QualityMonitors["my_monitor"].OutputSchemaName)
}
