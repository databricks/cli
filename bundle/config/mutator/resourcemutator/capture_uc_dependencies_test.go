package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shared bundle with schemas for resolveSchema tests.
func bundleWithSchemas() *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"schema1": {CreateSchema: catalog.CreateSchema{CatalogName: "catalog1", Name: "foobar"}},
					"schema2": {CreateSchema: catalog.CreateSchema{CatalogName: "catalog2", Name: "foobar"}},
					"schema3": {CreateSchema: catalog.CreateSchema{CatalogName: "catalog1", Name: "barfoo"}},
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
					"dev_catalog":  {CreateCatalog: catalog.CreateCatalog{Name: "catalog1"}},
					"prod_catalog": {CreateCatalog: catalog.CreateCatalog{Name: "catalog2"}},
				},
			},
		},
	}
}

func dependsOnNodes(lc resources.Lifecycle) []string {
	var nodes []string
	for _, d := range lc.DependsOn {
		nodes = append(nodes, d.Resource)
	}
	return nodes
}

func TestDependsOnUC(t *testing.T) {
	b := bundleWithSchemas()
	b.Config.Resources.Catalogs = bundleWithCatalogs().Config.Resources.Catalogs

	tests := []struct {
		name        string
		catalogName string
		schemaName  string
		expected    []string
	}{
		{"schema_and_catalog", "catalog1", "foobar", []string{"schemas.schema1", "catalogs.dev_catalog"}},
		{"other_catalog", "catalog2", "foobar", []string{"schemas.schema2", "catalogs.prod_catalog"}},
		{"catalog_only", "catalog1", "schemaX", []string{"catalogs.dev_catalog"}},
		{"no_match", "catalogX", "foobar", nil},
		{"empty_catalog", "", "foobar", nil},
		{"empty_schema", "catalogX", "", nil},
		{"both_empty", "", "", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var lc resources.Lifecycle
			assert.Equal(t, tc.schemaName, dependsOnUC(b, &lc, tc.catalogName, tc.schemaName))
			assert.Equal(t, tc.expected, dependsOnNodes(lc))
		})
	}
}

// Test that all resource types are wired correctly by defining a catalog, schema,
// and one of each resource type in a single bundle.
func TestCaptureUCDependencies(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"my_catalog": {CreateCatalog: catalog.CreateCatalog{Name: "mycatalog"}},
				},
				Schemas: map[string]*resources.Schema{
					"my_schema": {CreateSchema: catalog.CreateSchema{CatalogName: "mycatalog", Name: "myschema"}},
				},
				Volumes: map[string]*resources.Volume{
					"my_volume": {CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
						CatalogName: "mycatalog", SchemaName: "myschema",
					}},
				},
				RegisteredModels: map[string]*resources.RegisteredModel{
					"my_model": {CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{
						CatalogName: "mycatalog", SchemaName: "myschema",
					}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my_pipeline": {CreatePipeline: pipelines.CreatePipeline{
						Catalog: "mycatalog", Schema: "myschema",
					}},
				},
				QualityMonitors: map[string]*resources.QualityMonitor{
					"my_monitor": {CreateMonitor: catalog.CreateMonitor{
						OutputSchemaName: "mycatalog.myschema",
					}},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"my_endpoint": {CreateServingEndpoint: serving.CreateServingEndpoint{
						AiGateway: &serving.AiGatewayConfig{
							InferenceTableConfig: &serving.AiGatewayInferenceTableConfig{
								CatalogName: "mycatalog", SchemaName: "myschema",
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
						Name: "mycatalog.myschema.myindex",
					}},
				},
				Features: map[string]*resources.Feature{
					"my_feature": {Feature: ml.Feature{
						FullName: "mycatalog.myschema.myfeature",
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

	catalogOnly := []string{"catalogs.my_catalog"}
	both := []string{"schemas.my_schema", "catalogs.my_catalog"}
	r := b.Config.Resources

	assert.Equal(t, catalogOnly, dependsOnNodes(r.Schemas["my_schema"].Lifecycle))
	assert.Equal(t, both, dependsOnNodes(r.Volumes["my_volume"].Lifecycle))
	assert.Equal(t, both, dependsOnNodes(r.RegisteredModels["my_model"].Lifecycle))
	assert.Equal(t, both, dependsOnNodes(r.Pipelines["my_pipeline"].Lifecycle))
	// Compound "catalog.schema" field.
	assert.Equal(t, both, dependsOnNodes(r.QualityMonitors["my_monitor"].Lifecycle))
	assert.Equal(t, both, dependsOnNodes(r.ModelServingEndpoints["my_endpoint"].Lifecycle))
	// Compound "schemas/{catalog}.{schema}" parent field.
	assert.Equal(t, both, dependsOnNodes(r.ModelServices["my_model_service"].Lifecycle))
	assert.Equal(t, both, dependsOnNodes(r.McpServices["my_mcp_service"].Lifecycle))
	assert.Equal(t, both, dependsOnNodes(r.ModelProviderServices["my_mps"].Lifecycle))
	// Three-part "catalog.schema.index" name.
	assert.Equal(t, both, dependsOnNodes(r.VectorSearchIndexes["my_index"].Lifecycle))
	// Three-part "catalog.schema.feature" full name.
	assert.Equal(t, both, dependsOnNodes(r.Features["my_feature"].Lifecycle))

	// Field values stay literal.
	assert.Equal(t, "myschema", r.Volumes["my_volume"].SchemaName)
	assert.Equal(t, "mycatalog.myschema.myindex", r.VectorSearchIndexes["my_index"].Name)
}

// Pipeline schema and target are mutually exclusive; either one captures the dependency.
func TestCaptureUCDependenciesPipelineSchemaTarget(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"s": {CreateSchema: catalog.CreateSchema{CatalogName: "c", Name: "n"}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"with_schema": {CreatePipeline: pipelines.CreatePipeline{Catalog: "c", Schema: "n"}},
					"with_target": {CreatePipeline: pipelines.CreatePipeline{Catalog: "c", Target: "n"}},
				},
			},
		},
	}

	d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
	require.Nil(t, d)

	assert.Equal(t, []string{"schemas.s"}, dependsOnNodes(b.Config.Resources.Pipelines["with_schema"].Lifecycle))
	assert.Equal(t, []string{"schemas.s"}, dependsOnNodes(b.Config.Resources.Pipelines["with_target"].Lifecycle))
}

func TestCaptureUCDependenciesQualityMonitorEdgeCases(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"my_catalog": {CreateCatalog: catalog.CreateCatalog{Name: "mycatalog"}},
				},
				Schemas: map[string]*resources.Schema{
					"my_schema": {CreateSchema: catalog.CreateSchema{CatalogName: "mycatalog", Name: "myschema"}},
				},
				QualityMonitors: map[string]*resources.QualityMonitor{
					"catalog_only": {CreateMonitor: catalog.CreateMonitor{OutputSchemaName: "mycatalog.other"}},
					"no_match":     {CreateMonitor: catalog.CreateMonitor{OutputSchemaName: "other.other"}},
					"empty":        {CreateMonitor: catalog.CreateMonitor{OutputSchemaName: ""}},
					"no_dot":       {CreateMonitor: catalog.CreateMonitor{OutputSchemaName: "nodot"}},
					"nil_monitor":  nil,
				},
			},
		},
	}

	d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
	require.Nil(t, d)

	qms := b.Config.Resources.QualityMonitors
	assert.Equal(t, []string{"catalogs.my_catalog"}, dependsOnNodes(qms["catalog_only"].Lifecycle))
	assert.Empty(t, dependsOnNodes(qms["no_match"].Lifecycle))
	assert.Empty(t, dependsOnNodes(qms["empty"].Lifecycle))
	assert.Empty(t, dependsOnNodes(qms["no_dot"].Lifecycle))
	assert.Nil(t, qms["nil_monitor"])
}

func TestCaptureUCDependenciesModelServingEndpointEdgeCases(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"my_catalog": {CreateCatalog: catalog.CreateCatalog{Name: "mycatalog"}},
				},
				Schemas: map[string]*resources.Schema{
					"my_schema": {CreateSchema: catalog.CreateSchema{CatalogName: "mycatalog", Name: "myschema"}},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					// AutoCaptureConfig path.
					"auto_capture": {CreateServingEndpoint: serving.CreateServingEndpoint{
						Config: &serving.EndpointCoreConfigInput{
							//nolint:staticcheck // SA1019: deprecated AutoCaptureConfigInput kept for bundle config compatibility
							AutoCaptureConfig: &serving.AutoCaptureConfigInput{
								CatalogName: "mycatalog", SchemaName: "myschema",
							},
						},
					}},
					// No match.
					"no_match": {CreateServingEndpoint: serving.CreateServingEndpoint{
						AiGateway: &serving.AiGatewayConfig{
							InferenceTableConfig: &serving.AiGatewayInferenceTableConfig{
								CatalogName: "other", SchemaName: "other",
							},
						},
					}},
					// Various nil nesting levels.
					"nil_gateway":         {CreateServingEndpoint: serving.CreateServingEndpoint{}},
					"nil_inference_table": {CreateServingEndpoint: serving.CreateServingEndpoint{AiGateway: &serving.AiGatewayConfig{}}},
					"nil_endpoint":        nil,
				},
			},
		},
	}

	d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
	require.Nil(t, d)

	mses := b.Config.Resources.ModelServingEndpoints
	assert.Equal(t, []string{"schemas.my_schema", "catalogs.my_catalog"}, dependsOnNodes(mses["auto_capture"].Lifecycle))
	assert.Empty(t, dependsOnNodes(mses["no_match"].Lifecycle))
	assert.Nil(t, mses["nil_endpoint"])
}

func TestCaptureUCDependenciesVectorSearchIndexEdgeCases(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"my_catalog": {CreateCatalog: catalog.CreateCatalog{Name: "mycatalog"}},
				},
				Schemas: map[string]*resources.Schema{
					"my_schema": {CreateSchema: catalog.CreateSchema{CatalogName: "mycatalog", Name: "myschema"}},
				},
				VectorSearchIndexes: map[string]*resources.VectorSearchIndex{
					"catalog_only": {CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{Name: "mycatalog.other.myindex"}},
					"no_match":     {CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{Name: "other.other.myindex"}},
					"empty":        {CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{Name: ""}},
					"two_part":     {CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{Name: "mycatalog.myschema"}},
					"nil_index":    nil,
				},
			},
		},
	}

	d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
	require.Nil(t, d)

	idxs := b.Config.Resources.VectorSearchIndexes
	assert.Equal(t, []string{"catalogs.my_catalog"}, dependsOnNodes(idxs["catalog_only"].Lifecycle))
	assert.Empty(t, dependsOnNodes(idxs["no_match"].Lifecycle))
	assert.Empty(t, dependsOnNodes(idxs["empty"].Lifecycle))
	assert.Empty(t, dependsOnNodes(idxs["two_part"].Lifecycle))
	assert.Nil(t, idxs["nil_index"])
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

// Only a fully literal three-component vector search index name has its catalog and schema captured.
func TestCaptureUCDependenciesVectorSearchIndexReferences(t *testing.T) {
	newBundle := func(indexName string) *bundle.Bundle {
		return &bundle.Bundle{
			Config: config.Root{
				Resources: config.Resources{
					Catalogs: map[string]*resources.Catalog{
						"dev_catalog": {CreateCatalog: catalog.CreateCatalog{Name: "catalog1"}},
					},
					Schemas: map[string]*resources.Schema{
						"my_schema": {CreateSchema: catalog.CreateSchema{CatalogName: "catalog1", Name: "myschema"}},
					},
					VectorSearchIndexes: map[string]*resources.VectorSearchIndex{
						"idx": {CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{Name: indexName}},
					},
				},
			},
		}
	}

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"all_literal_captured", "catalog1.myschema.myindex", []string{"schemas.my_schema", "catalogs.dev_catalog"}},
		{"catalog_reference_unsupported", "${resources.catalogs.dev_catalog.name}.myschema.myindex", nil},
		{"fully_referenced_skipped", "${resources.catalogs.dev_catalog.name}.${resources.schemas.my_schema.name}.myindex", nil},
		{"more_than_three_components_skipped", "catalog1.myschema.my.index", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := newBundle(tc.input)
			d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
			require.Nil(t, d)
			assert.Equal(t, tc.expected, dependsOnNodes(b.Config.Resources.VectorSearchIndexes["idx"].Lifecycle))
			assert.Equal(t, tc.input, b.Config.Resources.VectorSearchIndexes["idx"].Name)
		})
	}
}
