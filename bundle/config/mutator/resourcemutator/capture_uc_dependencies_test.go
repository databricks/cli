package resourcemutator

import (
	"github.com/databricks/cli/libs/qualitymonitors"
	"github.com/databricks/cli/libs/sdktypes/uc/registeredmodels/v1"
	"github.com/databricks/cli/libs/utils"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdktypes/modelserving/v1"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"github.com/databricks/cli/libs/sdktypes/vectorsearch/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shared bundle with schemas for resolveSchema tests.
func bundleWithSchemas() *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"schema1": {CreateSchema: resources.CreateSchema{CatalogName: new("catalog1"), Name: new("foobar")}},
					"schema2": {CreateSchema: resources.CreateSchema{CatalogName: new("catalog2"), Name: new("foobar")}},
					"schema3": {CreateSchema: resources.CreateSchema{CatalogName: new("catalog1"), Name: new("barfoo")}},
				},
			},
		},
	}
}

// Shared bundle with catalogs for resolveCatalog tests.
func bundleWithCatalogs() *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"dev_catalog":  {CreateCatalog: resources.CreateCatalog{Name: new("catalog1")}},
					"prod_catalog": {CreateCatalog: resources.CreateCatalog{Name: new("catalog2")}},
				},
			},
		},
	}
}

func TestResolveSchema(t *testing.T) {
	b := bundleWithSchemas()

	tests := []struct {
		name        string
		catalogName string
		schemaName  string
		expected    string
	}{
		{"match_catalog1_foobar", "catalog1", "foobar", "${resources.schemas.schema1.name}"},
		{"match_catalog2_foobar", "catalog2", "foobar", "${resources.schemas.schema2.name}"},
		{"match_catalog1_barfoo", "catalog1", "barfoo", "${resources.schemas.schema3.name}"},
		{"no_match_wrong_catalog", "catalogX", "foobar", "foobar"},
		{"no_match_wrong_schema", "catalog1", "schemaX", "schemaX"},
		{"empty_catalog", "", "foobar", "foobar"},
		{"empty_schema", "catalog1", "", ""},
		{"both_empty", "", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, resolveSchema(b, tc.catalogName, tc.schemaName))
		})
	}
}

func TestResolveCatalog(t *testing.T) {
	b := bundleWithCatalogs()

	tests := []struct {
		name        string
		catalogName string
		expected    string
	}{
		{"match_catalog1", "catalog1", "${resources.catalogs.dev_catalog.name}"},
		{"match_catalog2", "catalog2", "${resources.catalogs.prod_catalog.name}"},
		{"no_match", "catalogX", "catalogX"},
		{"empty", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, resolveCatalog(b, tc.catalogName))
		})
	}
}

// Test that all resource types are wired correctly by defining a catalog, schema,
// and one of each resource type in a single bundle. Also verifies the ordering fix:
// schemas must be resolved last since their CatalogName gets mutated.
func TestCaptureUCDependencies(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"my_catalog": {CreateCatalog: resources.CreateCatalog{Name: new("mycatalog")}},
				},
				Schemas: map[string]*resources.Schema{
					"my_schema": {CreateSchema: resources.CreateSchema{CatalogName: new("mycatalog"), Name: new("myschema")}},
				},
				Volumes: map[string]*resources.Volume{
					"my_volume": {CreateVolumeRequestContent: resources.CreateVolumeRequestContent{
						CatalogName: new("mycatalog"), SchemaName: new("myschema"),
					}},
				},
				RegisteredModels: map[string]*resources.RegisteredModel{
					"my_model": {CreateRegisteredModelRequest: registeredmodels.CreateRegisteredModelRequest{
						CatalogName: new("mycatalog"), SchemaName: new("myschema"),
					}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my_pipeline": {CreatePipelineRequest: pipelines.CreatePipelineRequest{
						Catalog: new("mycatalog"), Schema: new("myschema"),
					}},
				},
				QualityMonitors: map[string]*resources.QualityMonitor{
					"my_monitor": {CreateMonitor: qualitymonitors.CreateMonitor{
						OutputSchemaName: "mycatalog.myschema",
					}},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"my_endpoint": {CreateInferenceEndpointRequest: modelserving.CreateInferenceEndpointRequest{
						AiGateway: &modelserving.AiGatewayConfig{
							InferenceTableConfig: &modelserving.InferenceTableConfig{
								CatalogName: new("mycatalog"), SchemaName: new("myschema"),
							},
						},
					}},
				},
				ModelServices: map[string]*resources.ModelService{
					"my_model_service": {ModelServiceConfig: resources.ModelServiceConfig{
						Parent: "schemas/mycatalog.myschema", ModelServiceId: "myservice",
					}},
				},
				VectorSearchIndexes: map[string]*resources.VectorSearchIndex{
					"my_index": {CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{
						Name: new("mycatalog.myschema.myindex"),
					}},
				},
				McpServices: map[string]*resources.McpService{
					"my_mcp_service": {McpServiceConfig: resources.McpServiceConfig{
						Parent: "schemas/mycatalog.myschema", McpServiceId: "mymcp",
					}},
				},
				ModelProviderServices: map[string]*resources.ModelProviderService{
					"my_mps": {ModelProviderServiceConfig: resources.ModelProviderServiceConfig{
						Parent: "schemas/mycatalog.myschema", ModelProviderServiceId: "mymps",
					}},
				},
			},
		},
	}

	d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
	require.Nil(t, d)

	schemaRef := "${resources.schemas.my_schema.name}"
	catalogRef := "${resources.catalogs.my_catalog.name}"

	// Schema catalog dependency.
	assert.Equal(t, catalogRef, utils.Deref(b.Config.Resources.Schemas["my_schema"].CatalogName))

	// Volume.
	assert.Equal(t, schemaRef, utils.Deref(b.Config.Resources.Volumes["my_volume"].SchemaName))
	assert.Equal(t, catalogRef, utils.Deref(b.Config.Resources.Volumes["my_volume"].CatalogName))

	// Registered model.
	assert.Equal(t, schemaRef, utils.Deref(b.Config.Resources.RegisteredModels["my_model"].SchemaName))
	assert.Equal(t, catalogRef, utils.Deref(b.Config.Resources.RegisteredModels["my_model"].CatalogName))

	// Pipeline.
	assert.Equal(t, schemaRef, utils.Deref(b.Config.Resources.Pipelines["my_pipeline"].Schema))
	assert.Equal(t, catalogRef, utils.Deref(b.Config.Resources.Pipelines["my_pipeline"].Catalog))

	// Quality monitor (compound "catalog.schema" field).
	assert.Equal(t, catalogRef+"."+schemaRef, b.Config.Resources.QualityMonitors["my_monitor"].OutputSchemaName)

	// Model serving endpoint.
	itc := b.Config.Resources.ModelServingEndpoints["my_endpoint"].AiGateway.InferenceTableConfig
	assert.Equal(t, schemaRef, utils.Deref(itc.SchemaName))
	assert.Equal(t, catalogRef, utils.Deref(itc.CatalogName))

	// Model service (compound "schemas/{catalog}.{schema}" parent field).
	assert.Equal(t, "schemas/"+catalogRef+"."+schemaRef, b.Config.Resources.ModelServices["my_model_service"].Parent)

	// Vector search index (three-part "catalog.schema.index" name).
	assert.Equal(t, catalogRef+"."+schemaRef+".myindex", utils.Deref(b.Config.Resources.VectorSearchIndexes["my_index"].Name))

	// MCP service (same compound parent field).
	assert.Equal(t, "schemas/"+catalogRef+"."+schemaRef, b.Config.Resources.McpServices["my_mcp_service"].Parent)

	// Model provider service (same compound parent field).
	assert.Equal(t, "schemas/"+catalogRef+"."+schemaRef, b.Config.Resources.ModelProviderServices["my_mps"].Parent)
}

// Pipeline schema and target are mutually exclusive; only the populated field
// should be resolved.
func TestCaptureUCDependenciesPipelineSchemaTarget(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"s": {CreateSchema: resources.CreateSchema{CatalogName: new("c"), Name: new("n")}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"with_schema": {CreatePipelineRequest: pipelines.CreatePipelineRequest{Catalog: new("c"), Schema: new("n")}},
					"with_target": {CreatePipelineRequest: pipelines.CreatePipelineRequest{Catalog: new("c"), Target: new("n")}},
				},
			},
		},
	}

	d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
	require.Nil(t, d)

	ref := "${resources.schemas.s.name}"

	assert.Equal(t, ref, utils.Deref(b.Config.Resources.Pipelines["with_schema"].Schema))
	assert.Empty(t, utils.Deref(b.Config.Resources.Pipelines["with_schema"].Target))

	assert.Equal(t, ref, utils.Deref(b.Config.Resources.Pipelines["with_target"].Target))
	assert.Empty(t, utils.Deref(b.Config.Resources.Pipelines["with_target"].Schema))
}

func TestCaptureUCDependenciesQualityMonitorEdgeCases(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"my_catalog": {CreateCatalog: resources.CreateCatalog{Name: new("mycatalog")}},
				},
				Schemas: map[string]*resources.Schema{
					"my_schema": {CreateSchema: resources.CreateSchema{CatalogName: new("mycatalog"), Name: new("myschema")}},
				},
				QualityMonitors: map[string]*resources.QualityMonitor{
					"catalog_only": {CreateMonitor: qualitymonitors.CreateMonitor{OutputSchemaName: "mycatalog.other"}},
					"no_match":     {CreateMonitor: qualitymonitors.CreateMonitor{OutputSchemaName: "other.other"}},
					"empty":        {CreateMonitor: qualitymonitors.CreateMonitor{OutputSchemaName: ""}},
					"no_dot":       {CreateMonitor: qualitymonitors.CreateMonitor{OutputSchemaName: "nodot"}},
					"nil_monitor":  nil,
				},
			},
		},
	}

	d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
	require.Nil(t, d)

	assert.Equal(t, "${resources.catalogs.my_catalog.name}.other", b.Config.Resources.QualityMonitors["catalog_only"].OutputSchemaName)
	assert.Equal(t, "other.other", b.Config.Resources.QualityMonitors["no_match"].OutputSchemaName)
	assert.Empty(t, b.Config.Resources.QualityMonitors["empty"].OutputSchemaName)
	assert.Equal(t, "nodot", b.Config.Resources.QualityMonitors["no_dot"].OutputSchemaName)
	assert.Nil(t, b.Config.Resources.QualityMonitors["nil_monitor"])
}

func TestCaptureUCDependenciesModelServingEndpointEdgeCases(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"my_catalog": {CreateCatalog: resources.CreateCatalog{Name: new("mycatalog")}},
				},
				Schemas: map[string]*resources.Schema{
					"my_schema": {CreateSchema: resources.CreateSchema{CatalogName: new("mycatalog"), Name: new("myschema")}},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					// AutoCaptureConfig path.
					"auto_capture": {CreateInferenceEndpointRequest: modelserving.CreateInferenceEndpointRequest{
						Config: &modelserving.EndpointCoreConfig{
							//nolint:staticcheck // SA1019: deprecated AutoCaptureConfigInput kept for bundle config compatibility
							AutoCaptureConfig: &modelserving.AutoCaptureConfig{
								CatalogName: new("mycatalog"), SchemaName: new("myschema"),
							},
						},
					}},
					// No match.
					"no_match": {CreateInferenceEndpointRequest: modelserving.CreateInferenceEndpointRequest{
						AiGateway: &modelserving.AiGatewayConfig{
							InferenceTableConfig: &modelserving.InferenceTableConfig{
								CatalogName: new("other"), SchemaName: new("other"),
							},
						},
					}},
					// Various nil nesting levels.
					"nil_gateway":         {CreateInferenceEndpointRequest: modelserving.CreateInferenceEndpointRequest{}},
					"nil_inference_table": {CreateInferenceEndpointRequest: modelserving.CreateInferenceEndpointRequest{AiGateway: &modelserving.AiGatewayConfig{}}},
					"nil_endpoint":        nil,
				},
			},
		},
	}

	d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
	require.Nil(t, d)

	schemaRef := "${resources.schemas.my_schema.name}"
	catalogRef := "${resources.catalogs.my_catalog.name}"

	acc := b.Config.Resources.ModelServingEndpoints["auto_capture"].Config.AutoCaptureConfig
	assert.Equal(t, schemaRef, utils.Deref(acc.SchemaName))
	assert.Equal(t, catalogRef, utils.Deref(acc.CatalogName))

	itc := b.Config.Resources.ModelServingEndpoints["no_match"].AiGateway.InferenceTableConfig
	assert.Equal(t, "other", utils.Deref(itc.CatalogName))
	assert.Equal(t, "other", utils.Deref(itc.SchemaName))

	assert.Nil(t, b.Config.Resources.ModelServingEndpoints["nil_endpoint"])
}

func TestCaptureUCDependenciesVectorSearchIndexEdgeCases(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"my_catalog": {CreateCatalog: resources.CreateCatalog{Name: new("mycatalog")}},
				},
				Schemas: map[string]*resources.Schema{
					"my_schema": {CreateSchema: resources.CreateSchema{CatalogName: new("mycatalog"), Name: new("myschema")}},
				},
				VectorSearchIndexes: map[string]*resources.VectorSearchIndex{
					"catalog_only": {CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{Name: new("mycatalog.other.myindex")}},
					"no_match":     {CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{Name: new("other.other.myindex")}},
					"empty":        {CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{Name: new("")}},
					"two_part":     {CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{Name: new("mycatalog.myschema")}},
					"nil_index":    nil,
				},
			},
		},
	}

	d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
	require.Nil(t, d)

	assert.Equal(t, "${resources.catalogs.my_catalog.name}.other.myindex", utils.Deref(b.Config.Resources.VectorSearchIndexes["catalog_only"].Name))
	assert.Equal(t, "other.other.myindex", utils.Deref(b.Config.Resources.VectorSearchIndexes["no_match"].Name))
	assert.Empty(t, utils.Deref(b.Config.Resources.VectorSearchIndexes["empty"].Name))
	assert.Equal(t, "mycatalog.myschema", utils.Deref(b.Config.Resources.VectorSearchIndexes["two_part"].Name))
	assert.Nil(t, b.Config.Resources.VectorSearchIndexes["nil_index"])
}

// Nil and empty resources should not panic.
func TestCaptureUCDependenciesNilResources(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas:               map[string]*resources.Schema{"nil": nil, "empty": {}},
				Catalogs:              map[string]*resources.Catalog{"nil": nil, "empty": {}},
				Volumes:               map[string]*resources.Volume{"nil": nil, "empty": {}},
				RegisteredModels:      map[string]*resources.RegisteredModel{"nil": nil, "empty": {}},
				Pipelines:             map[string]*resources.Pipeline{"nil": nil, "empty": {}},
				QualityMonitors:       map[string]*resources.QualityMonitor{"nil": nil, "empty": {}},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{"nil": nil, "empty": {}},
				VectorSearchIndexes:   map[string]*resources.VectorSearchIndex{"nil": nil, "empty": {}},
			},
		},
	}

	d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
	require.Nil(t, d)
}

func TestSplitUCName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		n        int
		expected []string
		ok       bool
	}{
		{"three_part", "catalog.schema.index", 3, []string{"catalog", "schema", "index"}, true},
		{"two_part", "catalog.schema", 2, []string{"catalog", "schema"}, true},
		{"too_many_components", "a.b.c.d", 3, nil, false},
		{"too_few_components", "a.b", 3, nil, false},
		{"single_component", "a", 3, nil, false},
		{"empty", "", 3, nil, false},
		{"reference_over_counts", "${resources.catalogs.c.name}.schema.index", 3, nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parts, ok := splitUCName(tc.input, tc.n)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.expected, parts)
		})
	}
}

// A vector search index name that already contains a ${resources...} reference is left
// untouched: a mix of references and literals is not supported. Only a fully literal name
// with exactly three components has its catalog and schema captured.
func TestCaptureUCDependenciesVectorSearchIndexReferences(t *testing.T) {
	newBundle := func(indexName string) *bundle.Bundle {
		return &bundle.Bundle{
			Config: config.Root{
				Resources: config.Resources{
					Catalogs: map[string]*resources.Catalog{
						"dev_catalog": {CreateCatalog: resources.CreateCatalog{Name: new("catalog1")}},
					},
					Schemas: map[string]*resources.Schema{
						"my_schema": {CreateSchema: resources.CreateSchema{CatalogName: new("catalog1"), Name: new("myschema")}},
					},
					VectorSearchIndexes: map[string]*resources.VectorSearchIndex{
						"idx": {CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{Name: new(indexName)}},
					},
				},
			},
		}
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"all_literal_captured",
			"catalog1.myschema.myindex",
			"${resources.catalogs.dev_catalog.name}.${resources.schemas.my_schema.name}.myindex",
		},
		{
			"catalog_reference_with_literal_schema_unsupported",
			"${resources.catalogs.dev_catalog.name}.myschema.myindex",
			"${resources.catalogs.dev_catalog.name}.myschema.myindex",
		},
		{
			"fully_referenced_unchanged",
			"${resources.catalogs.dev_catalog.name}.${resources.schemas.my_schema.name}.myindex",
			"${resources.catalogs.dev_catalog.name}.${resources.schemas.my_schema.name}.myindex",
		},
		{
			"more_than_three_components_skipped",
			"catalog1.myschema.my.index",
			"catalog1.myschema.my.index",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := newBundle(tc.input)
			d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
			require.Nil(t, d)
			assert.Equal(t, tc.expected, utils.Deref(b.Config.Resources.VectorSearchIndexes["idx"].Name))
		})
	}
}
