package resourcemutator

import (
	"github.com/databricks/cli/libs/qualitymonitors"
	"github.com/databricks/cli/libs/sdktypes/alerts/v2"
	"github.com/databricks/cli/libs/sdktypes/clusterpolicies/v2"
	"github.com/databricks/cli/libs/sdktypes/clusters/v2"
	"github.com/databricks/cli/libs/sdktypes/experiments/v1"
	"github.com/databricks/cli/libs/sdktypes/instancepools/v2"
	"github.com/databricks/cli/libs/sdktypes/modelregistry/v1"
	"github.com/databricks/cli/libs/sdktypes/uc/registeredmodels/v1"
	ucsecrets "github.com/databricks/cli/libs/sdktypes/uc/secrets/v1"
	"github.com/databricks/cli/libs/sdktypes/warehouses/v1"
	"github.com/databricks/cli/libs/utils"
	"reflect"
	"slices"
	"testing"

	"github.com/databricks/cli/libs/sdktypes/database/v1"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdktypes/apps/v1"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/sdktypes/modelserving/v1"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"github.com/databricks/cli/libs/sdktypes/postgres/v1"
	"github.com/databricks/cli/libs/sdktypes/scim/v1"
	"github.com/databricks/cli/libs/sdktypes/vectorsearch/v1"
	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/cli/libs/vfs"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FIXME apply_target_mode_test should be untied from apply_presets_test
// and apply_target_mode with validate_target_mode should be moved into mutators package.

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
					User: &scim.User{
						UserName: new("lennart@company.com"),
						Id:       new("1"),
					},
				},
				StatePath:    "/Users/lennart@company.com/.bundle/x/y/state",
				ArtifactPath: "/Users/lennart@company.com/.bundle/x/y/artifacts",
				FilePath:     "/Users/lennart@company.com/.bundle/x/y/files",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Name: new("job1"),
							Schedule: &jobs.CronSchedule{
								QuartzCronExpression: new("* * * * *"),
							},
							Tags: map[string]string{"existing": "tag"},
						},
					},
					"job2": {
						JobSettings: jobs.JobSettings{
							Name: new("job2"),
							Schedule: &jobs.CronSchedule{
								QuartzCronExpression: new("* * * * *"),
								PauseStatus:          jobs.SchedulePauseStatus("UNPAUSED"),
							},
						},
					},
					"job3": {
						JobSettings: jobs.JobSettings{
							Name: new("job3"),
							Trigger: &jobs.TriggerSettings{
								FileArrival: &jobs.FileArrivalTriggerConfiguration{
									Url: new("test.com"),
								},
							},
						},
					},
					"job4": {
						JobSettings: jobs.JobSettings{
							Name: new("job4"),
							Continuous: &jobs.ContinuousSettings{
								PauseStatus: jobs.SchedulePauseStatus_Paused,
							},
						},
					},
				},
				JobRuns: map[string]*resources.JobRun{
					"job_run1": {RunNowRequest: jobs.RunNowRequest{JobId: new(int64(1234))}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline1": {CreatePipelineRequest: pipelines.CreatePipelineRequest{Name: new("pipeline1"), Continuous: new(true)}}, //nolint:staticcheck // SA1019: pipeline continuous is deprecated in the SDK but remains a supported bundle config field
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment1": {CreateExperimentRequest: experiments.CreateExperimentRequest{Name: new("/Users/lennart.kats@databricks.com/experiment1")}},
					"experiment2": {CreateExperimentRequest: experiments.CreateExperimentRequest{Name: new("experiment2")}},
				},
				Models: map[string]*resources.MlflowModel{
					"model1": {CreateRegisteredModelRequest: modelregistry.CreateRegisteredModelRequest{Name: new("model1")}},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"servingendpoint1": {CreateInferenceEndpointRequest: modelserving.CreateInferenceEndpointRequest{Name: new("servingendpoint1")}},
				},
				ModelServices: map[string]*resources.ModelService{
					"modelservice1": {ModelServiceConfig: resources.ModelServiceConfig{Parent: "schemas/main.default", ModelServiceId: "modelservice1"}},
				},
				McpServices: map[string]*resources.McpService{
					"mcpservice1": {McpServiceConfig: resources.McpServiceConfig{Parent: "schemas/main.default", McpServiceId: "mcpservice1"}},
				},
				ModelProviderServices: map[string]*resources.ModelProviderService{
					"modelproviderservice1": {ModelProviderServiceConfig: resources.ModelProviderServiceConfig{Parent: "schemas/main.default", ModelProviderServiceId: "modelproviderservice1"}},
				},
				RegisteredModels: map[string]*resources.RegisteredModel{
					"registeredmodel1": {CreateRegisteredModelRequest: registeredmodels.CreateRegisteredModelRequest{Name: new("registeredmodel1")}},
				},
				QualityMonitors: map[string]*resources.QualityMonitor{
					"qualityMonitor1": {
						TableName: "qualityMonitor1",
						CreateMonitor: qualitymonitors.CreateMonitor{
							OutputSchemaName: "catalog.schema",
						},
					},
					"qualityMonitor2": {
						TableName: "qualityMonitor2",
						CreateMonitor: qualitymonitors.CreateMonitor{
							OutputSchemaName: "catalog.schema",
							Schedule:         &qualitymonitors.MonitorCronSchedule{},
						},
					},
					"qualityMonitor3": {
						TableName: "qualityMonitor3",
						CreateMonitor: qualitymonitors.CreateMonitor{
							OutputSchemaName: "catalog.schema",
							Schedule: &qualitymonitors.MonitorCronSchedule{
								PauseStatus: qualitymonitors.MonitorCronSchedulePauseStatusUnpaused,
							},
						},
					},
				},
				Catalogs: map[string]*resources.Catalog{
					"catalog1": {CreateCatalog: resources.CreateCatalog{Name: new("catalog1")}},
				},
				ExternalLocations: map[string]*resources.ExternalLocation{
					"externalLocation1": {CreateExternalLocation: resources.CreateExternalLocation{Name: new("externalLocation1")}},
				},
				Schemas: map[string]*resources.Schema{
					"schema1": {CreateSchema: resources.CreateSchema{Name: new("schema1")}},
				},
				Volumes: map[string]*resources.Volume{
					"volume1": {CreateVolumeRequestContent: resources.CreateVolumeRequestContent{Name: new("volume1")}},
				},
				Clusters: map[string]*resources.Cluster{
					"cluster1": {CreateClusterRequest: clusters.CreateClusterRequest{ClusterName: new("cluster1"), SparkVersion: new("13.2.x"), NumWorkers: new(1)}},
				},
				InstancePools: map[string]*resources.InstancePool{
					"instance_pool1": {CreateInstancePoolRequest: instancepools.CreateInstancePoolRequest{InstancePoolName: new("instance_pool1"), NodeTypeId: new("i3.xlarge")}},
				},
				ClusterPolicies: map[string]*resources.ClusterPolicy{
					"cluster_policy1": {CreatePolicyRequest: clusterpolicies.CreatePolicyRequest{Name: new("cluster_policy1")}},
				},
				Dashboards: map[string]*resources.Dashboard{
					"dashboard1": {
						DashboardConfig: resources.DashboardConfig{
							DisplayName: "dashboard1",
						},
					},
				},
				GenieSpaces: map[string]*resources.GenieSpace{
					"geniespace1": {
						GenieSpaceConfig: resources.GenieSpaceConfig{
							Title: "geniespace1",
						},
					},
				},
				Apps: map[string]*resources.App{
					"app1": {
						App: apps.App{
							Name: new("app1"),
						},
					},
				},
				SecretScopes: map[string]*resources.SecretScope{
					"secretScope1": {
						Name: "secretScope1",
					},
				},
				Secrets: map[string]*resources.Secret{
					"secret1": {
						Secret: ucsecrets.Secret{
							CatalogName: new("main"),
							SchemaName:  new("default"),
							Name:        new("secret1"),
						},
					},
				},
				SqlWarehouses: map[string]*resources.SqlWarehouse{
					"sql_warehouse1": {
						CreateWarehouseRequest: warehouses.CreateWarehouseRequest{
							Name: new("sql_warehouse1"),
						},
					},
				},
				DatabaseInstances: map[string]*resources.DatabaseInstance{
					"database_instance1": {
						DatabaseInstance: database.DatabaseInstance{
							Name: new("database_instance1"),
						},
					},
				},
				DatabaseCatalogs: map[string]*resources.DatabaseCatalog{
					"database_catalog1": {
						DatabaseCatalog: database.DatabaseCatalog{
							Name: new("database_catalog1"),
						},
					},
				},
				SyncedDatabaseTables: map[string]*resources.SyncedDatabaseTable{
					"synced_database_table1": {
						SyncedDatabaseTable: database.SyncedDatabaseTable{
							Name: new("synced_database_table1"),
						},
					},
				},
				Alerts: map[string]*resources.Alert{
					"alert1": {
						Alert: alerts.Alert{
							DisplayName: new("alert1"),
							Schedule: &alerts.CronSchedule{
								QuartzCronSchedule: new("0 0 12 * * ?"),
							},
						},
					},
					"alert2": {
						Alert: alerts.Alert{
							DisplayName: new("alert2"),
							Schedule: &alerts.CronSchedule{
								QuartzCronSchedule: new("0 0 12 * * ?"),
								PauseStatus:        alerts.SchedulePauseStatus_Unpaused,
							},
						},
					},
					"alert3": {
						Alert: alerts.Alert{
							DisplayName: new("alert3"),
						},
					},
				},
				PostgresProjects: map[string]*resources.PostgresProject{
					"postgres_project1": {
						PostgresProjectConfig: resources.PostgresProjectConfig{
							ProjectId: "postgres-project-1",
							ProjectSpec: postgres.ProjectSpec{
								DisplayName: new("postgres_project1"),
							},
						},
					},
				},
				PostgresBranches: map[string]*resources.PostgresBranch{
					"postgres_branch1": {
						PostgresBranchConfig: resources.PostgresBranchConfig{
							BranchId: "postgres-branch-1",
							Parent:   "projects/postgres-project-1",
						},
					},
				},
				PostgresEndpoints: map[string]*resources.PostgresEndpoint{
					"postgres_endpoint1": {
						PostgresEndpointConfig: resources.PostgresEndpointConfig{
							EndpointId: "postgres-endpoint-1",
							Parent:     "projects/postgres-project-1/branches/postgres-branch-1",
							EndpointSpec: postgres.EndpointSpec{
								EndpointType: postgres.EndpointType_EndpointTypeReadWrite,
							},
						},
					},
				},
				PostgresCatalogs: map[string]*resources.PostgresCatalog{
					"postgres_catalog1": {
						PostgresCatalogConfig: resources.PostgresCatalogConfig{
							CatalogId: "postgres_catalog_1",
						},
					},
				},
				PostgresDatabases: map[string]*resources.PostgresDatabase{
					"postgres_database1": {
						PostgresDatabaseConfig: resources.PostgresDatabaseConfig{
							DatabaseId: "postgres-database-1",
							Parent:     "projects/postgres-project-1/branches/postgres-branch-1",
						},
					},
				},
				PostgresRoles: map[string]*resources.PostgresRole{
					"postgres_role1": {
						PostgresRoleConfig: resources.PostgresRoleConfig{
							RoleId: "postgres-role-1",
							Parent: "projects/postgres-project-1/branches/postgres-branch-1",
							Role_RoleSpec: postgres.Role_RoleSpec{
								PostgresRole: new("postgres_role_1"),
							},
						},
					},
				},
				PostgresSyncedTables: map[string]*resources.PostgresSyncedTable{
					"postgres_synced_table1": {
						PostgresSyncedTableConfig: resources.PostgresSyncedTableConfig{
							SyncedTableId: "catalog.schema.table1",
						},
					},
				},
				PostgresSnapshotSchedules: map[string]*resources.PostgresSnapshotSchedule{
					"postgres_snapshot_schedule1": {
						PostgresSnapshotScheduleConfig: resources.PostgresSnapshotScheduleConfig{
							Branch: "projects/project1/branches/branch1",
						},
					},
				},
				VectorSearchEndpoints: map[string]*resources.VectorSearchEndpoint{
					"vs_endpoint1": {
						CreateEndpointRequest: vectorsearch.CreateEndpointRequest{
							Name:         new("vs_endpoint1"),
							EndpointType: vectorsearch.EndpointType_Standard,
						},
					},
				},
				VectorSearchIndexes: map[string]*resources.VectorSearchIndex{
					"vs_index1": {
						CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{
							Name:         new("main.default.vs_index1"),
							EndpointName: new("vs_endpoint1"),
							PrimaryKey:   new("id"),
							IndexType:    vectorsearch.VectorIndexType("DELTA_SYNC"),
						},
					},
				},
				Snapshots: map[string]*resources.Snapshot{
					"snapshot1": {
						BundleID: "bundle1",
						ACL:      nil,
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

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	// Job 1
	assert.Equal(t, "[dev lennart] job1", utils.Deref(b.Config.Resources.Jobs["job1"].Name))
	assert.Equal(t, "tag", b.Config.Resources.Jobs["job1"].Tags["existing"])
	assert.Equal(t, "lennart", b.Config.Resources.Jobs["job1"].Tags["dev"])
	assert.Equal(t, jobs.SchedulePauseStatus_Paused, b.Config.Resources.Jobs["job1"].Schedule.PauseStatus)

	// Job 2
	assert.Equal(t, "[dev lennart] job2", utils.Deref(b.Config.Resources.Jobs["job2"].Name))
	assert.Equal(t, "lennart", b.Config.Resources.Jobs["job2"].Tags["dev"])
	assert.Equal(t, jobs.SchedulePauseStatus("UNPAUSED"), b.Config.Resources.Jobs["job2"].Schedule.PauseStatus)

	// Pipeline 1
	assert.Equal(t, "[dev lennart] pipeline1", utils.Deref(b.Config.Resources.Pipelines["pipeline1"].Name))
	assert.False(t, utils.Deref(b.Config.Resources.Pipelines["pipeline1"].Continuous)) //nolint:staticcheck // SA1019: pipeline continuous is deprecated in the SDK but remains a supported bundle config field
	assert.True(t, utils.Deref(b.Config.Resources.Pipelines["pipeline1"].Development)) //nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field

	// Experiment 1
	assert.Equal(t, "/Users/lennart.kats@databricks.com/[dev lennart] experiment1", utils.Deref(b.Config.Resources.Experiments["experiment1"].Name))
	assert.Contains(t, b.Config.Resources.Experiments["experiment1"].Tags, experiments.ExperimentTag{Key: new("dev"), Value: new("lennart")})
	assert.Equal(t, "dev", utils.Deref(b.Config.Resources.Experiments["experiment1"].CreateExperimentRequest.Tags[0].Key))

	// Experiment 2
	assert.Equal(t, "[dev lennart] experiment2", utils.Deref(b.Config.Resources.Experiments["experiment2"].Name))
	assert.Contains(t, b.Config.Resources.Experiments["experiment2"].Tags, experiments.ExperimentTag{Key: new("dev"), Value: new("lennart")})

	// Model 1
	assert.Equal(t, "[dev lennart] model1", utils.Deref(b.Config.Resources.Models["model1"].Name))
	assert.Contains(t, b.Config.Resources.Models["model1"].Tags, modelregistry.RegisteredModelTag{Key: new("dev"), Value: new("lennart")})

	// Model serving endpoint 1
	assert.Equal(t, "dev_lennart_servingendpoint1", utils.Deref(b.Config.Resources.ModelServingEndpoints["servingendpoint1"].Name))

	// Vector search endpoint 1: name is the primary key, so it must not be prefixed.
	assert.Equal(t, "vs_endpoint1", utils.Deref(b.Config.Resources.VectorSearchEndpoints["vs_endpoint1"].Name))

	// Vector search index 1: name is the primary key, so it must not be prefixed.
	assert.Equal(t, "main.default.vs_index1", utils.Deref(b.Config.Resources.VectorSearchIndexes["vs_index1"].Name))

	// Registered model 1
	assert.Equal(t, "dev_lennart_registeredmodel1", utils.Deref(b.Config.Resources.RegisteredModels["registeredmodel1"].Name))

	// Quality Monitor 1
	assert.Equal(t, "qualityMonitor1", b.Config.Resources.QualityMonitors["qualityMonitor1"].TableName)
	assert.Nil(t, b.Config.Resources.QualityMonitors["qualityMonitor2"].Schedule)
	assert.Equal(t, qualitymonitors.MonitorCronSchedulePauseStatusUnpaused, b.Config.Resources.QualityMonitors["qualityMonitor3"].Schedule.PauseStatus)

	// Schema 1
	assert.Equal(t, "dev_lennart_schema1", utils.Deref(b.Config.Resources.Schemas["schema1"].Name))

	// Clusters
	assert.Equal(t, "[dev lennart] cluster1", utils.Deref(b.Config.Resources.Clusters["cluster1"].ClusterName))

	// Instance pools
	assert.Equal(t, "[dev lennart] instance_pool1", utils.Deref(b.Config.Resources.InstancePools["instance_pool1"].InstancePoolName))

	// Dashboards
	assert.Equal(t, "[dev lennart] dashboard1", b.Config.Resources.Dashboards["dashboard1"].DisplayName)

	// Genie Spaces
	assert.Equal(t, "[dev lennart] geniespace1", b.Config.Resources.GenieSpaces["geniespace1"].Title)

	// Alert 1: has schedule without pause status set - should be paused
	assert.Equal(t, "[dev lennart] alert1", utils.Deref(b.Config.Resources.Alerts["alert1"].DisplayName))
	assert.Equal(t, alerts.SchedulePauseStatus_Paused, b.Config.Resources.Alerts["alert1"].Schedule.PauseStatus)

	// Alert 2: has schedule with pause status already set to unpaused - should remain unpaused
	assert.Equal(t, "[dev lennart] alert2", utils.Deref(b.Config.Resources.Alerts["alert2"].DisplayName))
	assert.Equal(t, alerts.SchedulePauseStatus_Unpaused, b.Config.Resources.Alerts["alert2"].Schedule.PauseStatus)

	// Alert 3: no schedule - pause status should remain empty
	assert.Equal(t, "[dev lennart] alert3", utils.Deref(b.Config.Resources.Alerts["alert3"].DisplayName))
	assert.Nil(t, b.Config.Resources.Alerts["alert3"].Schedule)
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
	assert.Equal(t, "job1", utils.Deref(b.Config.Resources.Jobs["job1"].Name))
	assert.Equal(t, "pipeline1", utils.Deref(b.Config.Resources.Pipelines["pipeline1"].Name))
	assert.False(t, utils.Deref(b.Config.Resources.Pipelines["pipeline1"].Development)) //nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
	assert.Equal(t, "servingendpoint1", utils.Deref(b.Config.Resources.ModelServingEndpoints["servingendpoint1"].Name))
	assert.Equal(t, "registeredmodel1", utils.Deref(b.Config.Resources.RegisteredModels["registeredmodel1"].Name))
	assert.Equal(t, "qualityMonitor1", b.Config.Resources.QualityMonitors["qualityMonitor1"].TableName)
	assert.Equal(t, "catalog1", utils.Deref(b.Config.Resources.Catalogs["catalog1"].Name))
	assert.Equal(t, "schema1", utils.Deref(b.Config.Resources.Schemas["schema1"].Name))
	assert.Equal(t, "volume1", utils.Deref(b.Config.Resources.Volumes["volume1"].Name))
	assert.Equal(t, "cluster1", utils.Deref(b.Config.Resources.Clusters["cluster1"].ClusterName))
	assert.Equal(t, "instance_pool1", utils.Deref(b.Config.Resources.InstancePools["instance_pool1"].InstancePoolName))
	assert.Equal(t, "sql_warehouse1", utils.Deref(b.Config.Resources.SqlWarehouses["sql_warehouse1"].Name))
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
				"apply_target_mode should support '%s' (please add it to apply_target_mode.go and extend the test suite)",
				resources.Type().Field(i).Name,
			)
		}
	}
}

// TestAppropriateResourcesAreRenamed checks that every resource with a user-facing
// Name field is renamed by dev-mode / presets.name_prefix, except for an
// explicit carve-out list. The carve-out applies to resources whose Name is
// the API primary key / object id (not a display name) — prefixing those
// would change the resource's identity rather than its label.
func TestAppropriateResourcesAreRenamed(t *testing.T) {
	b := mockBundle(config.Development)

	notRenamedFields := []reflect.Type{
		reflect.TypeFor[*resources.Catalog](),
		reflect.TypeFor[*resources.ExternalLocation](),
		reflect.TypeFor[*resources.Volume](),
		reflect.TypeFor[*resources.VectorSearchEndpoint](),
		reflect.TypeFor[*resources.VectorSearchIndex](),
	}

	// Resources whose Name is server-generated or otherwise not a user-facing
	// label, so the rename matrix doesn't apply. Reflection still finds a
	// Name field on these via embedded SDK types, hence the explicit skip.
	notUserNamed := []string{
		"Apps",
		"SecretScopes",
		"Secrets",
		"DatabaseInstances",
		"DatabaseCatalogs",
		"SyncedDatabaseTables",
		"PostgresProjects",
		"PostgresBranches",
		"PostgresEndpoints",
		"PostgresCatalogs",
		"PostgresDatabases",
		"PostgresSyncedTables",
	}

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	resources := reflect.ValueOf(b.Config.Resources)
	for i := range resources.NumField() {
		field := resources.Field(i)
		if field.Kind() != reflect.Map {
			continue
		}
		resourceType := resources.Type().Field(i).Name
		if slices.Contains(notUserNamed, resourceType) {
			continue
		}
		for _, key := range field.MapKeys() {
			resource := field.MapIndex(key)
			nameField := resource.Elem().FieldByName("Name")
			if !nameField.IsValid() || nameField.Kind() != reflect.String {
				continue
			}

			if slices.Contains(notRenamedFields, resource.Type()) {
				assert.NotContains(t, nameField.String(), "dev", "process_target_mode should not rename '%s' in '%s'", key, resourceType)
			} else {
				assert.Contains(t, nameField.String(), "dev", "process_target_mode should rename '%s' in '%s'", key, resourceType)
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

	assert.Equal(t, "custom_lennart_deploy_job1", utils.Deref(b.Config.Resources.Jobs["job1"].Name))
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

	assert.Equal(t, 10, utils.Deref(b.Config.Resources.Jobs["job1"].MaxConcurrentRuns))
}

func TestJobsMaxConcurrentRunsDisabled(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.JobsMaxConcurrentRuns = 1

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	assert.Equal(t, 1, utils.Deref(b.Config.Resources.Jobs["job1"].MaxConcurrentRuns))
}

func TestPipelinesDevelopmentDisabled(t *testing.T) {
	b := mockBundle(config.Development)
	notEnabled := false
	b.Config.Presets.PipelinesDevelopment = &notEnabled

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	assert.False(t, utils.Deref(b.Config.Resources.Pipelines["pipeline1"].Development)) //nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
}
