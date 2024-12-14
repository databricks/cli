package mutator_test

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
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
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/require"
)

type RecordedField struct {
	Path  string
	Value string
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
					"object": {
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
					"object": {
						PipelineSpec: &pipelines.PipelineSpec{
							Name:    "pipeline",
							Catalog: "<catalog>",
							Target:  "<schema>",
						},
					},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"object": {
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
					"object": {
						CreateRegisteredModelRequest: &catalog.CreateRegisteredModelRequest{
							Name:        "registered_model",
							CatalogName: "<catalog>",
							SchemaName:  "<schema>",
						},
					},
				},
				QualityMonitors: map[string]*resources.QualityMonitor{
					"object": {
						TableName: "table",
						CreateMonitor: &catalog.CreateMonitor{
							OutputSchemaName: "<catalog>.<schema>",
						},
					},
				},
				Schemas: map[string]*resources.Schema{
					"object": {
						CreateSchema: &catalog.CreateSchema{
							Name:        "<schema>",
							CatalogName: "<catalog>",
						},
					},
				},
				Models: map[string]*resources.MlflowModel{
					"object": {
						Model: &ml.Model{
							Name: "<catalog>.<schema>.model",
						},
					},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"object": {
						Experiment: &ml.Experiment{
							Name: "<catalog>.<schema>.experiment",
						},
					},
				},
				Clusters: map[string]*resources.Cluster{
					"object": {
						ClusterSpec: &compute.ClusterSpec{
							ClusterName: "cluster",
						},
					},
				},
				Dashboards: map[string]*resources.Dashboard{
					"object": {
						Dashboard: &dashboards.Dashboard{
							DisplayName: "dashboard",
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

	// Stage 1: Apply presets BEFORE cleanup.
	// Because all fields are already set to placeholders, Apply should NOT overwrite them (no-op).
	ctx := context.Background()
	diags := bundle.Apply(ctx, b, mutator.ApplyPresets())
	require.False(t, diags.HasError(), "unexpected error before cleanup: %v", diags.Error())
	verifyNoChangesBeforeCleanup(t, b.Config)

	// Stage 2: Cleanup all "<catalog>" and "<schema>" placeholders
	// and record where they were.
	b.Config.MarkMutatorEntry(ctx)
	resources := reflect.ValueOf(&b.Config.Resources).Elem()
	recordedFields := recordAndCleanupFields(resources, "Resources")
	b.Config.Resources.Jobs["object"].Parameters = nil
	b.Config.MarkMutatorExit(ctx)

	// Stage 3: Apply presets after cleanup.
	diags = bundle.Apply(ctx, b, mutator.ApplyPresets())
	require.False(t, diags.HasError(), "unexpected error after cleanup: %v", diags.Error())
	verifyAllFields(t, b.Config, recordedFields)

	// Stage 4: Verify that all known fields in config.Resources have been processed.
	checkCompleteness(t, &b.Config.Resources, "Resources", recordedFields)
}

func verifyNoChangesBeforeCleanup(t *testing.T, cfg config.Root) {
	t.Helper()

	// Just check a few representative fields to ensure they are still placeholders.
	// For example: Job parameter defaults should still have "<catalog>" and "<schema>"
	jobParams := cfg.Resources.Jobs["object"].Parameters
	require.Len(t, jobParams, 2, "job parameters count mismatch")
	require.Equal(t, "<catalog>", jobParams[0].Default, "expected no changes before cleanup")
	require.Equal(t, "<schema>", jobParams[1].Default, "expected no changes before cleanup")

	pipeline := cfg.Resources.Pipelines["object"]
	require.Equal(t, "<catalog>", pipeline.Catalog, "expected no changes before cleanup")
	require.Equal(t, "<schema>", pipeline.Target, "expected no changes before cleanup")
}

// recordAndCleanupFields recursively finds all Catalog/CatalogName/Schema/SchemaName fields,
// records their original values, and replaces them with empty strings.
func recordAndCleanupFields(rv reflect.Value, path string) []RecordedField {
	var recordedFields []RecordedField

	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface:
		if !rv.IsNil() {
			recordedFields = append(recordedFields, recordAndCleanupFields(rv.Elem(), path)...)
		}

	case reflect.Struct:
		tp := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			ft := tp.Field(i)
			fv := rv.Field(i)
			fPath := path + "." + ft.Name

			if fv.Kind() == reflect.String {
				original := fv.String()
				newVal := cleanedValue(original)
				if newVal != original {
					fv.SetString(newVal)
					recordedFields = append(recordedFields, RecordedField{fPath, original})
				}
			}

			recordedFields = append(recordedFields, recordAndCleanupFieldsRecursive(fv, fPath)...)
		}

	case reflect.Map:
		for _, mk := range rv.MapKeys() {
			mVal := rv.MapIndex(mk)
			recordedFields = append(recordedFields, recordAndCleanupFieldsRecursive(mVal, path+"."+mk.String())...)
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			recordedFields = append(recordedFields, recordAndCleanupFieldsRecursive(rv.Index(i), fmt.Sprintf("%s[%d]", path, i))...)
		}
	}

	return recordedFields
}

// verifyAllFields checks if all collected fields are now properly replaced after ApplyPresets.
func verifyAllFields(t *testing.T, cfg config.Root, recordedFields []RecordedField) {
	t.Helper()
	for _, f := range recordedFields {
		expected := replaceCatalogSchemaPlaceholders(f.Value)
		got := getStringValueAtPath(t, reflect.ValueOf(cfg), f.Path)
		require.Equal(t, expected, got, "expected catalog/schema to be replaced by preset values at %s", f.Path)
	}
}

// checkCompleteness ensures that all catalog/schema fields have been processed.
func checkCompleteness(t *testing.T, root interface{}, rootPath string, recordedFields []RecordedField) {
	t.Helper()
	recordedSet := make(map[string]bool)
	for _, f := range recordedFields {
		recordedSet[f.Path] = true
	}

	var check func(rv reflect.Value, path string)
	check = func(rv reflect.Value, path string) {
		switch rv.Kind() {
		case reflect.Ptr, reflect.Interface:
			if !rv.IsNil() {
				check(rv.Elem(), path)
			}
		case reflect.Struct:
			tp := rv.Type()
			for i := 0; i < rv.NumField(); i++ {
				ft := tp.Field(i)
				fv := rv.Field(i)
				fPath := path + "." + ft.Name
				if isCatalogOrSchemaField(ft.Name) {
					require.Truef(t, recordedSet[fPath],
						"Field %s was not recorded in recordedFields (completeness check failed)", fPath)
				}
				check(fv, fPath)
			}
		case reflect.Map:
			for _, mk := range rv.MapKeys() {
				mVal := rv.MapIndex(mk)
				check(mVal, path+"."+mk.String())
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < rv.Len(); i++ {
				check(rv.Index(i), fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}

	rv := reflect.ValueOf(root)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	check(rv, rootPath)
}

// getStringValueAtPath navigates the given path and returns the string value at that path.
func getStringValueAtPath(t *testing.T, root reflect.Value, path string) string {
	t.Helper()
	parts := strings.Split(path, ".")
	return navigatePath(t, root, parts)
}

func navigatePath(t *testing.T, rv reflect.Value, parts []string) string {
	t.Helper()

	// Trim empty parts if any
	for len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}

	for len(parts) > 0 {
		part := parts[0]
		parts = parts[1:]

		// Dereference pointers/interfaces before proceeding
		for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
			require.Falsef(t, rv.IsNil(), "nil pointer or interface encountered at part '%s'", part)
			rv = rv.Elem()
		}

		// If the part has indexing like "Parameters[0]", split it into "Parameters" and "[0]"
		var indexPart string
		fieldName := part
		if idx := strings.IndexRune(part, '['); idx != -1 {
			// e.g. part = "Parameters[0]"
			fieldName = part[:idx] // "Parameters"
			indexPart = part[idx:] // "[0]"
			require.Truef(t, strings.HasPrefix(indexPart, "["), "expected '[' in indexing")
			require.Truef(t, strings.HasSuffix(indexPart, "]"), "expected ']' at end of indexing")
		}

		// Navigate down structures/maps
		switch rv.Kind() {
		case reflect.Struct:
			// Find the struct field by name
			ft, ok := rv.Type().FieldByName(fieldName)
			if !ok {
				t.Fatalf("Could not find field '%s' in struct at path", fieldName)
			}
			rv = rv.FieldByIndex(ft.Index)

		case reflect.Map:
			// Use fieldName as map key
			mapVal := rv.MapIndex(reflect.ValueOf(fieldName))
			require.Truef(t, mapVal.IsValid(), "no map entry '%s' found in path", fieldName)
			rv = mapVal

		default:
			// If we're here, maybe we expected a struct or map but got something else
			t.Fatalf("Unexpected kind '%s' when looking for '%s'", rv.Kind(), fieldName)
		}

		// If there's an index part, apply it now
		if indexPart != "" {
			// Dereference again if needed
			for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
				require.False(t, rv.IsNil(), "nil pointer or interface when indexing")
				rv = rv.Elem()
			}

			require.Truef(t, rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array, "expected slice/array for indexing but got %s", rv.Kind())

			idxStr := indexPart[1 : len(indexPart)-1] // remove [ and ]
			idx, err := strconv.Atoi(idxStr)
			require.NoError(t, err, "invalid slice index %s", indexPart)

			require.Truef(t, idx < rv.Len(), "index %d out of range in slice/array of length %d", idx, rv.Len())
			rv = rv.Index(idx)
		}
	}

	// Dereference if needed at the leaf
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		require.False(t, rv.IsNil(), "nil pointer or interface at leaf")
		rv = rv.Elem()
	}

	require.Equal(t, reflect.String, rv.Kind(), "expected a string at the final path")
	return rv.String()
}

func isCatalogOrSchemaField(name string) bool {
	switch name {
	case "Catalog", "CatalogName", "Schema", "SchemaName", "Target":
		return true
	default:
		return false
	}
}

func cleanedValue(value string) string {
	value = strings.ReplaceAll(value, "<catalog>.", "")
	value = strings.ReplaceAll(value, "<schema>.", "")
	value = strings.ReplaceAll(value, "<catalog>", "")
	value = strings.ReplaceAll(value, "<schema>", "")
	return value
}

// replaceCatalogSchemaPlaceholders replaces placeholders with the final expected values.
func replaceCatalogSchemaPlaceholders(value string) string {
	value = strings.ReplaceAll(value, "<catalog>", "my_catalog")
	value = strings.ReplaceAll(value, "<schema>", "my_schema")
	return value
}
