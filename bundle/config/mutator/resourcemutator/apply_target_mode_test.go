package resourcemutator

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/cli/libs/vfs"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FIXME apply_target_mode_test should be untied from apply_presets_test
// and apply_target_mode with validate_target_mode should be moved into mutators package.

// targetModeExamples maps a resource key to a function that puts the example
// resources of that type into a config.Resources. Each type registers its
// examples from <key>_target_mode_test.go. They are functions so that every
// mockBundle call gets fresh values that a test can modify.
var targetModeExamples = map[string]func(*config.Resources){}

func registerTargetModeExamples(key string, set func(*config.Resources)) {
	if _, ok := targetModeExamples[key]; ok {
		panic(fmt.Sprintf("target mode examples for %q are registered more than once", key))
	}
	targetModeExamples[key] = set
}

func mockBundle(mode config.Mode) *bundle.Bundle {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Mode: mode,
				Git: config.Git{
					OriginURL: "http://origin",
					Branch:    "main",
				},
			},
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					ShortName: "lennart",
					User: &iam.User{
						UserName: "lennart@company.com",
						Id:       "1",
					},
				},
				StatePath:    "/Users/lennart@company.com/.bundle/x/y/state",
				ArtifactPath: "/Users/lennart@company.com/.bundle/x/y/artifacts",
				FilePath:     "/Users/lennart@company.com/.bundle/x/y/files",
			},
		},
		SyncRoot: vfs.MustNew("/Users/lennart.kats@databricks.com"),
		// Use AWS implementation for testing.
		Tagging: tags.ForCloud(&sdkconfig.Config{
			Host: "https://company.cloud.databricks.com",
		}),
	}
	for _, key := range slices.Sorted(maps.Keys(targetModeExamples)) {
		targetModeExamples[key](&b.Config.Resources)
	}
	return b
}

func TestProcessTargetModeDevelopment(t *testing.T) {
	b := mockBundle(config.Development)

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	// Job 1
	assert.Equal(t, "[dev lennart] job1", b.Config.Resources.Jobs["job1"].Name)
	assert.Equal(t, "tag", b.Config.Resources.Jobs["job1"].Tags["existing"])
	assert.Equal(t, "lennart", b.Config.Resources.Jobs["job1"].Tags["dev"])
	assert.Equal(t, jobs.PauseStatusPaused, b.Config.Resources.Jobs["job1"].Schedule.PauseStatus)

	// Job 2
	assert.Equal(t, "[dev lennart] job2", b.Config.Resources.Jobs["job2"].Name)
	assert.Equal(t, "lennart", b.Config.Resources.Jobs["job2"].Tags["dev"])
	assert.Equal(t, jobs.PauseStatusUnpaused, b.Config.Resources.Jobs["job2"].Schedule.PauseStatus)

	// Pipeline 1
	assert.Equal(t, "[dev lennart] pipeline1", b.Config.Resources.Pipelines["pipeline1"].Name)
	assert.False(t, b.Config.Resources.Pipelines["pipeline1"].Continuous) //nolint:staticcheck // SA1019: pipeline continuous is deprecated in the SDK but remains a supported bundle config field
	assert.True(t, b.Config.Resources.Pipelines["pipeline1"].Development) //nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field

	// Experiment 1
	assert.Equal(t, "/Users/lennart.kats@databricks.com/[dev lennart] experiment1", b.Config.Resources.Experiments["experiment1"].Name)
	assert.Contains(t, b.Config.Resources.Experiments["experiment1"].Tags, ml.ExperimentTag{Key: "dev", Value: "lennart"})
	assert.Equal(t, "dev", b.Config.Resources.Experiments["experiment1"].CreateExperiment.Tags[0].Key)

	// Experiment 2
	assert.Equal(t, "[dev lennart] experiment2", b.Config.Resources.Experiments["experiment2"].Name)
	assert.Contains(t, b.Config.Resources.Experiments["experiment2"].Tags, ml.ExperimentTag{Key: "dev", Value: "lennart"})

	// Model 1
	assert.Equal(t, "[dev lennart] model1", b.Config.Resources.Models["model1"].Name)
	assert.Contains(t, b.Config.Resources.Models["model1"].Tags, ml.ModelTag{Key: "dev", Value: "lennart"})

	// Model serving endpoint 1
	assert.Equal(t, "dev_lennart_servingendpoint1", b.Config.Resources.ModelServingEndpoints["servingendpoint1"].Name)

	// Vector search endpoint 1: name is the primary key, so it must not be prefixed.
	assert.Equal(t, "vs_endpoint1", b.Config.Resources.VectorSearchEndpoints["vs_endpoint1"].Name)

	// Vector search index 1: name is the primary key, so it must not be prefixed.
	assert.Equal(t, "main.default.vs_index1", b.Config.Resources.VectorSearchIndexes["vs_index1"].Name)

	// Registered model 1
	assert.Equal(t, "dev_lennart_registeredmodel1", b.Config.Resources.RegisteredModels["registeredmodel1"].Name)

	// Quality Monitor 1
	assert.Equal(t, "qualityMonitor1", b.Config.Resources.QualityMonitors["qualityMonitor1"].TableName)
	assert.Nil(t, b.Config.Resources.QualityMonitors["qualityMonitor2"].Schedule)
	assert.Equal(t, catalog.MonitorCronSchedulePauseStatusUnpaused, b.Config.Resources.QualityMonitors["qualityMonitor3"].Schedule.PauseStatus)
	assert.Nil(t, b.Config.Resources.DataQualityMonitors["dataQualityMonitor2"].DataProfilingConfig.Schedule)

	// Schema 1
	assert.Equal(t, "dev_lennart_schema1", b.Config.Resources.Schemas["schema1"].Name)

	// Clusters
	assert.Equal(t, "[dev lennart] cluster1", b.Config.Resources.Clusters["cluster1"].ClusterName)

	// Instance pools
	assert.Equal(t, "[dev lennart] instance_pool1", b.Config.Resources.InstancePools["instance_pool1"].InstancePoolName)

	// Dashboards
	assert.Equal(t, "[dev lennart] dashboard1", b.Config.Resources.Dashboards["dashboard1"].DisplayName)

	// Genie Spaces
	assert.Equal(t, "[dev lennart] geniespace1", b.Config.Resources.GenieSpaces["geniespace1"].Title)

	// Alert 1: has schedule without pause status set - should be paused
	assert.Equal(t, "[dev lennart] alert1", b.Config.Resources.Alerts["alert1"].DisplayName)
	assert.Equal(t, sql.SchedulePauseStatusPaused, b.Config.Resources.Alerts["alert1"].Schedule.PauseStatus)

	// Alert 2: has schedule with pause status already set to unpaused - should remain unpaused
	assert.Equal(t, "[dev lennart] alert2", b.Config.Resources.Alerts["alert2"].DisplayName)
	assert.Equal(t, sql.SchedulePauseStatusUnpaused, b.Config.Resources.Alerts["alert2"].Schedule.PauseStatus)

	// Alert 3: no schedule - pause status should remain empty
	assert.Equal(t, "[dev lennart] alert3", b.Config.Resources.Alerts["alert3"].DisplayName)
	assert.Empty(t, b.Config.Resources.Alerts["alert3"].Schedule.PauseStatus)
}

func TestProcessTargetModeDevelopmentTagNormalizationForAws(t *testing.T) {
	b := mockBundle(config.Development)
	b.Tagging = tags.ForCloud(&sdkconfig.Config{
		Host: "https://dbc-XXXXXXXX-YYYY.cloud.databricks.com/",
	})

	b.Config.Workspace.CurrentUser.ShortName = "Héllö wörld?!"
	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	// Assert that tag normalization took place.
	assert.Equal(t, "Hello world__", b.Config.Resources.Jobs["job1"].Tags["dev"])
}

func TestProcessTargetModeDevelopmentTagNormalizationForAzure(t *testing.T) {
	b := mockBundle(config.Development)
	b.Tagging = tags.ForCloud(&sdkconfig.Config{
		Host: "https://adb-xxx.y.azuredatabricks.net/",
	})

	b.Config.Workspace.CurrentUser.ShortName = "Héllö wörld?!"
	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	// Assert that tag normalization took place (Azure allows more characters than AWS).
	assert.Equal(t, "Héllö wörld?!", b.Config.Resources.Jobs["job1"].Tags["dev"])
}

func TestProcessTargetModeDevelopmentTagNormalizationForGcp(t *testing.T) {
	b := mockBundle(config.Development)
	b.Tagging = tags.ForCloud(&sdkconfig.Config{
		Host: "https://123.4.gcp.databricks.com/",
	})

	b.Config.Workspace.CurrentUser.ShortName = "Héllö wörld?!"
	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	// Assert that tag normalization took place.
	assert.Equal(t, "Hello_world", b.Config.Resources.Jobs["job1"].Tags["dev"])
}

func TestProcessTargetModeDefault(t *testing.T) {
	b := mockBundle("")

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())
	assert.Equal(t, "job1", b.Config.Resources.Jobs["job1"].Name)
	assert.Equal(t, "pipeline1", b.Config.Resources.Pipelines["pipeline1"].Name)
	assert.False(t, b.Config.Resources.Pipelines["pipeline1"].Development) //nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
	assert.Equal(t, "servingendpoint1", b.Config.Resources.ModelServingEndpoints["servingendpoint1"].Name)
	assert.Equal(t, "registeredmodel1", b.Config.Resources.RegisteredModels["registeredmodel1"].Name)
	assert.Equal(t, "qualityMonitor1", b.Config.Resources.QualityMonitors["qualityMonitor1"].TableName)
	assert.Equal(t, "catalog1", b.Config.Resources.Catalogs["catalog1"].Name)
	assert.Equal(t, "schema1", b.Config.Resources.Schemas["schema1"].Name)
	assert.Equal(t, "volume1", b.Config.Resources.Volumes["volume1"].Name)
	assert.Equal(t, "catalog1.schema1.feature1", b.Config.Resources.Features["feature1"].FullName)
	assert.Equal(t, "catalog1.schema1.feature_spec1", b.Config.Resources.FeatureSpecs["feature_spec1"].Name)
	assert.Equal(t, "cluster1", b.Config.Resources.Clusters["cluster1"].ClusterName)
	assert.Equal(t, "instance_pool1", b.Config.Resources.InstancePools["instance_pool1"].InstancePoolName)
	assert.Equal(t, "sql_warehouse1", b.Config.Resources.SqlWarehouses["sql_warehouse1"].Name)
}

// Make sure that we have test coverage for all resource types
func TestAllResourcesMocked(t *testing.T) {
	types := config.AllResourceTypes()
	for _, rt := range types {
		if _, ok := targetModeExamples[rt.Key]; !ok {
			t.Errorf("resource type %q has no examples for the target mode tests; add %s_target_mode_test.go that registers them with registerTargetModeExamples", rt.Key, rt.Key)
		}
	}

	for _, key := range slices.Sorted(maps.Keys(targetModeExamples)) {
		if _, ok := config.ResourceTypeAs[config.ConfigResource](key); !ok {
			t.Errorf("registerTargetModeExamples(%q): %q is not a resource type", key, key)
			continue
		}

		// A registration fills the group under its own key and no other, so that
		// mockBundle does not depend on the order in which it applies them.
		var r config.Resources
		targetModeExamples[key](&r)
		for i, group := range r.AllResources() {
			if types[i].Key == key {
				assert.NotEmpty(t, group.Resources, "registerTargetModeExamples(%q) adds no %s", key, key)
			} else {
				assert.Empty(t, group.Resources, "registerTargetModeExamples(%q) adds %s", key, types[i].Key)
			}
		}
	}
}

// TestAppropriateResourcesAreRenamed checks that dev-mode / presets.name_prefix
// renames every resource, unless its type implements resources.NamePrefixExempt,
// in which case its name must stay unchanged. TestAllResourcesMocked makes sure
// that the mock bundle has a resource of every type.
func TestAppropriateResourcesAreRenamed(t *testing.T) {
	b := mockBundle(config.Development)

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	for _, group := range b.Config.Resources.AllResources() {
		for key, r := range group.Resources {
			name := r.GetName()
			prefixed := strings.Contains(name, "[dev lennart] ") || strings.Contains(name, "dev_lennart_")
			if e, ok := r.(resources.NamePrefixExempt); ok {
				assert.NotEmpty(t, e.NamePrefixExemptReason(), "%s.%s", group.Description.PluralName, key)
				assert.False(t, prefixed, "%s.%s implements resources.NamePrefixExempt but its name %q is prefixed", group.Description.PluralName, key, name)
			} else {
				assert.True(t, prefixed, "%s.%s: name %q is not prefixed; prefix it in ApplyPresets or implement resources.NamePrefixExempt", group.Description.PluralName, key, name)
			}
		}
	}
}

func TestDisableLocking(t *testing.T) {
	ctx := t.Context()
	b := mockBundle(config.Development)

	diags := bundle.ApplySeq(ctx, b, ApplyTargetMode())
	require.NoError(t, diags.Error())

	assert.False(t, b.Config.Bundle.Deployment.Lock.IsEnabled())
}

func TestDisableLockingDisabled(t *testing.T) {
	ctx := t.Context()
	b := mockBundle(config.Development)
	explicitlyEnabled := true
	b.Config.Bundle.Deployment.Lock.Enabled = &explicitlyEnabled

	diags := bundle.ApplySeq(ctx, b, ApplyTargetMode())
	require.NoError(t, diags.Error())
	assert.True(t, b.Config.Bundle.Deployment.Lock.IsEnabled(), "Deployment lock should remain enabled in development mode when explicitly enabled")
}

func TestPrefixAlreadySet(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.NamePrefix = "custom_lennart_deploy_"

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	assert.Equal(t, "custom_lennart_deploy_job1", b.Config.Resources.Jobs["job1"].Name)
}

func TestTagsAlreadySet(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.Tags = map[string]string{
		"custom": "tag",
		"dev":    "foo",
	}

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	assert.Equal(t, "tag", b.Config.Resources.Jobs["job1"].Tags["custom"])
	assert.Equal(t, "foo", b.Config.Resources.Jobs["job1"].Tags["dev"])
}

func TestTagsNil(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.Tags = nil

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	assert.Equal(t, "lennart", b.Config.Resources.Jobs["job2"].Tags["dev"])
}

func TestTagsEmptySet(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.Tags = map[string]string{}

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	assert.Equal(t, "lennart", b.Config.Resources.Jobs["job2"].Tags["dev"])
}

func TestJobsMaxConcurrentRunsAlreadySet(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.JobsMaxConcurrentRuns = 10

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	assert.Equal(t, 10, b.Config.Resources.Jobs["job1"].MaxConcurrentRuns)
}

func TestJobsMaxConcurrentRunsDisabled(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.JobsMaxConcurrentRuns = 1

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	assert.Equal(t, 1, b.Config.Resources.Jobs["job1"].MaxConcurrentRuns)
}

func TestPipelinesDevelopmentDisabled(t *testing.T) {
	b := mockBundle(config.Development)
	notEnabled := false
	b.Config.Presets.PipelinesDevelopment = &notEnabled

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	assert.False(t, b.Config.Resources.Pipelines["pipeline1"].Development) //nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
}
