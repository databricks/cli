package statemgmt

import (
	"context"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
)

func TestStateToBundleEmptyLocalResources(t *testing.T) {
	config := config.Root{
		Resources: config.Resources{},
	}

	state := ExportedResourcesMap{
		"resources.jobs.test_job":                                     {ID: "1"},
		"resources.pipelines.test_pipeline":                           {ID: "1"},
		"resources.models.test_mlflow_model":                          {ID: "1"},
		"resources.experiments.test_mlflow_experiment":                {ID: "1"},
		"resources.model_serving_endpoints.test_model_serving":        {ID: "1"},
		"resources.registered_models.test_registered_model":           {ID: "1"},
		"resources.quality_monitors.test_monitor":                     {ID: "1"},
		"resources.schemas.test_schema":                               {ID: "1"},
		"resources.volumes.test_volume":                               {ID: "1"},
		"resources.clusters.test_cluster":                             {ID: "1"},
		"resources.dashboards.test_dashboard":                         {ID: "1"},
		"resources.genie_spaces.test_genie_space":                     {ID: "1"},
		"resources.apps.test_app":                                     {ID: "app1"},
		"resources.secret_scopes.test_secret_scope":                   {ID: "secret_scope1"},
		"resources.sql_warehouses.test_sql_warehouse":                 {ID: "1"},
		"resources.database_instances.test_database_instance":         {ID: "1"},
		"resources.database_catalogs.test_database_catalog":           {ID: "1"},
		"resources.synced_database_tables.test_synced_database_table": {ID: "1"},
		"resources.alerts.test_alert":                                 {ID: "1"},
	}
	err := StateToBundle(context.Background(), state, &config)
	assert.NoError(t, err)

	assert.Equal(t, "1", config.Resources.Jobs["test_job"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Jobs["test_job"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Pipelines["test_pipeline"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Pipelines["test_pipeline"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Models["test_mlflow_model"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Models["test_mlflow_model"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Experiments["test_mlflow_experiment"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Experiments["test_mlflow_experiment"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.ModelServingEndpoints["test_model_serving"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.ModelServingEndpoints["test_model_serving"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.RegisteredModels["test_registered_model"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.RegisteredModels["test_registered_model"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.QualityMonitors["test_monitor"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.QualityMonitors["test_monitor"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Schemas["test_schema"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Schemas["test_schema"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Volumes["test_volume"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Volumes["test_volume"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Clusters["test_cluster"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Clusters["test_cluster"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Dashboards["test_dashboard"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Dashboards["test_dashboard"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.GenieSpaces["test_genie_space"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.GenieSpaces["test_genie_space"].ModifiedStatus)

	assert.Equal(t, "app1", config.Resources.Apps["test_app"].ID)
	assert.Equal(t, "", config.Resources.Apps["test_app"].Name)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Apps["test_app"].ModifiedStatus)

	assert.Equal(t, "secret_scope1", config.Resources.SecretScopes["test_secret_scope"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.SecretScopes["test_secret_scope"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.SqlWarehouses["test_sql_warehouse"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.SqlWarehouses["test_sql_warehouse"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.DatabaseInstances["test_database_instance"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.DatabaseInstances["test_database_instance"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Alerts["test_alert"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Alerts["test_alert"].ModifiedStatus)

	AssertFullResourceCoverage(t, &config)
}

func TestStateToBundleEmptyRemoteResources(t *testing.T) {
	config := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"test_job": {
					JobSettings: jobs.JobSettings{
						Name: "test_job",
					},
				},
			},
			Pipelines: map[string]*resources.Pipeline{
				"test_pipeline": {
					CreatePipeline: pipelines.CreatePipeline{
						Name: "test_pipeline",
					},
				},
			},
			Models: map[string]*resources.MlflowModel{
				"test_mlflow_model": {
					CreateModelRequest: ml.CreateModelRequest{
						Name: "test_mlflow_model",
					},
				},
			},
			Experiments: map[string]*resources.MlflowExperiment{
				"test_mlflow_experiment": {
					CreateExperiment: ml.CreateExperiment{
						Name: "test_mlflow_experiment",
					},
				},
			},
			ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
				"test_model_serving": {
					CreateServingEndpoint: serving.CreateServingEndpoint{
						Name: "test_model_serving",
					},
				},
			},
			RegisteredModels: map[string]*resources.RegisteredModel{
				"test_registered_model": {
					CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{
						Name: "test_registered_model",
					},
				},
			},
			QualityMonitors: map[string]*resources.QualityMonitor{
				"test_monitor": {
					CreateMonitor: catalog.CreateMonitor{
						TableName: "test_monitor",
					},
				},
			},
			Schemas: map[string]*resources.Schema{
				"test_schema": {
					CreateSchema: catalog.CreateSchema{
						Name: "test_schema",
					},
				},
			},
			Volumes: map[string]*resources.Volume{
				"test_volume": {
					CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
						Name: "test_volume",
					},
				},
			},
			Clusters: map[string]*resources.Cluster{
				"test_cluster": {
					ClusterSpec: compute.ClusterSpec{
						ClusterName: "test_cluster",
					},
				},
			},
			Dashboards: map[string]*resources.Dashboard{
				"test_dashboard": {
					DashboardConfig: resources.DashboardConfig{
						DisplayName: "test_dashboard",
					},
				},
			},
			GenieSpaces: map[string]*resources.GenieSpace{
				"test_genie_space": {
					GenieSpaceConfig: resources.GenieSpaceConfig{
						Title: "test_genie_space",
					},
				},
			},
			Apps: map[string]*resources.App{
				"test_app": {
					App: apps.App{
						Description: "test_app",
					},
				},
			},
			SecretScopes: map[string]*resources.SecretScope{
				"test_secret_scope": {
					Name: "test_secret_scope",
				},
			},
			SqlWarehouses: map[string]*resources.SqlWarehouse{
				"test_sql_warehouse": {
					CreateWarehouseRequest: sql.CreateWarehouseRequest{
						Name: "test_sql_warehouse",
					},
				},
			},
			DatabaseInstances: map[string]*resources.DatabaseInstance{
				"test_database_instance": {
					DatabaseInstance: database.DatabaseInstance{
						Name: "test_database_instance",
					},
				},
			},
			DatabaseCatalogs: map[string]*resources.DatabaseCatalog{
				"test_database_catalog": {
					DatabaseCatalog: database.DatabaseCatalog{
						Name: "test_database_catalog",
					},
				},
			},
			SyncedDatabaseTables: map[string]*resources.SyncedDatabaseTable{
				"test_synced_database_table": {
					SyncedDatabaseTable: database.SyncedDatabaseTable{
						Name: "test_synced_database_table",
					},
				},
			},
			Alerts: map[string]*resources.Alert{
				"test_alert": {
					AlertV2: sql.AlertV2{
						DisplayName: "test_alert",
					},
				},
			},
		},
	}

	err := StateToBundle(context.Background(), nil, &config)
	assert.NoError(t, err)

	assert.Equal(t, "", config.Resources.Jobs["test_job"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Jobs["test_job"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.Pipelines["test_pipeline"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Pipelines["test_pipeline"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.Models["test_mlflow_model"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Models["test_mlflow_model"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.Experiments["test_mlflow_experiment"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Experiments["test_mlflow_experiment"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.ModelServingEndpoints["test_model_serving"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.ModelServingEndpoints["test_model_serving"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.RegisteredModels["test_registered_model"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.RegisteredModels["test_registered_model"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.QualityMonitors["test_monitor"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.QualityMonitors["test_monitor"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.Schemas["test_schema"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Schemas["test_schema"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.Volumes["test_volume"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Volumes["test_volume"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.Clusters["test_cluster"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Clusters["test_cluster"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.Dashboards["test_dashboard"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Dashboards["test_dashboard"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.GenieSpaces["test_genie_space"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.GenieSpaces["test_genie_space"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.Apps["test_app"].Name)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Apps["test_app"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.SecretScopes["test_secret_scope"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.SecretScopes["test_secret_scope"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.SqlWarehouses["test_sql_warehouse"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.SqlWarehouses["test_sql_warehouse"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.DatabaseInstances["test_database_instance"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.DatabaseInstances["test_database_instance"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.DatabaseCatalogs["test_database_catalog"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.DatabaseCatalogs["test_database_catalog"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.SyncedDatabaseTables["test_synced_database_table"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.SyncedDatabaseTables["test_synced_database_table"].ModifiedStatus)

	assert.Equal(t, "", config.Resources.Alerts["test_alert"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Alerts["test_alert"].ModifiedStatus)

	AssertFullResourceCoverage(t, &config)
}

func TestStateToBundleModifiedResources(t *testing.T) {
	config := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"test_job": {
					JobSettings: jobs.JobSettings{
						Name: "test_job",
					},
				},
				"test_job_new": {
					JobSettings: jobs.JobSettings{
						Name: "test_job_new",
					},
				},
			},
			Pipelines: map[string]*resources.Pipeline{
				"test_pipeline": {
					CreatePipeline: pipelines.CreatePipeline{
						Name: "test_pipeline",
					},
				},
				"test_pipeline_new": {
					CreatePipeline: pipelines.CreatePipeline{
						Name: "test_pipeline_new",
					},
				},
			},
			Models: map[string]*resources.MlflowModel{
				"test_mlflow_model": {
					CreateModelRequest: ml.CreateModelRequest{
						Name: "test_mlflow_model",
					},
				},
				"test_mlflow_model_new": {
					CreateModelRequest: ml.CreateModelRequest{
						Name: "test_mlflow_model_new",
					},
				},
			},
			Experiments: map[string]*resources.MlflowExperiment{
				"test_mlflow_experiment": {
					CreateExperiment: ml.CreateExperiment{
						Name: "test_mlflow_experiment",
					},
				},
				"test_mlflow_experiment_new": {
					CreateExperiment: ml.CreateExperiment{
						Name: "test_mlflow_experiment_new",
					},
				},
			},
			ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
				"test_model_serving": {
					CreateServingEndpoint: serving.CreateServingEndpoint{
						Name: "test_model_serving",
					},
				},
				"test_model_serving_new": {
					CreateServingEndpoint: serving.CreateServingEndpoint{
						Name: "test_model_serving_new",
					},
				},
			},
			RegisteredModels: map[string]*resources.RegisteredModel{
				"test_registered_model": {
					CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{
						Name: "test_registered_model",
					},
				},
				"test_registered_model_new": {
					CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{
						Name: "test_registered_model_new",
					},
				},
			},
			QualityMonitors: map[string]*resources.QualityMonitor{
				"test_monitor": {
					CreateMonitor: catalog.CreateMonitor{
						TableName: "test_monitor",
					},
				},
				"test_monitor_new": {
					CreateMonitor: catalog.CreateMonitor{
						TableName: "test_monitor_new",
					},
				},
			},
			Schemas: map[string]*resources.Schema{
				"test_schema": {
					CreateSchema: catalog.CreateSchema{
						Name: "test_schema",
					},
				},
				"test_schema_new": {
					CreateSchema: catalog.CreateSchema{
						Name: "test_schema_new",
					},
				},
			},
			Volumes: map[string]*resources.Volume{
				"test_volume": {
					CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
						Name: "test_volume",
					},
				},
				"test_volume_new": {
					CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
						Name: "test_volume_new",
					},
				},
			},
			Clusters: map[string]*resources.Cluster{
				"test_cluster": {
					ClusterSpec: compute.ClusterSpec{
						ClusterName: "test_cluster",
					},
				},
				"test_cluster_new": {
					ClusterSpec: compute.ClusterSpec{
						ClusterName: "test_cluster_new",
					},
				},
			},
			Dashboards: map[string]*resources.Dashboard{
				"test_dashboard": {
					DashboardConfig: resources.DashboardConfig{
						DisplayName: "test_dashboard",
					},
				},
				"test_dashboard_new": {
					DashboardConfig: resources.DashboardConfig{
						DisplayName: "test_dashboard_new",
					},
				},
			},
			GenieSpaces: map[string]*resources.GenieSpace{
				"test_genie_space": {
					GenieSpaceConfig: resources.GenieSpaceConfig{
						Title: "test_genie_space",
					},
				},
				"test_genie_space_new": {
					GenieSpaceConfig: resources.GenieSpaceConfig{
						Title: "test_genie_space_new",
					},
				},
			},
			Apps: map[string]*resources.App{
				"test_app": {
					App: apps.App{
						Name: "test_app",
					},
				},
				"test_app_new": {
					App: apps.App{
						Name: "test_app_new",
					},
				},
			},
			SecretScopes: map[string]*resources.SecretScope{
				"test_secret_scope": {
					Name: "test_secret_scope",
				},
				"test_secret_scope_new": {
					Name: "test_secret_scope_new",
				},
			},
			SqlWarehouses: map[string]*resources.SqlWarehouse{
				"test_sql_warehouse": {
					CreateWarehouseRequest: sql.CreateWarehouseRequest{
						Name: "test_sql_warehouse",
					},
				},
				"test_sql_warehouse_new": {
					CreateWarehouseRequest: sql.CreateWarehouseRequest{
						Name: "test_sql_warehouse_new",
					},
				},
			},
			DatabaseInstances: map[string]*resources.DatabaseInstance{
				"test_database_instance": {
					DatabaseInstance: database.DatabaseInstance{
						Name: "test_database_instance",
					},
				},
				"test_database_instance_new": {
					DatabaseInstance: database.DatabaseInstance{
						Name: "test_database_instance_new",
					},
				},
			},
			DatabaseCatalogs: map[string]*resources.DatabaseCatalog{
				"test_database_catalog": {
					DatabaseCatalog: database.DatabaseCatalog{
						Name: "test_database_catalog",
					},
				},
				"test_database_catalog_new": {
					DatabaseCatalog: database.DatabaseCatalog{
						Name: "test_database_catalog_new",
					},
				},
			},
			SyncedDatabaseTables: map[string]*resources.SyncedDatabaseTable{
				"test_synced_database_table": {
					SyncedDatabaseTable: database.SyncedDatabaseTable{
						Name: "test_synced_database_table",
					},
				},
				"test_synced_database_table_new": {
					SyncedDatabaseTable: database.SyncedDatabaseTable{
						Name: "test_synced_database_table_new",
					},
				},
			},
			Alerts: map[string]*resources.Alert{
				"test_alert": {
					AlertV2: sql.AlertV2{
						DisplayName: "test_alert",
					},
				},
				"test_alert_new": {
					AlertV2: sql.AlertV2{
						DisplayName: "test_alert_new",
					},
				},
			},
		},
	}
	state := ExportedResourcesMap{
		"resources.jobs.test_job":                                  {ID: "1"},
		"resources.jobs.test_job_old":                              {ID: "2"},
		"resources.pipelines.test_pipeline":                        {ID: "1"},
		"resources.pipelines.test_pipeline_old":                    {ID: "2"},
		"resources.models.test_mlflow_model":                       {ID: "1"},
		"resources.models.test_mlflow_model_old":                   {ID: "2"},
		"resources.experiments.test_mlflow_experiment":             {ID: "1"},
		"resources.experiments.test_mlflow_experiment_old":         {ID: "2"},
		"resources.model_serving_endpoints.test_model_serving":     {ID: "1"},
		"resources.model_serving_endpoints.test_model_serving_old": {ID: "2"},
		"resources.registered_models.test_registered_model":        {ID: "1"},
		"resources.registered_models.test_registered_model_old":    {ID: "2"},
		"resources.quality_monitors.test_monitor":                  {ID: "test_monitor"},
		"resources.quality_monitors.test_monitor_old":              {ID: "test_monitor_old"},
		"resources.schemas.test_schema":                            {ID: "1"},
		"resources.schemas.test_schema_old":                        {ID: "2"},
		"resources.volumes.test_volume":                            {ID: "1"},
		"resources.volumes.test_volume_old":                        {ID: "2"},
		"resources.clusters.test_cluster":                          {ID: "1"},
		"resources.clusters.test_cluster_old":                      {ID: "2"},
		"resources.dashboards.test_dashboard":                      {ID: "1"},
		"resources.dashboards.test_dashboard_old":                  {ID: "2"},
		"resources.genie_spaces.test_genie_space":                  {ID: "1"},
		"resources.genie_spaces.test_genie_space_old":              {ID: "2"},
		"resources.apps.test_app":                                  {ID: "test_app"},
		"resources.apps.test_app_old":                              {ID: "test_app_old"},
		"resources.secret_scopes.test_secret_scope":                {ID: "test_secret_scope"},
		"resources.secret_scopes.test_secret_scope_old":            {ID: "test_secret_scope_old"},
		"resources.sql_warehouses.test_sql_warehouse":              {ID: "1"},
		"resources.sql_warehouses.test_sql_warehouse_old":          {ID: "2"},
		"resources.database_instances.test_database_instance":      {ID: "1"},
		"resources.database_instances.test_database_instance_old":  {ID: "2"},
		"resources.alerts.test_alert":                              {ID: "1"},
		"resources.alerts.test_alert_old":                          {ID: "2"},
	}
	err := StateToBundle(context.Background(), state, &config)
	assert.NoError(t, err)

	assert.Equal(t, "1", config.Resources.Jobs["test_job"].ID)
	assert.Equal(t, "", config.Resources.Jobs["test_job"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.Jobs["test_job_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Jobs["test_job_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.Jobs["test_job_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Jobs["test_job_new"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Pipelines["test_pipeline"].ID)
	assert.Equal(t, "", config.Resources.Pipelines["test_pipeline"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.Pipelines["test_pipeline_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Pipelines["test_pipeline_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.Pipelines["test_pipeline_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Pipelines["test_pipeline_new"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Models["test_mlflow_model"].ID)
	assert.Equal(t, "", config.Resources.Models["test_mlflow_model"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.Models["test_mlflow_model_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Models["test_mlflow_model_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.Models["test_mlflow_model_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Models["test_mlflow_model_new"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.RegisteredModels["test_registered_model"].ID)
	assert.Equal(t, "", config.Resources.RegisteredModels["test_registered_model"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.RegisteredModels["test_registered_model_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.RegisteredModels["test_registered_model_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.RegisteredModels["test_registered_model_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.RegisteredModels["test_registered_model_new"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Experiments["test_mlflow_experiment"].ID)
	assert.Equal(t, "", config.Resources.Experiments["test_mlflow_experiment"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.Experiments["test_mlflow_experiment_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Experiments["test_mlflow_experiment_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.Experiments["test_mlflow_experiment_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Experiments["test_mlflow_experiment_new"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.ModelServingEndpoints["test_model_serving"].ID)
	assert.Equal(t, "", config.Resources.ModelServingEndpoints["test_model_serving"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.ModelServingEndpoints["test_model_serving_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.ModelServingEndpoints["test_model_serving_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.ModelServingEndpoints["test_model_serving_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.ModelServingEndpoints["test_model_serving_new"].ModifiedStatus)

	assert.Equal(t, "test_monitor", config.Resources.QualityMonitors["test_monitor"].ID)
	assert.Equal(t, "", config.Resources.QualityMonitors["test_monitor"].ModifiedStatus)
	assert.Equal(t, "test_monitor_old", config.Resources.QualityMonitors["test_monitor_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.QualityMonitors["test_monitor_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.QualityMonitors["test_monitor_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.QualityMonitors["test_monitor_new"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Schemas["test_schema"].ID)
	assert.Equal(t, "", config.Resources.Schemas["test_schema"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.Schemas["test_schema_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Schemas["test_schema_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.Schemas["test_schema_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Schemas["test_schema_new"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Volumes["test_volume"].ID)
	assert.Equal(t, "", config.Resources.Volumes["test_volume"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.Volumes["test_volume_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Volumes["test_volume_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.Volumes["test_volume_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Volumes["test_volume_new"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Clusters["test_cluster"].ID)
	assert.Equal(t, "", config.Resources.Clusters["test_cluster"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.Clusters["test_cluster_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Clusters["test_cluster_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.Clusters["test_cluster_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Clusters["test_cluster_new"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Dashboards["test_dashboard"].ID)
	assert.Equal(t, "", config.Resources.Dashboards["test_dashboard"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.Dashboards["test_dashboard_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Dashboards["test_dashboard_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.Dashboards["test_dashboard_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Dashboards["test_dashboard_new"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.GenieSpaces["test_genie_space"].ID)
	assert.Equal(t, "", config.Resources.GenieSpaces["test_genie_space"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.GenieSpaces["test_genie_space_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.GenieSpaces["test_genie_space_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.GenieSpaces["test_genie_space_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.GenieSpaces["test_genie_space_new"].ModifiedStatus)

	assert.Equal(t, "test_app", config.Resources.Apps["test_app"].Name)
	assert.Equal(t, "", config.Resources.Apps["test_app"].ModifiedStatus)
	assert.Equal(t, "test_app_old", config.Resources.Apps["test_app_old"].ID)
	assert.Equal(t, "", config.Resources.Apps["test_app_old"].Name)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Apps["test_app_old"].ModifiedStatus)
	assert.Equal(t, "test_app_new", config.Resources.Apps["test_app_new"].Name)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Apps["test_app_new"].ModifiedStatus)

	assert.Equal(t, "test_secret_scope", config.Resources.SecretScopes["test_secret_scope"].Name)
	assert.Equal(t, "", config.Resources.SecretScopes["test_secret_scope"].ModifiedStatus)
	assert.Equal(t, "test_secret_scope_old", config.Resources.SecretScopes["test_secret_scope_old"].ID)
	assert.Equal(t, "", config.Resources.SecretScopes["test_secret_scope_old"].Name)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.SecretScopes["test_secret_scope_old"].ModifiedStatus)
	assert.Equal(t, "test_secret_scope_new", config.Resources.SecretScopes["test_secret_scope_new"].Name)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.SecretScopes["test_secret_scope_new"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.SqlWarehouses["test_sql_warehouse"].ID)
	assert.Equal(t, "", config.Resources.SqlWarehouses["test_sql_warehouse"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.SqlWarehouses["test_sql_warehouse_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.SqlWarehouses["test_sql_warehouse_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.SqlWarehouses["test_sql_warehouse_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.SqlWarehouses["test_sql_warehouse_new"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.DatabaseInstances["test_database_instance"].ID)
	assert.Equal(t, "", config.Resources.DatabaseInstances["test_database_instance"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.DatabaseInstances["test_database_instance_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.DatabaseInstances["test_database_instance_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.DatabaseInstances["test_database_instance_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.DatabaseInstances["test_database_instance_new"].ModifiedStatus)

	assert.Equal(t, "1", config.Resources.Alerts["test_alert"].ID)
	assert.Equal(t, "", config.Resources.Alerts["test_alert"].ModifiedStatus)
	assert.Equal(t, "2", config.Resources.Alerts["test_alert_old"].ID)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Alerts["test_alert_old"].ModifiedStatus)
	assert.Equal(t, "", config.Resources.Alerts["test_alert_new"].ID)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Alerts["test_alert_new"].ModifiedStatus)

	AssertFullResourceCoverage(t, &config)
}

func AssertFullResourceCoverage(t *testing.T, config *config.Root) {
	resources := reflect.ValueOf(config.Resources)
	for i := range resources.NumField() {
		field := resources.Field(i)
		if field.Kind() == reflect.Map {
			assert.True(
				t,
				!field.IsNil() && field.Len() > 0,
				"StateToBundle should support '%s' (please add it to convert.go and extend the test suite)",
				resources.Type().Field(i).Name,
			)
		}
	}
}
