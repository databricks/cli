package resourcemutator

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/databricks/cli/bundle/permissions"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This list exists to ensure that this mutator is updated when new resource is added.
// These resources are there because they use grants, not permissions:
var unsupportedResources = []string{
	"catalogs",
	"volumes",
	"schemas",
	"quality_monitors",
	"registered_models",
	"database_catalogs",
	"synced_database_tables",
	"postgres_projects",
	"postgres_branches",
	"postgres_endpoints",
}

func TestApplyBundlePermissions(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/foo@bar.com",
			},
			Permissions: []resources.Permission{
				{Level: permissions.CAN_MANAGE, UserName: "TestUser"},
				{Level: permissions.CAN_VIEW, GroupName: "TestGroup"},
				{Level: permissions.CAN_RUN, ServicePrincipalName: "TestServicePrincipal"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {
						JobSettings: jobs.JobSettings{
							Name: "job_1",
						},
					},
					"job_2": {
						JobSettings: jobs.JobSettings{
							Name: "job_2",
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline_1": {},
					"pipeline_2": {},
				},
				Models: map[string]*resources.MlflowModel{
					"model_1": {},
					"model_2": {},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment_1": {},
					"experiment_2": {},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint_1": {},
					"endpoint_2": {},
				},
				Dashboards: map[string]*resources.Dashboard{
					"dashboard_1": {},
					"dashboard_2": {},
				},
				Apps: map[string]*resources.App{
					"app_1": {},
					"app_2": {},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, ApplyBundlePermissions())
	require.NoError(t, diags.Error())

	require.Len(t, b.Config.Resources.Jobs["job_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_MANAGE_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Jobs["job_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_MANAGE_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.PipelinePermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.PipelinePermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.PipelinePermission{Level: "CAN_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.PipelinePermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.PipelinePermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.PipelinePermission{Level: "CAN_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Models["model_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Models["model_1"].Permissions, resources.MlflowModelPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Models["model_1"].Permissions, resources.MlflowModelPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Models["model_2"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Models["model_2"].Permissions, resources.MlflowModelPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Models["model_2"].Permissions, resources.MlflowModelPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Experiments["experiment_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Experiments["experiment_1"].Permissions, resources.MlflowExperimentPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Experiments["experiment_1"].Permissions, resources.MlflowExperimentPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Experiments["experiment_2"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Experiments["experiment_2"].Permissions, resources.MlflowExperimentPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Experiments["experiment_2"].Permissions, resources.MlflowExperimentPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_QUERY", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_QUERY", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Dashboards["dashboard_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Dashboards["dashboard_1"].Permissions, resources.DashboardPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Dashboards["dashboard_1"].Permissions, resources.DashboardPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Apps["app_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Apps["app_1"].Permissions, resources.AppPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Apps["app_1"].Permissions, resources.AppPermission{Level: "CAN_USE", GroupName: "TestGroup"})
}

func TestWarningOnOverlapPermission(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/foo@bar.com",
			},
			Permissions: []resources.Permission{
				{Level: permissions.CAN_MANAGE, UserName: "TestUser"},
				{Level: permissions.CAN_VIEW, GroupName: "TestGroup"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {
						JobSettings: jobs.JobSettings{
							Name: "job_1",
						},
						Permissions: []resources.JobPermission{
							{Level: "CAN_VIEW", UserName: "TestUser"},
						},
					},
					"job_2": {
						JobSettings: jobs.JobSettings{
							Name: "job_2",
						},
						Permissions: []resources.JobPermission{
							{Level: "CAN_VIEW", UserName: "TestUser2"},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, ApplyBundlePermissions())
	require.NoError(t, diags.Error())

	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_VIEW", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_VIEW", UserName: "TestUser2"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
}

func TestAllResourcesExplicitlyDefinedForPermissionsSupport(t *testing.T) {
	full := resourcesWithOneOfEach()
	levelsMap := buildLevelsMap(full)

	for _, group := range full.AllResources() {
		name := group.Description.PluralName
		_, inLevels := levelsMap[name]
		isUnsupported := slices.Contains(unsupportedResources, name)

		if !inLevels && !isUnsupported {
			assert.Fail(t, fmt.Sprintf("Resource %s does not implement PermissionedResource and is not in unsupportedResources", name))
		}
		if inLevels && isUnsupported {
			assert.Fail(t, fmt.Sprintf("Resource %s implements PermissionedResource but is also in unsupportedResources", name))
		}
	}
}

func resourcesWithOneOfEach() *config.Resources {
	return &config.Resources{
		Jobs:                  map[string]*resources.Job{"j": {}},
		Pipelines:             map[string]*resources.Pipeline{"p": {}},
		Models:                map[string]*resources.MlflowModel{"m": {}},
		Experiments:           map[string]*resources.MlflowExperiment{"e": {}},
		ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{"mse": {}},
		RegisteredModels:      map[string]*resources.RegisteredModel{"rm": {}},
		QualityMonitors:       map[string]*resources.QualityMonitor{"qm": {}},
		Catalogs:              map[string]*resources.Catalog{"c": {}},
		Schemas:               map[string]*resources.Schema{"s": {}},
		Clusters:              map[string]*resources.Cluster{"cl": {}},
		Dashboards:            map[string]*resources.Dashboard{"d": {}},
		Volumes:               map[string]*resources.Volume{"v": {}},
		Apps:                  map[string]*resources.App{"a": {}},
		SecretScopes:          map[string]*resources.SecretScope{"ss": {}},
		Alerts:                map[string]*resources.Alert{"al": {}},
		SqlWarehouses:         map[string]*resources.SqlWarehouse{"sw": {}},
		DatabaseInstances:     map[string]*resources.DatabaseInstance{"di": {}},
		DatabaseCatalogs:      map[string]*resources.DatabaseCatalog{"dc": {}},
		SyncedDatabaseTables:  map[string]*resources.SyncedDatabaseTable{"sdt": {}},
		PostgresProjects:      map[string]*resources.PostgresProject{"pp": {}},
		PostgresBranches:      map[string]*resources.PostgresBranch{"pb": {}},
		PostgresEndpoints:     map[string]*resources.PostgresEndpoint{"pe": {}},
	}
}
