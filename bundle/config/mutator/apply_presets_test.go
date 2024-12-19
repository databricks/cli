package mutator_test

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dbr"
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

func TestApplyPresetsPrefix(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		job    *resources.Job
		want   string
	}{
		{
			name:   "add prefix to job",
			prefix: "prefix-",
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name: "job1",
				},
			},
			want: "prefix-job1",
		},
		{
			name:   "add empty prefix to job",
			prefix: "",
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name: "job1",
				},
			},
			want: "job1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"job1": tt.job,
						},
					},
					Presets: config.Presets{
						NamePrefix: tt.prefix,
					},
				},
			}

			ctx := context.Background()
			diag := bundle.Apply(ctx, b, mutator.ApplyPresets())

			if diag.HasError() {
				t.Fatalf("unexpected error: %v", diag)
			}

			require.Equal(t, tt.want, b.Config.Resources.Jobs["job1"].Name)
		})
	}
}

func TestApplyPresetsPrefixForUcSchema(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		schema *resources.Schema
		want   string
	}{
		{
			name:   "add prefix to schema",
			prefix: "[prefix]",
			schema: &resources.Schema{
				CreateSchema: &catalog.CreateSchema{
					Name: "schema1",
				},
			},
			want: "prefix_schema1",
		},
		{
			name:   "add empty prefix to schema",
			prefix: "",
			schema: &resources.Schema{
				CreateSchema: &catalog.CreateSchema{
					Name: "schema1",
				},
			},
			want: "schema1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Schemas: map[string]*resources.Schema{
							"schema1": tt.schema,
						},
					},
					Presets: config.Presets{
						NamePrefix: tt.prefix,
					},
				},
			}

			ctx := context.Background()
			diag := bundle.Apply(ctx, b, mutator.ApplyPresets())

			if diag.HasError() {
				t.Fatalf("unexpected error: %v", diag)
			}

			require.Equal(t, tt.want, b.Config.Resources.Schemas["schema1"].Name)
		})
	}
}

func TestApplyPresetsTags(t *testing.T) {
	tests := []struct {
		name string
		tags map[string]string
		job  *resources.Job
		want map[string]string
	}{
		{
			name: "add tags to job",
			tags: map[string]string{"env": "dev"},
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name: "job1",
					Tags: nil,
				},
			},
			want: map[string]string{"env": "dev"},
		},
		{
			name: "merge tags with existing job tags",
			tags: map[string]string{"env": "dev"},
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name: "job1",
					Tags: map[string]string{"team": "data"},
				},
			},
			want: map[string]string{"env": "dev", "team": "data"},
		},
		{
			name: "don't override existing job tags",
			tags: map[string]string{"env": "dev"},
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name: "job1",
					Tags: map[string]string{"env": "prod"},
				},
			},
			want: map[string]string{"env": "prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"job1": tt.job,
						},
					},
					Presets: config.Presets{
						Tags: tt.tags,
					},
				},
			}

			ctx := context.Background()
			diag := bundle.Apply(ctx, b, mutator.ApplyPresets())

			if diag.HasError() {
				t.Fatalf("unexpected error: %v", diag)
			}

			tags := b.Config.Resources.Jobs["job1"].Tags
			require.Equal(t, tt.want, tags)
		})
	}
}

func TestApplyPresetsJobsMaxConcurrentRuns(t *testing.T) {
	tests := []struct {
		name    string
		job     *resources.Job
		setting int
		want    int
	}{
		{
			name: "set max concurrent runs",
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name:              "job1",
					MaxConcurrentRuns: 0,
				},
			},
			setting: 5,
			want:    5,
		},
		{
			name: "do not override existing max concurrent runs",
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name:              "job1",
					MaxConcurrentRuns: 3,
				},
			},
			setting: 5,
			want:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"job1": tt.job,
						},
					},
					Presets: config.Presets{
						JobsMaxConcurrentRuns: tt.setting,
					},
				},
			}
			ctx := context.Background()
			diag := bundle.Apply(ctx, b, mutator.ApplyPresets())

			if diag.HasError() {
				t.Fatalf("unexpected error: %v", diag)
			}

			require.Equal(t, tt.want, b.Config.Resources.Jobs["job1"].MaxConcurrentRuns)
		})
	}
}

func TestApplyPresetsPrefixWithoutJobSettings(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {}, // no jobsettings inside
				},
			},
			Presets: config.Presets{
				NamePrefix: "prefix-",
			},
		},
	}

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, mutator.ApplyPresets())

	require.ErrorContains(t, diags.Error(), "job job1 is not defined")
}

func TestApplyPresetsResourceNotDefined(t *testing.T) {
	tests := []struct {
		resources config.Resources
		error     string
	}{
		{
			resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {}, // no jobsettings inside
				},
			},
			error: "job job1 is not defined",
		},
		{
			resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline1": {}, // no pipelinespec inside
				},
			},
			error: "pipeline pipeline1 is not defined",
		},
		{
			resources: config.Resources{
				Models: map[string]*resources.MlflowModel{
					"model1": {}, // no model inside
				},
			},
			error: "model model1 is not defined",
		},
		{
			resources: config.Resources{
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment1": {}, // no experiment inside
				},
			},
			error: "experiment experiment1 is not defined",
		},
		{
			resources: config.Resources{
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint1": {}, // no CreateServingEndpoint inside
				},
				RegisteredModels: map[string]*resources.RegisteredModel{
					"model1": {}, // no CreateRegisteredModelRequest inside
				},
			},
			error: "model serving endpoint endpoint1 is not defined",
		},
		{
			resources: config.Resources{
				QualityMonitors: map[string]*resources.QualityMonitor{
					"monitor1": {}, // no CreateMonitor inside
				},
			},
			error: "quality monitor monitor1 is not defined",
		},
		{
			resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"schema1": {}, // no CreateSchema inside
				},
			},
			error: "schema schema1 is not defined",
		},
		{
			resources: config.Resources{
				Clusters: map[string]*resources.Cluster{
					"cluster1": {}, // no ClusterSpec inside
				},
			},
			error: "cluster cluster1 is not defined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.error, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: tt.resources,
					Presets: config.Presets{
						TriggerPauseStatus: config.Paused,
					},
				},
			}

			ctx := context.Background()
			diags := bundle.Apply(ctx, b, mutator.ApplyPresets())

			require.ErrorContains(t, diags.Error(), tt.error)
		})
	}
}

func TestApplyPresetsSourceLinkedDeployment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this test is not applicable on Windows because source-linked mode works only in the Databricks Workspace")
	}

	testContext := context.Background()
	enabled := true
	disabled := false
	workspacePath := "/Workspace/user.name@company.com"

	tests := []struct {
		bundlePath      string
		ctx             context.Context
		name            string
		initialValue    *bool
		expectedValue   *bool
		expectedWarning string
	}{
		{
			name:          "preset enabled, bundle in Workspace, databricks runtime",
			bundlePath:    workspacePath,
			ctx:           dbr.MockRuntime(testContext, true),
			initialValue:  &enabled,
			expectedValue: &enabled,
		},
		{
			name:            "preset enabled, bundle not in Workspace, databricks runtime",
			bundlePath:      "/Users/user.name@company.com",
			ctx:             dbr.MockRuntime(testContext, true),
			initialValue:    &enabled,
			expectedValue:   &disabled,
			expectedWarning: "source-linked deployment is available only in the Databricks Workspace",
		},
		{
			name:            "preset enabled, bundle in Workspace, not databricks runtime",
			bundlePath:      workspacePath,
			ctx:             dbr.MockRuntime(testContext, false),
			initialValue:    &enabled,
			expectedValue:   &disabled,
			expectedWarning: "source-linked deployment is available only in the Databricks Workspace",
		},
		{
			name:          "preset disabled, bundle in Workspace, databricks runtime",
			bundlePath:    workspacePath,
			ctx:           dbr.MockRuntime(testContext, true),
			initialValue:  &disabled,
			expectedValue: &disabled,
		},
		{
			name:          "preset nil, bundle in Workspace, databricks runtime",
			bundlePath:    workspacePath,
			ctx:           dbr.MockRuntime(testContext, true),
			initialValue:  nil,
			expectedValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				SyncRootPath: tt.bundlePath,
				Config: config.Root{
					Presets: config.Presets{
						SourceLinkedDeployment: tt.initialValue,
					},
				},
			}

			bundletest.SetLocation(b, "presets.source_linked_deployment", []dyn.Location{{File: "databricks.yml"}})
			diags := bundle.Apply(tt.ctx, b, mutator.ApplyPresets())
			if diags.HasError() {
				t.Fatalf("unexpected error: %v", diags)
			}

			if tt.expectedWarning != "" {
				require.Equal(t, tt.expectedWarning, diags[0].Summary)
				require.NotEmpty(t, diags[0].Locations)
			}

			require.Equal(t, tt.expectedValue, b.Config.Presets.SourceLinkedDeployment)
		})
	}

}

func PresetsMock() *bundle.Bundle {
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
		},
	}
}

// Any fields that should be ignored in the completeness check
var PresetsIgnoredFields = map[string]string{
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

func TestApplyPresetsCatalogSchema(t *testing.T) {
	b := PresetsMock()
	b.Config.Presets = config.Presets{
		Catalog: "my_catalog",
		Schema:  "my_schema",
	}
	ctx := context.Background()

	// Initial scan: record all fields that contain placeholders.
	// We do this before the first apply so we can verify no changes occur.
	var recordedFields []recordedField
	require.NoError(t, b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
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
	}))

	// Convert the recordedFields to a set for easier lookup
	recordedSet := make(map[string]struct{})
	for _, field := range recordedFields {
		recordedSet[field.PathString] = struct{}{}
		if i := strings.Index(field.PathString, "["); i >= 0 {
			// For entries like resources.jobs.key.parameters[1].default, just add resources.jobs.key.parameters
			recordedSet[field.PathString[:i]] = struct{}{}
		}
	}

	// Stage 1: Apply presets before cleanup, should be no-op.
	diags := bundle.Apply(ctx, b, mutator.ApplyPresets())
	require.False(t, diags.HasError(), "unexpected error before cleanup: %v", diags.Error())

	// Verify that no recorded fields changed
	verifyNoChangesBeforeCleanup(t, b.Config.Value(), recordedFields)

	// Stage 2: Cleanup: Walk over rootVal and remove placeholders, adjusting recordedFields Expected values.
	require.NoError(t, b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		for _, f := range recordedFields {
			value, err := dyn.GetByPath(root, f.Path)
			require.NoError(t, err)

			val := value.MustString()
			cleanedVal := removePlaceholders(val)
			root, err = dyn.SetByPath(root, f.Path, dyn.V(cleanedVal))
			require.NoError(t, err)
		}
		root, err := dyn.Set(root, "resources.jobs.key.parameters", dyn.NilValue)
		require.NoError(t, err)
		return root, nil
	}))

	// Stage 3: Apply presets after cleanup.
	diags = bundle.Apply(ctx, b, mutator.ApplyPresets())
	require.False(t, diags.HasError(), "unexpected error after cleanup: %v", diags.Error())

	// Verify that fields have the expected replacements
	config := b.Config.Value()
	for _, f := range recordedFields {
		val, err := dyn.GetByPath(config, f.Path)
		require.NoError(t, err, "failed to get path %s", f.Path)
		assert.Equal(t, f.Expected, val.MustString(), "preset value expected for %s based on placeholder %s", f.Path, f.Placeholder)
	}

	// Stage 4: Check completeness
	expectedFields := findCatalogSchemaFields()
	assert.GreaterOrEqual(t, len(expectedFields), 42, "expected at least 42 catalog/schema fields, but got %d", len(expectedFields))
	for _, field := range expectedFields {
		if _, recorded := recordedSet[field]; !recorded {
			if _, ignored := PresetsIgnoredFields[field]; !ignored {
				t.Errorf("Field %s was not included in the catalog/schema presets test. If this is a new field, please add it to PresetsMock or PresetsIgnoredFields and add support for it as appropriate.", field)
			}
		}
	}
}

func verifyNoChangesBeforeCleanup(t *testing.T, rootVal dyn.Value, recordedFields []recordedField) {
	t.Helper()

	for _, f := range recordedFields {
		val, err := dyn.GetByPath(rootVal, f.Path)
		require.NoError(t, err, "failed to get path %s", f.Path)
		require.Equal(t, f.Placeholder, val.MustString(),
			"expected placeholder '%s' at %s to remain unchanged before cleanup", f.Placeholder, f.Path)
	}
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
