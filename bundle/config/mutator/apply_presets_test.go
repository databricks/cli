package mutator_test

import (
	"context"
	"fmt"
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
	"github.com/stretchr/testify/require"
)

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
	// Create a bundle in a known mode, e.g. development or production doesn't matter much here.
	b := mockBundle(config.Development)
	// Set the catalog and schema in presets.
	b.Config.Presets.Catalog = "my_catalog"
	b.Config.Presets.Schema = "my_schema"

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, mutator.ApplyPresets())
	require.NoError(t, diags.Error())

	// Verify that jobs got catalog/schema if they support it.
	// For DBT tasks in jobs:
	for _, job := range b.Config.Resources.Jobs {
		if job.JobSettings != nil && job.Tasks != nil {
			for _, task := range job.Tasks {
				if task.DbtTask != nil {
					require.Equal(t, "my_catalog", task.DbtTask.Catalog, "dbt catalog should be set")
					require.Equal(t, "my_schema", task.DbtTask.Schema, "dbt schema should be set")
				}
			}
		}
	}

	// Pipelines: Catalog/Schema
	for _, p := range b.Config.Resources.Pipelines {
		if p.PipelineSpec != nil {
			// pipeline catalog and schema
			if p.Catalog == "" || p.Catalog == "hive_metastore" {
				require.Equal(t, "my_catalog", p.Catalog, "pipeline catalog should be set")
			}
			require.Equal(t, "my_schema", p.Target, "pipeline schema (target) should be set")
		}
	}

	// Registered models: Catalog/Schema
	for _, rm := range b.Config.Resources.RegisteredModels {
		if rm.CreateRegisteredModelRequest != nil {
			require.Equal(t, "my_catalog", rm.CatalogName, "registered model catalog should be set")
			require.Equal(t, "my_schema", rm.SchemaName, "registered model schema should be set")
		}
	}

	// Quality monitors: If paused, we rewrite tableName to include catalog.schema.
	// In our code, if paused, we prepend catalog/schema if tableName wasn't already fully qualified.
	// Let's verify that:
	for _, qm := range b.Config.Resources.QualityMonitors {
		// If not fully qualified (3 parts), it should have been rewritten.
		parts := strings.Split(qm.TableName, ".")
		if len(parts) != 3 {
			require.Equal(t, fmt.Sprintf("my_catalog.my_schema.%s", parts[0]), qm.TableName, "quality monitor tableName should include catalog and schema")
		}
	}

	// Schemas: If there's a schema preset, we might replace the schema name or catalog name.
	for _, s := range b.Config.Resources.Schemas {
		if s.CreateSchema != nil {
			// If catalog was empty before, now should be set:
			require.Equal(t, "my_catalog", s.CatalogName, "schema catalog should be set")
			// If schema was empty before, it should be set, but we did have "schema1",
			// so let's verify that if schema had a name, prefix logic may apply:
			// The code attempts to handle schema naming carefully. If t.Schema != "" and s.Name == "",
			// s.Name is set to t.Schema. Since s.Name was originally "schema1", it should remain "schema1" with prefix applied.
			// If you want to verify behavior, do so explicitly if changed code logic.
		}
	}

	// Model serving endpoints currently return a warning that they don't support catalog/schema presets.
	// We can just verify that the warning is generated or that no fields were set since they are not supported.
	// The ApplyPresets code emits a diag error if we attempt to use catalog/schema with model serving endpoints.
	// Let's check that we got an error diagnostic:
	// The code currently returns a diag error if model serving endpoints are present and catalog/schema are set.
	// So we verify diags here:
	foundEndpointError := false
	for _, d := range diags {
		if strings.Contains(d.Summary, "model serving endpoints are not supported with catalog/schema presets") {
			foundEndpointError = true
			break
		}
	}
	require.True(t, foundEndpointError, "should have diag error for model serving endpoints")

	// Add assertions for any other resources that support catalog/schema if needed.
	// This list is maintained manually. If you add new resource types that support catalog/schema,
	// add them here as well.
}
