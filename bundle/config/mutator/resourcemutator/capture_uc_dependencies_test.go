package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
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
		{"schema_reference_passthrough", "catalog1", "${resources.schemas.schema1.name}", "${resources.schemas.schema1.name}"},
		{"catalog_reference_unresolved", "${resources.catalogs.x.name}", "foobar", "foobar"},
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
		{"reference_passthrough", "${resources.catalogs.dev_catalog.name}", "${resources.catalogs.dev_catalog.name}"},
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
			},
		},
	}

	d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
	require.Nil(t, d)

	schemaRef := "${resources.schemas.my_schema.name}"
	catalogRef := "${resources.catalogs.my_catalog.name}"

	// Schema catalog dependency.
	assert.Equal(t, catalogRef, b.Config.Resources.Schemas["my_schema"].CatalogName)

	// Volume.
	assert.Equal(t, schemaRef, b.Config.Resources.Volumes["my_volume"].SchemaName)
	assert.Equal(t, catalogRef, b.Config.Resources.Volumes["my_volume"].CatalogName)

	// Registered model.
	assert.Equal(t, schemaRef, b.Config.Resources.RegisteredModels["my_model"].SchemaName)
	assert.Equal(t, catalogRef, b.Config.Resources.RegisteredModels["my_model"].CatalogName)

	// Pipeline.
	assert.Equal(t, schemaRef, b.Config.Resources.Pipelines["my_pipeline"].Schema)
	assert.Equal(t, catalogRef, b.Config.Resources.Pipelines["my_pipeline"].Catalog)

	// Quality monitor (compound "catalog.schema" field).
	assert.Equal(t, catalogRef+"."+schemaRef, b.Config.Resources.QualityMonitors["my_monitor"].OutputSchemaName)

	// Model serving endpoint.
	itc := b.Config.Resources.ModelServingEndpoints["my_endpoint"].AiGateway.InferenceTableConfig
	assert.Equal(t, schemaRef, itc.SchemaName)
	assert.Equal(t, catalogRef, itc.CatalogName)

	// Model service (compound "schemas/{catalog}.{schema}" parent field).
	assert.Equal(t, "schemas/"+catalogRef+"."+schemaRef, b.Config.Resources.ModelServices["my_model_service"].Parent)

	// Vector search index (three-part "catalog.schema.index" name).
	assert.Equal(t, catalogRef+"."+schemaRef+".myindex", b.Config.Resources.VectorSearchIndexes["my_index"].Name)
}

// Pipeline schema and target are mutually exclusive; only the populated field
// should be resolved.
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

	ref := "${resources.schemas.s.name}"

	assert.Equal(t, ref, b.Config.Resources.Pipelines["with_schema"].Schema)
	assert.Empty(t, b.Config.Resources.Pipelines["with_schema"].Target)

	assert.Equal(t, ref, b.Config.Resources.Pipelines["with_target"].Target)
	assert.Empty(t, b.Config.Resources.Pipelines["with_target"].Schema)
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

	schemaRef := "${resources.schemas.my_schema.name}"
	catalogRef := "${resources.catalogs.my_catalog.name}"

	acc := b.Config.Resources.ModelServingEndpoints["auto_capture"].Config.AutoCaptureConfig
	assert.Equal(t, schemaRef, acc.SchemaName)
	assert.Equal(t, catalogRef, acc.CatalogName)

	itc := b.Config.Resources.ModelServingEndpoints["no_match"].AiGateway.InferenceTableConfig
	assert.Equal(t, "other", itc.CatalogName)
	assert.Equal(t, "other", itc.SchemaName)

	assert.Nil(t, b.Config.Resources.ModelServingEndpoints["nil_endpoint"])
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

	assert.Equal(t, "${resources.catalogs.my_catalog.name}.other.myindex", b.Config.Resources.VectorSearchIndexes["catalog_only"].Name)
	assert.Equal(t, "other.other.myindex", b.Config.Resources.VectorSearchIndexes["no_match"].Name)
	assert.Empty(t, b.Config.Resources.VectorSearchIndexes["empty"].Name)
	assert.Equal(t, "mycatalog.myschema", b.Config.Resources.VectorSearchIndexes["two_part"].Name)
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
	}{
		{"three_part", "catalog.schema.index", 3, []string{"catalog", "schema", "index"}},
		{"two_part", "catalog.schema", 2, []string{"catalog", "schema"}},
		{"extra_dots_kept_in_last", "a.b.c.d", 3, []string{"a", "b", "c.d"}},
		{"fewer_parts_than_n", "a.b", 3, []string{"a", "b"}},
		{"single_component", "a", 3, []string{"a"}},
		{"empty", "", 3, []string{""}},
		{"reference_catalog_atomic", "${resources.catalogs.c.name}.schema.index", 3, []string{"${resources.catalogs.c.name}", "schema", "index"}},
		{"reference_catalog_and_schema_atomic", "${resources.catalogs.c.name}.${resources.schemas.s.name}.index", 3, []string{"${resources.catalogs.c.name}", "${resources.schemas.s.name}", "index"}},
		{"reference_in_two_part", "${resources.catalogs.c.name}.schema", 2, []string{"${resources.catalogs.c.name}", "schema"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, splitUCName(tc.input, tc.n))
		})
	}
}

func TestLiteralCatalogName(t *testing.T) {
	b := bundleWithCatalogs()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"literal", "catalog1", "catalog1"},
		{"reference_resolves", "${resources.catalogs.dev_catalog.name}", "catalog1"},
		{"reference_missing_catalog", "${resources.catalogs.unknown.name}", ""},
		{"reference_wrong_resource_type", "${resources.schemas.dev_catalog.name}", ""},
		{"impure_reference", "prefix-${resources.catalogs.dev_catalog.name}", ""},
		{"empty", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, literalCatalogName(b, tc.input))
		})
	}
}

// A UC name that already contains ${resources...} references is left intact, while
// any literal catalog/schema component is still captured. This covers the mixed case
// where the catalog is a reference but the sibling schema is a literal, in both
// directions (a bundle schema may itself carry its catalog as a literal or reference).
func TestCaptureUCDependenciesVectorSearchIndexReferences(t *testing.T) {
	newBundle := func(indexName string) *bundle.Bundle {
		return &bundle.Bundle{
			Config: config.Root{
				Resources: config.Resources{
					Catalogs: map[string]*resources.Catalog{
						"dev_catalog": {CreateCatalog: catalog.CreateCatalog{Name: "catalog1"}},
					},
					Schemas: map[string]*resources.Schema{
						"schema_lit": {CreateSchema: catalog.CreateSchema{CatalogName: "catalog1", Name: "foobar"}},
						"schema_ref": {CreateSchema: catalog.CreateSchema{CatalogName: "${resources.catalogs.dev_catalog.name}", Name: "barbaz"}},
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
		expected string
	}{
		{
			"catalog_reference_literal_schema",
			"${resources.catalogs.dev_catalog.name}.foobar.my_index",
			"${resources.catalogs.dev_catalog.name}.${resources.schemas.schema_lit.name}.my_index",
		},
		{
			"catalog_reference_schema_whose_catalog_is_a_reference",
			"${resources.catalogs.dev_catalog.name}.barbaz.my_index",
			"${resources.catalogs.dev_catalog.name}.${resources.schemas.schema_ref.name}.my_index",
		},
		{
			"fully_referenced_passthrough",
			"${resources.catalogs.dev_catalog.name}.${resources.schemas.schema_lit.name}.my_index",
			"${resources.catalogs.dev_catalog.name}.${resources.schemas.schema_lit.name}.my_index",
		},
		{
			"all_literal",
			"catalog1.foobar.my_index",
			"${resources.catalogs.dev_catalog.name}.${resources.schemas.schema_lit.name}.my_index",
		},
		{
			"literal_catalog_schema_whose_catalog_is_a_reference",
			"catalog1.barbaz.my_index",
			"${resources.catalogs.dev_catalog.name}.${resources.schemas.schema_ref.name}.my_index",
		},
		{
			"reference_to_unknown_catalog_unchanged",
			"${resources.catalogs.unknown.name}.foobar.my_index",
			"${resources.catalogs.unknown.name}.foobar.my_index",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := newBundle(tc.input)
			d := bundle.Apply(t.Context(), b, CaptureUCDependencies())
			require.Nil(t, d)
			assert.Equal(t, tc.expected, b.Config.Resources.VectorSearchIndexes["idx"].Name)
		})
	}
}
