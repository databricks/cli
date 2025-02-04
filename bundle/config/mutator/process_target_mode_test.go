package mutator

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/cli/libs/vfs"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockBundle(mode config.Mode) *bundle.Bundle {
	return &bundle.Bundle{
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
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							Name: "job1",
							Schedule: &jobs.CronSchedule{
								QuartzCronExpression: "* * * * *",
							},
							Tags: map[string]string{"existing": "tag"},
						},
					},
					"job2": {
						JobSettings: &jobs.JobSettings{
							Name: "job2",
							Schedule: &jobs.CronSchedule{
								QuartzCronExpression: "* * * * *",
								PauseStatus:          jobs.PauseStatusUnpaused,
							},
						},
					},
					"job3": {
						JobSettings: &jobs.JobSettings{
							Name: "job3",
							Trigger: &jobs.TriggerSettings{
								FileArrival: &jobs.FileArrivalTriggerConfiguration{
									Url: "test.com",
								},
							},
						},
					},
					"job4": {
						JobSettings: &jobs.JobSettings{
							Name: "job4",
							Continuous: &jobs.Continuous{
								PauseStatus: jobs.PauseStatusPaused,
							},
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline1": {CreatePipeline: &pipelines.CreatePipeline{Name: "pipeline1", Continuous: true}},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment1": {Experiment: &ml.Experiment{Name: "/Users/lennart.kats@databricks.com/experiment1"}},
					"experiment2": {Experiment: &ml.Experiment{Name: "experiment2"}},
				},
				Models: map[string]*resources.MlflowModel{
					"model1": {Model: &ml.Model{Name: "model1"}},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"servingendpoint1": {CreateServingEndpoint: &serving.CreateServingEndpoint{Name: "servingendpoint1"}},
				},
				RegisteredModels: map[string]*resources.RegisteredModel{
					"registeredmodel1": {CreateRegisteredModelRequest: &catalog.CreateRegisteredModelRequest{Name: "registeredmodel1"}},
				},
				QualityMonitors: map[string]*resources.QualityMonitor{
					"qualityMonitor1": {
						TableName: "qualityMonitor1",
						CreateMonitor: &catalog.CreateMonitor{
							OutputSchemaName: "catalog.schema",
						},
					},
					"qualityMonitor2": {
						TableName: "qualityMonitor2",
						CreateMonitor: &catalog.CreateMonitor{
							OutputSchemaName: "catalog.schema",
							Schedule:         &catalog.MonitorCronSchedule{},
						},
					},
					"qualityMonitor3": {
						TableName: "qualityMonitor3",
						CreateMonitor: &catalog.CreateMonitor{
							OutputSchemaName: "catalog.schema",
							Schedule: &catalog.MonitorCronSchedule{
								PauseStatus: catalog.MonitorCronSchedulePauseStatusUnpaused,
							},
						},
					},
				},
				Schemas: map[string]*resources.Schema{
					"schema1": {CreateSchema: &catalog.CreateSchema{Name: "schema1"}},
				},
				Volumes: map[string]*resources.Volume{
					"volume1": {CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{Name: "volume1"}},
				},
				Clusters: map[string]*resources.Cluster{
					"cluster1": {ClusterSpec: &compute.ClusterSpec{ClusterName: "cluster1", SparkVersion: "13.2.x", NumWorkers: 1}},
				},
				Dashboards: map[string]*resources.Dashboard{
					"dashboard1": {
						Dashboard: &dashboards.Dashboard{
							DisplayName: "dashboard1",
						},
					},
				},
				Apps: map[string]*resources.App{
					"app1": {
						App: &apps.App{
							Name: "app1",
						},
					},
				},
			},
		},
		SyncRoot: vfs.MustNew("/Users/lennart.kats@databricks.com"),
		// Use AWS implementation for testing.
		Tagging: tags.ForCloud(&sdkconfig.Config{
			Host: "https://company.cloud.databricks.com",
		}),
	}
}

func TestProcessTargetModeDevelopment(t *testing.T) {
	b := mockBundle(config.Development)

	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
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
	assert.False(t, b.Config.Resources.Pipelines["pipeline1"].Continuous)
	assert.True(t, b.Config.Resources.Pipelines["pipeline1"].CreatePipeline.Development)

	// Experiment 1
	assert.Equal(t, "/Users/lennart.kats@databricks.com/[dev lennart] experiment1", b.Config.Resources.Experiments["experiment1"].Name)
	assert.Contains(t, b.Config.Resources.Experiments["experiment1"].Experiment.Tags, ml.ExperimentTag{Key: "dev", Value: "lennart"})
	assert.Equal(t, "dev", b.Config.Resources.Experiments["experiment1"].Experiment.Tags[0].Key)

	// Experiment 2
	assert.Equal(t, "[dev lennart] experiment2", b.Config.Resources.Experiments["experiment2"].Name)
	assert.Contains(t, b.Config.Resources.Experiments["experiment2"].Experiment.Tags, ml.ExperimentTag{Key: "dev", Value: "lennart"})

	// Model 1
	assert.Equal(t, "[dev lennart] model1", b.Config.Resources.Models["model1"].Name)
	assert.Contains(t, b.Config.Resources.Models["model1"].Tags, ml.ModelTag{Key: "dev", Value: "lennart"})

	// Model serving endpoint 1
	assert.Equal(t, "dev_lennart_servingendpoint1", b.Config.Resources.ModelServingEndpoints["servingendpoint1"].Name)

	// Registered model 1
	assert.Equal(t, "dev_lennart_registeredmodel1", b.Config.Resources.RegisteredModels["registeredmodel1"].Name)

	// Quality Monitor 1
	assert.Equal(t, "qualityMonitor1", b.Config.Resources.QualityMonitors["qualityMonitor1"].TableName)
	assert.Nil(t, b.Config.Resources.QualityMonitors["qualityMonitor2"].Schedule)
	assert.Equal(t, catalog.MonitorCronSchedulePauseStatusUnpaused, b.Config.Resources.QualityMonitors["qualityMonitor3"].Schedule.PauseStatus)

	// Schema 1
	assert.Equal(t, "dev_lennart_schema1", b.Config.Resources.Schemas["schema1"].Name)

	// Clusters
	assert.Equal(t, "[dev lennart] cluster1", b.Config.Resources.Clusters["cluster1"].ClusterName)

	// Dashboards
	assert.Equal(t, "[dev lennart] dashboard1", b.Config.Resources.Dashboards["dashboard1"].DisplayName)
}

func TestProcessTargetModeDevelopmentTagNormalizationForAws(t *testing.T) {
	b := mockBundle(config.Development)
	b.Tagging = tags.ForCloud(&sdkconfig.Config{
		Host: "https://dbc-XXXXXXXX-YYYY.cloud.databricks.com/",
	})

	b.Config.Workspace.CurrentUser.ShortName = "Héllö wörld?!"
	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
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
	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
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
	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	// Assert that tag normalization took place.
	assert.Equal(t, "Hello_world", b.Config.Resources.Jobs["job1"].Tags["dev"])
}

func TestValidateDevelopmentMode(t *testing.T) {
	// Test with a valid development mode bundle
	b := mockBundle(config.Development)
	diags := validateDevelopmentMode(b)
	require.NoError(t, diags.Error())

	// Test with /Volumes path
	b = mockBundle(config.Development)
	b.Config.Workspace.ArtifactPath = "/Volumes/catalog/schema/lennart/libs"
	diags = validateDevelopmentMode(b)
	require.NoError(t, diags.Error())
	b.Config.Workspace.ArtifactPath = "/Volumes/catalog/schema/libs"
	diags = validateDevelopmentMode(b)
	require.ErrorContains(t, diags.Error(), "artifact_path should contain the current username or ${workspace.current_user.short_name} to ensure uniqueness when using 'mode: development'")

	// Test with a bundle that has a non-user path
	b = mockBundle(config.Development)
	b.Config.Workspace.RootPath = "/Shared/.bundle/x/y/state"
	diags = validateDevelopmentMode(b)
	require.ErrorContains(t, diags.Error(), "root_path must start with '~/' or contain the current username to ensure uniqueness when using 'mode: development'")

	// Test with a bundle that has an unpaused trigger pause status
	b = mockBundle(config.Development)
	b.Config.Presets.TriggerPauseStatus = config.Unpaused
	diags = validateDevelopmentMode(b)
	require.ErrorContains(t, diags.Error(), "UNPAUSED")

	// Test with a bundle that has a prefix not containing the username or short name
	b = mockBundle(config.Development)
	b.Config.Presets.NamePrefix = "[prod]"
	diags = validateDevelopmentMode(b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "")

	// Test with a bundle that has valid user paths
	b = mockBundle(config.Development)
	b.Config.Workspace.RootPath = "/Users/lennart@company.com/.bundle/x/y/state"
	b.Config.Workspace.StatePath = "/Users/lennart@company.com/.bundle/x/y/state"
	b.Config.Workspace.FilePath = "/Users/lennart@company.com/.bundle/x/y/files"
	b.Config.Workspace.ArtifactPath = "/Users/lennart@company.com/.bundle/x/y/artifacts"
	b.Config.Workspace.ResourcePath = "/Users/lennart@company.com/.bundle/x/y/resources"
	diags = validateDevelopmentMode(b)
	require.NoError(t, diags.Error())
}

func TestProcessTargetModeDefault(t *testing.T) {
	b := mockBundle("")

	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())
	assert.Equal(t, "job1", b.Config.Resources.Jobs["job1"].Name)
	assert.Equal(t, "pipeline1", b.Config.Resources.Pipelines["pipeline1"].Name)
	assert.False(t, b.Config.Resources.Pipelines["pipeline1"].CreatePipeline.Development)
	assert.Equal(t, "servingendpoint1", b.Config.Resources.ModelServingEndpoints["servingendpoint1"].Name)
	assert.Equal(t, "registeredmodel1", b.Config.Resources.RegisteredModels["registeredmodel1"].Name)
	assert.Equal(t, "qualityMonitor1", b.Config.Resources.QualityMonitors["qualityMonitor1"].TableName)
	assert.Equal(t, "schema1", b.Config.Resources.Schemas["schema1"].Name)
	assert.Equal(t, "volume1", b.Config.Resources.Volumes["volume1"].Name)
	assert.Equal(t, "cluster1", b.Config.Resources.Clusters["cluster1"].ClusterName)
}

func TestProcessTargetModeProduction(t *testing.T) {
	b := mockBundle(config.Production)

	diags := validateProductionMode(context.Background(), b, false)
	require.ErrorContains(t, diags.Error(), "target with 'mode: production' must set 'workspace.root_path' to make sure only one copy is deployed. A common practice is to use a username or principal name in this path, i.e. root_path: /Workspace/Users/lennart@company.com/.bundle/${bundle.name}/${bundle.target}")

	b.Config.Workspace.StatePath = "/Shared/.bundle/x/y/state"
	b.Config.Workspace.ArtifactPath = "/Shared/.bundle/x/y/artifacts"
	b.Config.Workspace.FilePath = "/Shared/.bundle/x/y/files"
	b.Config.Workspace.ResourcePath = "/Shared/.bundle/x/y/resources"

	diags = validateProductionMode(context.Background(), b, false)
	require.ErrorContains(t, diags.Error(), "target with 'mode: production' must set 'workspace.root_path' to make sure only one copy is deployed. A common practice is to use a username or principal name in this path, i.e. root_path: /Workspace/Users/lennart@company.com/.bundle/${bundle.name}/${bundle.target}")

	permissions := []resources.Permission{
		{
			Level:    "CAN_MANAGE",
			UserName: "user@company.com",
		},
	}
	b.Config.Resources.Jobs["job1"].Permissions = permissions
	b.Config.Resources.Jobs["job1"].RunAs = &jobs.JobRunAs{UserName: "user@company.com"}
	b.Config.Resources.Jobs["job2"].RunAs = &jobs.JobRunAs{UserName: "user@company.com"}
	b.Config.Resources.Jobs["job3"].RunAs = &jobs.JobRunAs{UserName: "user@company.com"}
	b.Config.Resources.Jobs["job4"].RunAs = &jobs.JobRunAs{UserName: "user@company.com"}
	b.Config.Resources.Pipelines["pipeline1"].Permissions = permissions
	b.Config.Resources.Experiments["experiment1"].Permissions = permissions
	b.Config.Resources.Experiments["experiment2"].Permissions = permissions
	b.Config.Resources.Models["model1"].Permissions = permissions
	b.Config.Resources.ModelServingEndpoints["servingendpoint1"].Permissions = permissions
	b.Config.Resources.Clusters["cluster1"].Permissions = permissions

	diags = validateProductionMode(context.Background(), b, false)
	require.NoError(t, diags.Error())

	assert.Equal(t, "job1", b.Config.Resources.Jobs["job1"].Name)
	assert.Equal(t, "pipeline1", b.Config.Resources.Pipelines["pipeline1"].Name)
	assert.False(t, b.Config.Resources.Pipelines["pipeline1"].CreatePipeline.Development)
	assert.Equal(t, "servingendpoint1", b.Config.Resources.ModelServingEndpoints["servingendpoint1"].Name)
	assert.Equal(t, "registeredmodel1", b.Config.Resources.RegisteredModels["registeredmodel1"].Name)
	assert.Equal(t, "qualityMonitor1", b.Config.Resources.QualityMonitors["qualityMonitor1"].TableName)
	assert.Equal(t, "schema1", b.Config.Resources.Schemas["schema1"].Name)
	assert.Equal(t, "volume1", b.Config.Resources.Volumes["volume1"].Name)
	assert.Equal(t, "cluster1", b.Config.Resources.Clusters["cluster1"].ClusterName)
}

func TestProcessTargetModeProductionOkForPrincipal(t *testing.T) {
	b := mockBundle(config.Production)

	// Our target has all kinds of problems when not using service principals ...
	diags := validateProductionMode(context.Background(), b, false)
	require.Error(t, diags.Error())

	// ... but we're much less strict when a principal is used
	diags = validateProductionMode(context.Background(), b, true)
	require.NoError(t, diags.Error())
}

func TestProcessTargetModeProductionOkWithRootPath(t *testing.T) {
	b := mockBundle(config.Production)

	// Our target has all kinds of problems when not using service principals ...
	diags := validateProductionMode(context.Background(), b, false)
	require.Error(t, diags.Error())

	// ... but we're okay if we specify a root path
	b.Target = &config.Target{
		Workspace: &config.Workspace{
			RootPath: "some-root-path",
		},
	}
	diags = validateProductionMode(context.Background(), b, false)
	require.NoError(t, diags.Error())
}

// Make sure that we have test coverage for all resource types
func TestAllResourcesMocked(t *testing.T) {
	b := mockBundle(config.Development)
	resources := reflect.ValueOf(b.Config.Resources)

	for i := range resources.NumField() {
		field := resources.Field(i)
		if field.Kind() == reflect.Map {
			assert.True(
				t,
				!field.IsNil() && field.Len() > 0,
				"process_target_mode should support '%s' (please add it to process_target_mode.go and extend the test suite)",
				resources.Type().Field(i).Name,
			)
		}
	}
}

// Make sure that we at rename all non UC resources
func TestAllNonUcResourcesAreRenamed(t *testing.T) {
	b := mockBundle(config.Development)

	// UC resources should not have a prefix added to their name. Right now
	// this list only contains the Volume resource since we have yet to remove
	// prefixing support for UC schemas and registered models.
	ucFields := []reflect.Type{
		reflect.TypeOf(&resources.Volume{}),
	}

	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	resources := reflect.ValueOf(b.Config.Resources)
	for i := range resources.NumField() {
		field := resources.Field(i)

		if field.Kind() == reflect.Map {
			for _, key := range field.MapKeys() {
				resource := field.MapIndex(key)
				nameField := resource.Elem().FieldByName("Name")
				resourceType := resources.Type().Field(i).Name

				// Skip apps, as they are not renamed
				if resourceType == "Apps" {
					continue
				}

				if !nameField.IsValid() || nameField.Kind() != reflect.String {
					continue
				}

				if slices.Contains(ucFields, resource.Type()) {
					assert.NotContains(t, nameField.String(), "dev", "process_target_mode should not rename '%s' in '%s'", key, resources.Type().Field(i).Name)
				} else {
					assert.Contains(t, nameField.String(), "dev", "process_target_mode should rename '%s' in '%s'", key, resources.Type().Field(i).Name)
				}
			}
		}
	}
}

func TestDisableLocking(t *testing.T) {
	ctx := context.Background()
	b := mockBundle(config.Development)

	transformDevelopmentMode(ctx, b)
	assert.False(t, b.Config.Bundle.Deployment.Lock.IsEnabled())
}

func TestDisableLockingDisabled(t *testing.T) {
	ctx := context.Background()
	b := mockBundle(config.Development)
	explicitlyEnabled := true
	b.Config.Bundle.Deployment.Lock.Enabled = &explicitlyEnabled

	transformDevelopmentMode(ctx, b)
	assert.True(t, b.Config.Bundle.Deployment.Lock.IsEnabled(), "Deployment lock should remain enabled in development mode when explicitly enabled")
}

func TestPrefixAlreadySet(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.NamePrefix = "custom_lennart_deploy_"

	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	assert.Equal(t, "custom_lennart_deploy_job1", b.Config.Resources.Jobs["job1"].Name)
}

func TestTagsAlreadySet(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.Tags = map[string]string{
		"custom": "tag",
		"dev":    "foo",
	}

	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	assert.Equal(t, "tag", b.Config.Resources.Jobs["job1"].Tags["custom"])
	assert.Equal(t, "foo", b.Config.Resources.Jobs["job1"].Tags["dev"])
}

func TestTagsNil(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.Tags = nil

	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	assert.Equal(t, "lennart", b.Config.Resources.Jobs["job2"].Tags["dev"])
}

func TestTagsEmptySet(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.Tags = map[string]string{}

	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	assert.Equal(t, "lennart", b.Config.Resources.Jobs["job2"].Tags["dev"])
}

func TestJobsMaxConcurrentRunsAlreadySet(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.JobsMaxConcurrentRuns = 10

	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	assert.Equal(t, 10, b.Config.Resources.Jobs["job1"].MaxConcurrentRuns)
}

func TestJobsMaxConcurrentRunsDisabled(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.JobsMaxConcurrentRuns = 1

	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	assert.Equal(t, 1, b.Config.Resources.Jobs["job1"].MaxConcurrentRuns)
}

func TestTriggerPauseStatusWhenUnpaused(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.TriggerPauseStatus = config.Unpaused

	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
	require.ErrorContains(t, diags.Error(), "target with 'mode: development' cannot set trigger pause status to UNPAUSED by default")
}

func TestPipelinesDevelopmentDisabled(t *testing.T) {
	b := mockBundle(config.Development)
	notEnabled := false
	b.Config.Presets.PipelinesDevelopment = &notEnabled

	m := bundle.Seq(ProcessTargetMode(), ApplyPresets())
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	assert.False(t, b.Config.Resources.Pipelines["pipeline1"].CreatePipeline.Development)
}
