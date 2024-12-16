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

func TestApplyPresetsCatalogSchema(t *testing.T) {
	b := &bundle.Bundle{
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
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"key": {
						PipelineSpec: &pipelines.PipelineSpec{
							Name:    "pipeline",
							Catalog: "<catalog>",
							Target:  "<schema>",
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
		require.Equal(t, f.Expected, val.MustString(), "preset value expected for %s based on placeholder %s", f.Path, f.Placeholder)
	}

	// Stage 4: Check completeness
	ignoredFields := map[string]string{
		// Any fields that should be ignored in the completeness check
		// Example:
		// "resources.jobs.object.schema_something": "this property doesn't relate to the catalog/schema",
	}
	checkCompleteness(t, recordedFields, ignoredFields)
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

func checkCompleteness(t *testing.T, recordedFields []recordedField, ignoredFields map[string]string) {
	t.Helper()

	// Build a set for recorded fields
	recordedSet := make(map[string]struct{})
	for _, field := range recordedFields {
		recordedSet[field.PathString] = struct{}{}
	}

	// Obtain the type of config.Resources
	var r config.Resources
	resourcesType := reflect.TypeOf(r)

	// Track missing fields
	var missingFields []string

	// Keep track of visited types to prevent infinite loops (cycles)
	visited := make(map[reflect.Type]struct{})

	// Helper function to handle maps, slices, arrays, and nested pointers/interfaces
	verifyFieldType := func(fieldType reflect.Type, path string, fn func(reflect.Type, string)) {
		switch fieldType.Kind() {
		case reflect.Slice, reflect.Array:
			// For arrays/slices, inspect the element type
			fn(fieldType.Elem(), path+"[0]")
		case reflect.Map:
			// For maps, inspect the value type
			fn(fieldType.Elem(), path+".key")
		case reflect.Ptr, reflect.Interface:
			// For pointers/interfaces, inspect the element if it's a pointer
			if fieldType.Kind() == reflect.Ptr {
				fn(fieldType.Elem(), path)
			}
		case reflect.Struct:
			// For structs, directly recurse into their fields
			fn(fieldType, path)
		default:
			// For basic or unknown kinds, do nothing
		}
	}

	// Recursive function to verify the fields of a given type.
	var verifyTypeFields func(rt reflect.Type, path string)
	verifyTypeFields = func(rt reflect.Type, path string) {
		// Avoid cycles by skipping already visited types
		if _, seen := visited[rt]; seen {
			return
		}
		visited[rt] = struct{}{}

		switch rt.Kind() {
		case reflect.Ptr, reflect.Interface:
			// For pointers/interfaces, inspect the element type if available
			if rt.Kind() == reflect.Ptr {
				verifyTypeFields(rt.Elem(), path)
			}
		case reflect.Struct:
			for i := 0; i < rt.NumField(); i++ {
				ft := rt.Field(i)
				jsonTag := ft.Tag.Get("json")
				if jsonTag == "" || jsonTag == "-" {
					// Ignore field names when there's no JSON tag,
					// e.g. for Jobs.JobSettings
					verifyFieldType(ft.Type, path, verifyTypeFields)
					continue
				}

				fieldName := strings.Split(jsonTag, ",")[0]
				fieldPath := path + "." + fieldName

				if isCatalogOrSchemaField(fieldName) {
					// Only check if the field is a string
					if ft.Type.Kind() == reflect.String {
						if _, recorded := recordedSet[fieldPath]; !recorded {
							if _, ignored := ignoredFields[fieldPath]; !ignored {
								missingFields = append(missingFields, fieldPath)
							}
						}
					}
				}

				verifyFieldType(ft.Type, fieldPath, verifyTypeFields)
			}
		default:
			// For other kinds at this level, do nothing
		}
	}

	// Start from "resources"
	verifyTypeFields(resourcesType, "resources")

	// Report all missing fields
	for _, field := range missingFields {
		t.Errorf("Field %s was not included in the test (should be covered in 'recordedFields' or 'ignoredFields')", field)
	}

	// Fail the test if there were any missing fields
	if len(missingFields) > 0 {
		t.FailNow()
	}
}

// isCatalogOrSchemaField returns true for a field names in config.Resources that we suspect could contain a catalog or schema name
func isCatalogOrSchemaField(name string) bool {
	return strings.Contains(name, "catalog") || strings.Contains(name, "schema")
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
