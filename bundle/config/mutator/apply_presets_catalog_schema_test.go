package mutator_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedField struct {
	Path        dyn.Path
	PathString  string
	Placeholder string
	Expected    string
}

func mockPresetsCatalogSchema() *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"key": {
						JobSettings: &jobs.JobSettings{
							Name: "job",
							Parameters: []jobs.JobParameterDefinition{
								{Name: "catalog", Default: "<catalog>"},
								{Name: "schema", Default: "<schema>"},
							},
							Tasks: []jobs.Task{
								{
									DbtTask: &jobs.DbtTask{
										Catalog: "<catalog>",
										Schema:  "<schema>",
									},
								},
								{
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "/file",
									},
								},
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "/notebook",
									},
								},
							},
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"key": {
						PipelineSpec: &pipelines.PipelineSpec{
							Name:    "pipeline",
							Catalog: "<catalog>",
							Target:  "<schema>",
							GatewayDefinition: &pipelines.IngestionGatewayPipelineDefinition{
								GatewayStorageCatalog: "<catalog>",
								GatewayStorageSchema:  "<schema>",
							},
							IngestionDefinition: &pipelines.IngestionPipelineDefinition{
								Objects: []pipelines.IngestionConfig{
									{
										Report: &pipelines.ReportSpec{
											DestinationCatalog: "<catalog>",
											DestinationSchema:  "<schema>",
										},
										Schema: &pipelines.SchemaSpec{
											SourceCatalog:      "<catalog>",
											SourceSchema:       "<schema>",
											DestinationCatalog: "<catalog>",
											DestinationSchema:  "<schema>",
										},
										Table: &pipelines.TableSpec{
											SourceCatalog:      "<catalog>",
											SourceSchema:       "<schema>",
											DestinationCatalog: "<catalog>",
											DestinationSchema:  "<schema>",
										},
									},
								},
							},
						},
					},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"key": {
						CreateServingEndpoint: &serving.CreateServingEndpoint{
							Name: "serving",
							AiGateway: &serving.AiGatewayConfig{
								InferenceTableConfig: &serving.AiGatewayInferenceTableConfig{
									CatalogName: "<catalog>",
									SchemaName:  "<schema>",
								},
							},
							Config: serving.EndpointCoreConfigInput{
								AutoCaptureConfig: &serving.AutoCaptureConfigInput{
									CatalogName: "<catalog>",
									SchemaName:  "<schema>",
								},
								ServedEntities: []serving.ServedEntityInput{
									{EntityName: "<catalog>.<schema>.entity"},
								},
								ServedModels: []serving.ServedModelInput{
									{ModelName: "<catalog>.<schema>.model"},
								},
							},
						},
					},
				},
				RegisteredModels: map[string]*resources.RegisteredModel{
					"key": {
						CreateRegisteredModelRequest: &catalog.CreateRegisteredModelRequest{
							Name:        "registered_model",
							CatalogName: "<catalog>",
							SchemaName:  "<schema>",
						},
					},
				},
				QualityMonitors: map[string]*resources.QualityMonitor{
					"key": {
						TableName: "<catalog>.<schema>.table",
						CreateMonitor: &catalog.CreateMonitor{
							OutputSchemaName: "<catalog>.<schema>",
						},
					},
				},
				Schemas: map[string]*resources.Schema{
					"key": {
						CreateSchema: &catalog.CreateSchema{
							Name:        "<schema>",
							CatalogName: "<catalog>",
						},
					},
				},
			},
			Presets: config.Presets{
				Catalog: "my_catalog",
				Schema:  "my_schema",
			},
		},
	}
}

// ignoredFields are fields that should be ignored in the completeness check
var ignoredFields = map[string]string{
	"resources.pipelines.key.schema":                                 "schema is still in private preview",
	"resources.jobs.key.tasks[0].notebook_task.base_parameters":      "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].python_wheel_task.named_parameters": "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].python_wheel_task.parameters":       "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].run_job_task.job_parameters":        "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].spark_jar_task.parameters":          "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].spark_python_task.parameters":       "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].spark_submit_task.parameters":       "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].sql_task.parameters":                "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].run_job_task.jar_params":            "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].run_job_task.notebook_params":       "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].run_job_task.pipeline_params":       "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].run_job_task.python_named_params":   "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].run_job_task.python_params":         "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].run_job_task.spark_submit_params":   "catalog/schema are passed via job parameters",
	"resources.jobs.key.tasks[0].run_job_task.sql_params":            "catalog/schema are passed via job parameters",
	"resources.pipelines.key.ingestion_definition.objects[0].schema": "schema name is under schema.source_schema/destination_schema",
	"resources.schemas": "schema name of schemas is under resources.schemas.key.Name",
}

func TestApplyPresetsCatalogSchemaWhenAlreadySet(t *testing.T) {
	b := mockPresetsCatalogSchema()
	recordedFields := recordPlaceholderFields(t, b)

	diags := bundle.Apply(context.Background(), b, mutator.ApplyPresets())
	require.NoError(t, diags.Error())

	for _, f := range recordedFields {
		val, err := dyn.GetByPath(b.Config.Value(), f.Path)
		require.NoError(t, err, "failed to get path %s", f.Path)
		require.Equal(t, f.Placeholder, val.MustString(),
			"expected placeholder '%s' at %s to remain unchanged before cleanup", f.Placeholder, f.Path)
	}
}

func TestApplyPresetsCatalogSchemaWhenNotSet(t *testing.T) {
	b := mockPresetsCatalogSchema()
	recordedFields := recordPlaceholderFields(t, b)

	// Set all catalog/schema fields to empty strings / nil
	require.NoError(t, b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		for _, f := range recordedFields {
			value, err := dyn.GetByPath(root, f.Path)
			require.NoError(t, err)

			val := value.MustString()
			cleanedVal := removePlaceholders(val)
			root, err = dyn.SetByPath(root, f.Path, dyn.V(cleanedVal))
			require.NoError(t, err)
		}
		return dyn.Set(root, "resources.jobs.key.parameters", dyn.NilValue)
	}))

	// Apply catalog/schema presets
	diags := bundle.Apply(context.Background(), b, mutator.ApplyPresetsCatalogSchema())
	require.NoError(t, diags.Error())

	// Verify that all catalog/schema fields have been set to the presets
	for _, f := range recordedFields {
		val, err := dyn.GetByPath(b.Config.Value(), f.Path)
		require.NoError(t, err, "could not find expected field(s) at %s", f.Path)
		assert.Equal(t, f.Expected, val.MustString(), "preset value expected for %s based on placeholder %s", f.Path, f.Placeholder)
	}
}

func TestApplyPresetsCatalogSchemaCompleteness(t *testing.T) {
	b := mockPresetsCatalogSchema()
	recordedFields := recordPlaceholderFields(t, b)

	// Convert the recordedFields to a set for easier lookup
	recordedPaths := make(map[string]struct{})
	for _, field := range recordedFields {
		recordedPaths[field.PathString] = struct{}{}
		if i := strings.Index(field.PathString, "["); i >= 0 {
			// For entries like resources.jobs.key.parameters[1].default, just add resources.jobs.key.parameters
			recordedPaths[field.PathString[:i]] = struct{}{}
		}
	}

	// Find all catalog/schema fields that we think should be covered based
	// on all properties in config.Resources.
	expectedFields := findCatalogSchemaFields()
	assert.GreaterOrEqual(t, len(expectedFields), 42, "expected at least 42 catalog/schema fields, but got %d", len(expectedFields))

	// Verify that all expected fields are there
	for _, field := range expectedFields {
		if _, recorded := recordedPaths[field]; !recorded {
			if _, ignored := ignoredFields[field]; !ignored {
				t.Errorf("Field %s was not included in the catalog/schema presets test. If this is a new field, please add it to PresetsMock or PresetsIgnoredFields and add support for it as appropriate.", field)
			}
		}
	}
}

// recordPlaceholderFields scans the config and records all fields containing catalog/schema placeholders
func recordPlaceholderFields(t *testing.T, b *bundle.Bundle) []recordedField {
	t.Helper()

	var recordedFields []recordedField
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		_, err := dyn.Walk(b.Config.Value(), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if v.Kind() == dyn.KindString {
				val := v.MustString()
				if strings.Contains(val, "<catalog>") || strings.Contains(val, "<schema>") {
					pathCopy := make(dyn.Path, len(p))
					copy(pathCopy, p)
					recordedFields = append(recordedFields, recordedField{
						Path:        pathCopy,
						PathString:  pathCopy.String(),
						Placeholder: val,
						Expected:    replacePlaceholders(val, "my_catalog", "my_schema"),
					})
				}
			}
			return v, nil
		})
		return root, err
	})
	require.NoError(t, err)
	return recordedFields
}

// findCatalogSchemaFields finds all fields in config.Resources that might refer
// to a catalog or schema. Returns a slice of field paths.
func findCatalogSchemaFields() []string {
	visited := make(map[reflect.Type]struct{})
	var results []string

	// verifyTypeFields is a recursive function to verify the fields of a given type
	var walkTypeFields func(rt reflect.Type, path string)
	walkTypeFields = func(rt reflect.Type, path string) {
		if _, seen := visited[rt]; seen {
			return
		}
		visited[rt] = struct{}{}

		switch rt.Kind() {
		case reflect.Slice, reflect.Array:
			walkTypeFields(rt.Elem(), path+"[0]")
		case reflect.Map:
			walkTypeFields(rt.Elem(), path+".key")
		case reflect.Ptr:
			walkTypeFields(rt.Elem(), path)
		case reflect.Struct:
			for i := 0; i < rt.NumField(); i++ {
				ft := rt.Field(i)
				jsonTag := ft.Tag.Get("json")
				if jsonTag == "" || jsonTag == "-" {
					// Ignore field names when there's no JSON tag, e.g. for Jobs.JobSettings
					walkTypeFields(ft.Type, path)
					continue
				}

				fieldName := strings.Split(jsonTag, ",")[0]
				fieldPath := path + "." + fieldName

				if isCatalogOrSchemaField(fieldName) {
					results = append(results, fieldPath)
				}

				walkTypeFields(ft.Type, fieldPath)
			}
		}
	}

	var r config.Resources
	walkTypeFields(reflect.TypeOf(r), "resources")
	return results
}

// isCatalogOrSchemaField returns true for a field names in config.Resources that we suspect could contain a catalog or schema name
func isCatalogOrSchemaField(name string) bool {
	return strings.Contains(name, "catalog") ||
		strings.Contains(name, "schema") ||
		strings.Contains(name, "parameters") ||
		strings.Contains(name, "params")
}

func removePlaceholders(value string) string {
	value = strings.ReplaceAll(value, "<catalog>.", "")
	value = strings.ReplaceAll(value, "<schema>.", "")
	value = strings.ReplaceAll(value, "<catalog>", "")
	value = strings.ReplaceAll(value, "<schema>", "")
	return value
}

func replacePlaceholders(placeholder, catalog, schema string) string {
	expected := strings.ReplaceAll(placeholder, "<catalog>", catalog)
	expected = strings.ReplaceAll(expected, "<schema>", schema)
	return expected
}
