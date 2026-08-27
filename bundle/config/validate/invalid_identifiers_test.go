package validate_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredRejectsEmptyAndControlCharIdentifiers(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Models: map[string]*resources.MlflowModel{
					"empty": {CreateModelRequest: ml.CreateModelRequest{Name: ""}},
				},
				Catalogs: map[string]*resources.Catalog{
					"blank": {CreateCatalog: catalog.CreateCatalog{Name: "   "}},
				},
				Volumes: map[string]*resources.Volume{
					"ctrl": {
						CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
							Name:        "tab\there",
							CatalogName: "main",
							SchemaName:  "default",
							VolumeType:  catalog.VolumeTypeManaged,
						},
					},
					"empty_parent": {
						CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
							Name:        "v",
							CatalogName: "",
							SchemaName:  "default",
							VolumeType:  catalog.VolumeTypeManaged,
						},
					},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"nl": {
						CreateServingEndpoint: serving.CreateServingEndpoint{
							Name: "line1\nline2",
						},
					},
				},
				VectorSearchEndpoints: map[string]*resources.VectorSearchEndpoint{
					"vse": {
						CreateEndpoint: vectorsearch.CreateEndpoint{
							Name:         "bad\tname",
							EndpointType: vectorsearch.EndpointTypeStandard,
						},
					},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"e": {CreateExperiment: ml.CreateExperiment{Name: ""}},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, validate.Required())
	require.True(t, diags.HasError())

	assert.ElementsMatch(t, []string{
		"catalog name must not be blank",
		"model name is required",
		"volume name must not contain control characters",
		"model_serving_endpoint name must not contain control characters",
		"vector_search_endpoint name must not contain control characters",
		"experiment name is required",
		// empty catalog_name is omitted by FromTyped(omitempty) in tests; warning only.
		"required field \"catalog_name\" is not set",
	}, diagSummaries(diags))
}

func TestRequiredRejectsExplicitEmptyUCParentInDyn(t *testing.T) {
	b := &bundle.Bundle{}
	require.NoError(t, b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.V(map[string]dyn.Value{
			"resources": dyn.V(map[string]dyn.Value{
				"volumes": dyn.V(map[string]dyn.Value{
					"v": dyn.V(map[string]dyn.Value{
						"name":         dyn.V("v"),
						"catalog_name": dyn.V(""),
						"schema_name":  dyn.V("default"),
						"volume_type":  dyn.V("MANAGED"),
					}),
				}),
			}),
		}), nil
	}))

	diags := bundle.Apply(t.Context(), b, validate.Required())
	require.True(t, diags.HasError())
	assert.Contains(t, diagSummaries(diags), "volume catalog_name is required")
}

func TestRequiredIdentifierValidationScope(t *testing.T) {
	b := &bundle.Bundle{}
	require.NoError(t, b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.V(map[string]dyn.Value{
			"bundle": dyn.V(map[string]dyn.Value{
				"name": dyn.V(" \t"),
			}),
			"resources": dyn.V(map[string]dyn.Value{
				"dashboards": dyn.V(map[string]dyn.Value{
					"dashboard": dyn.V(map[string]dyn.Value{
						"display_name": dyn.V("bad\nname"),
						"warehouse_id": dyn.V("warehouse"),
					}),
				}),
				"jobs": dyn.V(map[string]dyn.Value{
					"job": dyn.V(map[string]dyn.Value{
						"parameters": dyn.V([]dyn.Value{
							dyn.V(map[string]dyn.Value{"default": dyn.V("value")}),
						}),
					}),
				}),
				"sql_warehouses": dyn.V(map[string]dyn.Value{
					"warehouse": dyn.V(map[string]dyn.Value{
						"name": dyn.V("   "),
					}),
				}),
			}),
		}), nil
	}))

	diags := bundle.Apply(t.Context(), b, validate.Required())
	assert.ElementsMatch(t, []string{
		"bundle name must not contain control characters",
		"dashboard display_name must not contain control characters",
		"required field \"name\" is not set",
		"sql_warehouse name must not be blank",
	}, diagSummaries(diags))
}

func TestRequiredPreservesLocationForMetacharacterResourceKey(t *testing.T) {
	location := dyn.Location{File: "databricks.yml", Line: 6, Column: 13}
	b := &bundle.Bundle{}
	require.NoError(t, b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.V(map[string]dyn.Value{
			"resources": dyn.V(map[string]dyn.Value{
				"volumes": dyn.V(map[string]dyn.Value{
					"weird[0]key": dyn.V(map[string]dyn.Value{
						"name":         dyn.NewValue("", []dyn.Location{location}),
						"catalog_name": dyn.V("main"),
						"schema_name":  dyn.V("default"),
						"volume_type":  dyn.V("MANAGED"),
					}),
				}),
			}),
		}), nil
	}))

	diags := bundle.Apply(t.Context(), b, validate.Required())
	require.True(t, diags.HasError())
	require.Len(t, diags, 1)
	assert.Equal(t, "volume name is required", diags[0].Summary)
	assert.Equal(t, []dyn.Location{location}, diags[0].Locations)
}

func TestRequiredAcceptsValidIdentifiers(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Models: map[string]*resources.MlflowModel{
					"model": {CreateModelRequest: ml.CreateModelRequest{Name: "model"}},
				},
			},
		},
	}

	assert.Empty(t, bundle.Apply(t.Context(), b, validate.Required()))
}

func TestRequiredAcceptsMissingOptionalUCParents(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				RegisteredModels: map[string]*resources.RegisteredModel{
					"model": {
						CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{
							Name: "model",
						},
					},
				},
			},
		},
	}

	assert.Empty(t, bundle.Apply(t.Context(), b, validate.Required()))
}

func TestRequiredRejectsBlankOptionalUCParent(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				RegisteredModels: map[string]*resources.RegisteredModel{
					"model": {
						CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{
							Name:        "model",
							CatalogName: "   ",
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, validate.Required())
	require.True(t, diags.HasError())
	assert.Contains(t, diagSummaries(diags), "registered_model catalog_name must not be blank")
}

func TestRequiredRejectsMissingResourceIdentifiers(t *testing.T) {
	tests := []struct {
		resourceType string
		summary      string
	}{
		{"alerts", "alert display_name is required"},
		{"apps", "app name is required"},
		{"catalogs", "catalog name is required"},
		{"dashboards", "dashboard display_name is required"},
		{"database_catalogs", "database_catalog name is required"},
		{"database_instances", "database_instance name is required"},
		{"experiments", "experiment name is required"},
		{"external_locations", "external_location name is required"},
		{"instance_pools", "instance_pool instance_pool_name is required"},
		{"model_serving_endpoints", "model_serving_endpoint name is required"},
		{"models", "model name is required"},
		{"registered_models", "registered_model name is required"},
		{"schemas", "schema name is required"},
		{"secret_scopes", "secret_scope name is required"},
		{"secrets", "secret name is required"},
		{"sql_warehouses", "sql_warehouse name is required"},
		{"synced_database_tables", "synced_database_table name is required"},
		{"vector_search_endpoints", "vector_search_endpoint name is required"},
		{"vector_search_indexes", "vector_search_index name is required"},
		{"volumes", "volume name is required"},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			b := &bundle.Bundle{}
			require.NoError(t, b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
				return dyn.V(map[string]dyn.Value{
					"bundle": dyn.V(map[string]dyn.Value{
						"name": dyn.V("bundle"),
					}),
					"resources": dyn.V(map[string]dyn.Value{
						tt.resourceType: dyn.V(map[string]dyn.Value{
							"weird[0]key": dyn.V(map[string]dyn.Value{}),
						}),
					}),
				}), nil
			}))

			diags := bundle.Apply(t.Context(), b, validate.Required())
			assert.Contains(t, diagSummaries(diags), tt.summary)
		})
	}
}

func diagSummaries(diags diag.Diagnostics) []string {
	out := make([]string, 0, len(diags))
	for _, d := range diags {
		out = append(out, d.Summary)
	}
	return out
}
