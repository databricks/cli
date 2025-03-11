package terraform

import (
	"context"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func produceTerraformConfiguration(t *testing.T, config config.Root) *schema.Root {
	vin, err := convert.FromTyped(config, dyn.NilValue)
	require.NoError(t, err)
	out, err := BundleToTerraformWithDynValue(context.Background(), vin)
	require.NoError(t, err)
	return out
}

func convertToResourceStruct[T any](t *testing.T, resource *T, data any) {
	require.NotNil(t, resource)
	require.NotNil(t, data)

	// Convert data to a dyn.Value.
	vin, err := convert.FromTyped(data, dyn.NilValue)
	require.NoError(t, err)

	// Convert the dyn.Value to a struct.
	err = convert.ToTyped(resource, vin)
	require.NoError(t, err)
}

func TestBundleToTerraformJob(t *testing.T) {
	src := resources.Job{
		JobSettings: &jobs.JobSettings{
			Name: "my job",
			JobClusters: []jobs.JobCluster{
				{
					JobClusterKey: "key",
					NewCluster: compute.ClusterSpec{
						SparkVersion: "10.4.x-scala2.12",
					},
				},
			},
			GitSource: &jobs.GitSource{
				GitProvider: jobs.GitProviderGitHub,
				GitUrl:      "https://github.com/foo/bar",
			},
			Parameters: []jobs.JobParameterDefinition{
				{
					Name:    "param1",
					Default: "default1",
				},
				{
					Name:    "param2",
					Default: "default2",
				},
			},
		},
	}

	config := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job": &src,
			},
		},
	}

	var resource schema.ResourceJob
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.Job["my_job"])

	assert.Equal(t, "my job", resource.Name)
	assert.Len(t, resource.JobCluster, 1)
	assert.Equal(t, "https://github.com/foo/bar", resource.GitSource.Url)
	assert.Len(t, resource.Parameter, 2)
	assert.Equal(t, "param1", resource.Parameter[0].Name)
	assert.Equal(t, "param2", resource.Parameter[1].Name)
	assert.Nil(t, out.Data)
}

func TestBundleToTerraformJobPermissions(t *testing.T) {
	src := resources.Job{
		Permissions: []resources.Permission{
			{
				Level:    "CAN_VIEW",
				UserName: "jane@doe.com",
			},
		},
	}

	config := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job": &src,
			},
		},
	}

	var resource schema.ResourcePermissions
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.Permissions["job_my_job"])

	assert.NotEmpty(t, resource.JobId)
	assert.Len(t, resource.AccessControl, 1)
	assert.Equal(t, "jane@doe.com", resource.AccessControl[0].UserName)
	assert.Equal(t, "CAN_VIEW", resource.AccessControl[0].PermissionLevel)
}

func TestBundleToTerraformJobTaskLibraries(t *testing.T) {
	src := resources.Job{
		JobSettings: &jobs.JobSettings{
			Name: "my job",
			Tasks: []jobs.Task{
				{
					TaskKey: "key",
					Libraries: []compute.Library{
						{
							Pypi: &compute.PythonPyPiLibrary{
								Package: "mlflow",
							},
						},
					},
				},
			},
		},
	}

	config := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job": &src,
			},
		},
	}

	var resource schema.ResourceJob
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.Job["my_job"])

	assert.Equal(t, "my job", resource.Name)
	require.Len(t, resource.Task, 1)
	require.Len(t, resource.Task[0].Library, 1)
	assert.Equal(t, "mlflow", resource.Task[0].Library[0].Pypi.Package)
}

func TestBundleToTerraformForEachTaskLibraries(t *testing.T) {
	src := resources.Job{
		JobSettings: &jobs.JobSettings{
			Name: "my job",
			Tasks: []jobs.Task{
				{
					TaskKey: "key",
					ForEachTask: &jobs.ForEachTask{
						Inputs: "[1,2,3]",
						Task: jobs.Task{
							TaskKey: "iteration",
							Libraries: []compute.Library{
								{
									Pypi: &compute.PythonPyPiLibrary{
										Package: "mlflow",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	config := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job": &src,
			},
		},
	}

	var resource schema.ResourceJob
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.Job["my_job"])

	assert.Equal(t, "my job", resource.Name)
	require.Len(t, resource.Task, 1)
	require.Len(t, resource.Task[0].ForEachTask.Task.Library, 1)
	assert.Equal(t, "mlflow", resource.Task[0].ForEachTask.Task.Library[0].Pypi.Package)
}

func TestBundleToTerraformPipeline(t *testing.T) {
	src := resources.Pipeline{
		CreatePipeline: &pipelines.CreatePipeline{
			Name: "my pipeline",
			Libraries: []pipelines.PipelineLibrary{
				{
					Notebook: &pipelines.NotebookLibrary{
						Path: "notebook path",
					},
				},
				{
					File: &pipelines.FileLibrary{
						Path: "file path",
					},
				},
			},
			Notifications: []pipelines.Notifications{
				{
					Alerts: []string{
						"on-update-fatal-failure",
					},
					EmailRecipients: []string{
						"jane@doe.com",
					},
				},
				{
					Alerts: []string{
						"on-update-failure",
						"on-flow-failure",
					},
					EmailRecipients: []string{
						"jane@doe.com",
						"john@doe.com",
					},
				},
			},
		},
	}

	config := config.Root{
		Resources: config.Resources{
			Pipelines: map[string]*resources.Pipeline{
				"my_pipeline": &src,
			},
		},
	}

	var resource schema.ResourcePipeline
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.Pipeline["my_pipeline"])

	assert.Equal(t, "my pipeline", resource.Name)
	assert.Len(t, resource.Library, 2)
	assert.Len(t, resource.Notification, 2)
	assert.Equal(t, []string{"on-update-fatal-failure"}, resource.Notification[0].Alerts)
	assert.Equal(t, []string{"jane@doe.com"}, resource.Notification[0].EmailRecipients)
	assert.Equal(t, []string{"on-update-failure", "on-flow-failure"}, resource.Notification[1].Alerts)
	assert.Equal(t, []string{"jane@doe.com", "john@doe.com"}, resource.Notification[1].EmailRecipients)
	assert.Nil(t, out.Data)
}

func TestBundleToTerraformPipelinePermissions(t *testing.T) {
	src := resources.Pipeline{
		Permissions: []resources.Permission{
			{
				Level:    "CAN_VIEW",
				UserName: "jane@doe.com",
			},
		},
	}

	config := config.Root{
		Resources: config.Resources{
			Pipelines: map[string]*resources.Pipeline{
				"my_pipeline": &src,
			},
		},
	}

	var resource schema.ResourcePermissions
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.Permissions["pipeline_my_pipeline"])

	assert.NotEmpty(t, resource.PipelineId)
	assert.Len(t, resource.AccessControl, 1)
	assert.Equal(t, "jane@doe.com", resource.AccessControl[0].UserName)
	assert.Equal(t, "CAN_VIEW", resource.AccessControl[0].PermissionLevel)
}

func TestBundleToTerraformModel(t *testing.T) {
	src := resources.MlflowModel{
		Model: &ml.Model{
			Name:        "name",
			Description: "description",
			Tags: []ml.ModelTag{
				{
					Key:   "k1",
					Value: "v1",
				},
				{
					Key:   "k2",
					Value: "v2",
				},
			},
		},
	}

	config := config.Root{
		Resources: config.Resources{
			Models: map[string]*resources.MlflowModel{
				"my_model": &src,
			},
		},
	}

	var resource schema.ResourceMlflowModel
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.MlflowModel["my_model"])

	assert.Equal(t, "name", resource.Name)
	assert.Equal(t, "description", resource.Description)
	assert.Len(t, resource.Tags, 2)
	assert.Equal(t, "k1", resource.Tags[0].Key)
	assert.Equal(t, "v1", resource.Tags[0].Value)
	assert.Equal(t, "k2", resource.Tags[1].Key)
	assert.Equal(t, "v2", resource.Tags[1].Value)
	assert.Nil(t, out.Data)
}

func TestBundleToTerraformModelPermissions(t *testing.T) {
	src := resources.MlflowModel{
		Model: &ml.Model{
			Name: "name",
		},
		Permissions: []resources.Permission{
			{
				Level:    "CAN_READ",
				UserName: "jane@doe.com",
			},
		},
	}

	config := config.Root{
		Resources: config.Resources{
			Models: map[string]*resources.MlflowModel{
				"my_model": &src,
			},
		},
	}

	var resource schema.ResourcePermissions
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.Permissions["mlflow_model_my_model"])

	assert.NotEmpty(t, resource.RegisteredModelId)
	assert.Len(t, resource.AccessControl, 1)
	assert.Equal(t, "jane@doe.com", resource.AccessControl[0].UserName)
	assert.Equal(t, "CAN_READ", resource.AccessControl[0].PermissionLevel)
}

func TestBundleToTerraformExperiment(t *testing.T) {
	src := resources.MlflowExperiment{
		Experiment: &ml.Experiment{
			Name: "name",
		},
	}

	config := config.Root{
		Resources: config.Resources{
			Experiments: map[string]*resources.MlflowExperiment{
				"my_experiment": &src,
			},
		},
	}

	var resource schema.ResourceMlflowExperiment
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.MlflowExperiment["my_experiment"])

	assert.Equal(t, "name", resource.Name)
	assert.Nil(t, out.Data)
}

func TestBundleToTerraformExperimentPermissions(t *testing.T) {
	src := resources.MlflowExperiment{
		Experiment: &ml.Experiment{
			Name: "name",
		},
		Permissions: []resources.Permission{
			{
				Level:    "CAN_READ",
				UserName: "jane@doe.com",
			},
		},
	}

	config := config.Root{
		Resources: config.Resources{
			Experiments: map[string]*resources.MlflowExperiment{
				"my_experiment": &src,
			},
		},
	}

	var resource schema.ResourcePermissions
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.Permissions["mlflow_experiment_my_experiment"])

	assert.NotEmpty(t, resource.ExperimentId)
	assert.Len(t, resource.AccessControl, 1)
	assert.Equal(t, "jane@doe.com", resource.AccessControl[0].UserName)
	assert.Equal(t, "CAN_READ", resource.AccessControl[0].PermissionLevel)
}

func TestBundleToTerraformModelServing(t *testing.T) {
	src := resources.ModelServingEndpoint{
		CreateServingEndpoint: &serving.CreateServingEndpoint{
			Name: "name",
			Config: &serving.EndpointCoreConfigInput{
				ServedModels: []serving.ServedModelInput{
					{
						ModelName:          "model_name",
						ModelVersion:       "1",
						ScaleToZeroEnabled: true,
						WorkloadSize:       "Small",
					},
				},
				TrafficConfig: &serving.TrafficConfig{
					Routes: []serving.Route{
						{
							ServedModelName:   "model_name-1",
							TrafficPercentage: 100,
						},
					},
				},
			},
		},
	}

	config := config.Root{
		Resources: config.Resources{
			ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
				"my_model_serving_endpoint": &src,
			},
		},
	}

	var resource schema.ResourceModelServing
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.ModelServing["my_model_serving_endpoint"])

	assert.Equal(t, "name", resource.Name)
	assert.Equal(t, "model_name", resource.Config.ServedModels[0].ModelName)
	assert.Equal(t, "1", resource.Config.ServedModels[0].ModelVersion)
	assert.True(t, resource.Config.ServedModels[0].ScaleToZeroEnabled)
	assert.Equal(t, "Small", resource.Config.ServedModels[0].WorkloadSize)
	assert.Equal(t, "model_name-1", resource.Config.TrafficConfig.Routes[0].ServedModelName)
	assert.Equal(t, 100, resource.Config.TrafficConfig.Routes[0].TrafficPercentage)
	assert.Nil(t, out.Data)
}

func TestBundleToTerraformModelServingPermissions(t *testing.T) {
	src := resources.ModelServingEndpoint{
		CreateServingEndpoint: &serving.CreateServingEndpoint{
			Name: "name",

			// Need to specify this to satisfy the equivalence test:
			// The previous method of generation includes the "create" field
			// because it is required (not marked as `omitempty`).
			// The previous method used [json.Marshal] from the standard library
			// and as such observed the `omitempty` tag.
			// The new method leverages [dyn.Value] where any field that is not
			// explicitly set is not part of the value.
			Config: &serving.EndpointCoreConfigInput{
				ServedModels: []serving.ServedModelInput{
					{
						ModelName:          "model_name",
						ModelVersion:       "1",
						ScaleToZeroEnabled: true,
						WorkloadSize:       "Small",
					},
				},
			},
		},
		Permissions: []resources.Permission{
			{
				Level:    "CAN_VIEW",
				UserName: "jane@doe.com",
			},
		},
	}

	config := config.Root{
		Resources: config.Resources{
			ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
				"my_model_serving_endpoint": &src,
			},
		},
	}

	var resource schema.ResourcePermissions
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.Permissions["model_serving_my_model_serving_endpoint"])

	assert.NotEmpty(t, resource.ServingEndpointId)
	assert.Len(t, resource.AccessControl, 1)
	assert.Equal(t, "jane@doe.com", resource.AccessControl[0].UserName)
	assert.Equal(t, "CAN_VIEW", resource.AccessControl[0].PermissionLevel)
}

func TestBundleToTerraformRegisteredModel(t *testing.T) {
	src := resources.RegisteredModel{
		CreateRegisteredModelRequest: &catalog.CreateRegisteredModelRequest{
			Name:        "name",
			CatalogName: "catalog",
			SchemaName:  "schema",
			Comment:     "comment",
		},
	}

	config := config.Root{
		Resources: config.Resources{
			RegisteredModels: map[string]*resources.RegisteredModel{
				"my_registered_model": &src,
			},
		},
	}

	var resource schema.ResourceRegisteredModel
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.RegisteredModel["my_registered_model"])

	assert.Equal(t, "name", resource.Name)
	assert.Equal(t, "catalog", resource.CatalogName)
	assert.Equal(t, "schema", resource.SchemaName)
	assert.Equal(t, "comment", resource.Comment)
	assert.Nil(t, out.Data)
}

func TestBundleToTerraformRegisteredModelGrants(t *testing.T) {
	src := resources.RegisteredModel{
		CreateRegisteredModelRequest: &catalog.CreateRegisteredModelRequest{
			Name:        "name",
			CatalogName: "catalog",
			SchemaName:  "schema",
		},
		Grants: []resources.Grant{
			{
				Privileges: []string{"EXECUTE"},
				Principal:  "jane@doe.com",
			},
		},
	}

	config := config.Root{
		Resources: config.Resources{
			RegisteredModels: map[string]*resources.RegisteredModel{
				"my_registered_model": &src,
			},
		},
	}

	var resource schema.ResourceGrants
	out := produceTerraformConfiguration(t, config)
	convertToResourceStruct(t, &resource, out.Resource.Grants["registered_model_my_registered_model"])

	assert.NotEmpty(t, resource.Function)
	assert.Len(t, resource.Grant, 1)
	assert.Equal(t, "jane@doe.com", resource.Grant[0].Principal)
	assert.Equal(t, "EXECUTE", resource.Grant[0].Privileges[0])
}

func TestBundleToTerraformDeletedResources(t *testing.T) {
	job1 := resources.Job{
		JobSettings: &jobs.JobSettings{},
	}
	job2 := resources.Job{
		ModifiedStatus: resources.ModifiedStatusDeleted,
		JobSettings:    &jobs.JobSettings{},
	}
	config := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job1": &job1,
				"my_job2": &job2,
			},
		},
	}

	vin, err := convert.FromTyped(config, dyn.NilValue)
	require.NoError(t, err)
	out, err := BundleToTerraformWithDynValue(context.Background(), vin)
	require.NoError(t, err)

	_, ok := out.Resource.Job["my_job1"]
	assert.True(t, ok)
	_, ok = out.Resource.Job["my_job2"]
	assert.False(t, ok)
}

func TestTerraformToBundleEmptyLocalResources(t *testing.T) {
	config := config.Root{
		Resources: config.Resources{},
	}
	tfState := resourcesState{
		Resources: []stateResource{
			{
				Type: "databricks_job",
				Mode: "managed",
				Name: "test_job",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_pipeline",
				Mode: "managed",
				Name: "test_pipeline",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_mlflow_model",
				Mode: "managed",
				Name: "test_mlflow_model",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_mlflow_experiment",
				Mode: "managed",
				Name: "test_mlflow_experiment",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_model_serving",
				Mode: "managed",
				Name: "test_model_serving",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_registered_model",
				Mode: "managed",
				Name: "test_registered_model",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_quality_monitor",
				Mode: "managed",
				Name: "test_monitor",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_schema",
				Mode: "managed",
				Name: "test_schema",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_volume",
				Mode: "managed",
				Name: "test_volume",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_cluster",
				Mode: "managed",
				Name: "test_cluster",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_dashboard",
				Mode: "managed",
				Name: "test_dashboard",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_app",
				Mode: "managed",
				Name: "test_app",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{Name: "app1"}},
				},
			},
		},
	}
	err := TerraformToBundle(&tfState, &config)
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

	assert.Equal(t, "app1", config.Resources.Apps["test_app"].Name)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Apps["test_app"].ModifiedStatus)

	AssertFullResourceCoverage(t, &config)
}

func TestTerraformToBundleEmptyRemoteResources(t *testing.T) {
	config := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"test_job": {
					JobSettings: &jobs.JobSettings{
						Name: "test_job",
					},
				},
			},
			Pipelines: map[string]*resources.Pipeline{
				"test_pipeline": {
					CreatePipeline: &pipelines.CreatePipeline{
						Name: "test_pipeline",
					},
				},
			},
			Models: map[string]*resources.MlflowModel{
				"test_mlflow_model": {
					Model: &ml.Model{
						Name: "test_mlflow_model",
					},
				},
			},
			Experiments: map[string]*resources.MlflowExperiment{
				"test_mlflow_experiment": {
					Experiment: &ml.Experiment{
						Name: "test_mlflow_experiment",
					},
				},
			},
			ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
				"test_model_serving": {
					CreateServingEndpoint: &serving.CreateServingEndpoint{
						Name: "test_model_serving",
					},
				},
			},
			RegisteredModels: map[string]*resources.RegisteredModel{
				"test_registered_model": {
					CreateRegisteredModelRequest: &catalog.CreateRegisteredModelRequest{
						Name: "test_registered_model",
					},
				},
			},
			QualityMonitors: map[string]*resources.QualityMonitor{
				"test_monitor": {
					CreateMonitor: &catalog.CreateMonitor{
						TableName: "test_monitor",
					},
				},
			},
			Schemas: map[string]*resources.Schema{
				"test_schema": {
					CreateSchema: &catalog.CreateSchema{
						Name: "test_schema",
					},
				},
			},
			Volumes: map[string]*resources.Volume{
				"test_volume": {
					CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
						Name: "test_volume",
					},
				},
			},
			Clusters: map[string]*resources.Cluster{
				"test_cluster": {
					ClusterSpec: &compute.ClusterSpec{
						ClusterName: "test_cluster",
					},
				},
			},
			Dashboards: map[string]*resources.Dashboard{
				"test_dashboard": {
					Dashboard: &dashboards.Dashboard{
						DisplayName: "test_dashboard",
					},
				},
			},
			Apps: map[string]*resources.App{
				"test_app": {
					App: &apps.App{
						Description: "test_app",
					},
				},
			},
		},
	}
	tfState := resourcesState{
		Resources: nil,
	}
	err := TerraformToBundle(&tfState, &config)
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

	assert.Equal(t, "", config.Resources.Apps["test_app"].Name)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Apps["test_app"].ModifiedStatus)

	AssertFullResourceCoverage(t, &config)
}

func TestTerraformToBundleModifiedResources(t *testing.T) {
	config := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"test_job": {
					JobSettings: &jobs.JobSettings{
						Name: "test_job",
					},
				},
				"test_job_new": {
					JobSettings: &jobs.JobSettings{
						Name: "test_job_new",
					},
				},
			},
			Pipelines: map[string]*resources.Pipeline{
				"test_pipeline": {
					CreatePipeline: &pipelines.CreatePipeline{
						Name: "test_pipeline",
					},
				},
				"test_pipeline_new": {
					CreatePipeline: &pipelines.CreatePipeline{
						Name: "test_pipeline_new",
					},
				},
			},
			Models: map[string]*resources.MlflowModel{
				"test_mlflow_model": {
					Model: &ml.Model{
						Name: "test_mlflow_model",
					},
				},
				"test_mlflow_model_new": {
					Model: &ml.Model{
						Name: "test_mlflow_model_new",
					},
				},
			},
			Experiments: map[string]*resources.MlflowExperiment{
				"test_mlflow_experiment": {
					Experiment: &ml.Experiment{
						Name: "test_mlflow_experiment",
					},
				},
				"test_mlflow_experiment_new": {
					Experiment: &ml.Experiment{
						Name: "test_mlflow_experiment_new",
					},
				},
			},
			ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
				"test_model_serving": {
					CreateServingEndpoint: &serving.CreateServingEndpoint{
						Name: "test_model_serving",
					},
				},
				"test_model_serving_new": {
					CreateServingEndpoint: &serving.CreateServingEndpoint{
						Name: "test_model_serving_new",
					},
				},
			},
			RegisteredModels: map[string]*resources.RegisteredModel{
				"test_registered_model": {
					CreateRegisteredModelRequest: &catalog.CreateRegisteredModelRequest{
						Name: "test_registered_model",
					},
				},
				"test_registered_model_new": {
					CreateRegisteredModelRequest: &catalog.CreateRegisteredModelRequest{
						Name: "test_registered_model_new",
					},
				},
			},
			QualityMonitors: map[string]*resources.QualityMonitor{
				"test_monitor": {
					CreateMonitor: &catalog.CreateMonitor{
						TableName: "test_monitor",
					},
				},
				"test_monitor_new": {
					CreateMonitor: &catalog.CreateMonitor{
						TableName: "test_monitor_new",
					},
				},
			},
			Schemas: map[string]*resources.Schema{
				"test_schema": {
					CreateSchema: &catalog.CreateSchema{
						Name: "test_schema",
					},
				},
				"test_schema_new": {
					CreateSchema: &catalog.CreateSchema{
						Name: "test_schema_new",
					},
				},
			},
			Volumes: map[string]*resources.Volume{
				"test_volume": {
					CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
						Name: "test_volume",
					},
				},
				"test_volume_new": {
					CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
						Name: "test_volume_new",
					},
				},
			},
			Clusters: map[string]*resources.Cluster{
				"test_cluster": {
					ClusterSpec: &compute.ClusterSpec{
						ClusterName: "test_cluster",
					},
				},
				"test_cluster_new": {
					ClusterSpec: &compute.ClusterSpec{
						ClusterName: "test_cluster_new",
					},
				},
			},
			Dashboards: map[string]*resources.Dashboard{
				"test_dashboard": {
					Dashboard: &dashboards.Dashboard{
						DisplayName: "test_dashboard",
					},
				},
				"test_dashboard_new": {
					Dashboard: &dashboards.Dashboard{
						DisplayName: "test_dashboard_new",
					},
				},
			},
			Apps: map[string]*resources.App{
				"test_app": {
					App: &apps.App{
						Name: "test_app",
					},
				},
				"test_app_new": {
					App: &apps.App{
						Name: "test_app_new",
					},
				},
			},
		},
	}
	tfState := resourcesState{
		Resources: []stateResource{
			{
				Type: "databricks_job",
				Mode: "managed",
				Name: "test_job",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_job",
				Mode: "managed",
				Name: "test_job_old",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "2"}},
				},
			},
			{
				Type: "databricks_pipeline",
				Mode: "managed",
				Name: "test_pipeline",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_pipeline",
				Mode: "managed",
				Name: "test_pipeline_old",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "2"}},
				},
			},
			{
				Type: "databricks_mlflow_model",
				Mode: "managed",
				Name: "test_mlflow_model",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_mlflow_model",
				Mode: "managed",
				Name: "test_mlflow_model_old",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "2"}},
				},
			},
			{
				Type: "databricks_mlflow_experiment",
				Mode: "managed",
				Name: "test_mlflow_experiment",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_mlflow_experiment",
				Mode: "managed",
				Name: "test_mlflow_experiment_old",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "2"}},
				},
			},
			{
				Type: "databricks_model_serving",
				Mode: "managed",
				Name: "test_model_serving",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_model_serving",
				Mode: "managed",
				Name: "test_model_serving_old",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "2"}},
				},
			},
			{
				Type: "databricks_registered_model",
				Mode: "managed",
				Name: "test_registered_model",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_registered_model",
				Mode: "managed",
				Name: "test_registered_model_old",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "2"}},
				},
			},
			{
				Type: "databricks_quality_monitor",
				Mode: "managed",
				Name: "test_monitor",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "test_monitor"}},
				},
			},
			{
				Type: "databricks_quality_monitor",
				Mode: "managed",
				Name: "test_monitor_old",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "test_monitor_old"}},
				},
			},
			{
				Type: "databricks_schema",
				Mode: "managed",
				Name: "test_schema",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_schema",
				Mode: "managed",
				Name: "test_schema_old",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "2"}},
				},
			},
			{
				Type: "databricks_volume",
				Mode: "managed",
				Name: "test_volume",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_volume",
				Mode: "managed",
				Name: "test_volume_old",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "2"}},
				},
			},
			{
				Type: "databricks_cluster",
				Mode: "managed",
				Name: "test_cluster",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_cluster",
				Mode: "managed",
				Name: "test_cluster_old",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "2"}},
				},
			},
			{
				Type: "databricks_dashboard",
				Mode: "managed",
				Name: "test_dashboard",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "1"}},
				},
			},
			{
				Type: "databricks_dashboard",
				Mode: "managed",
				Name: "test_dashboard_old",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "2"}},
				},
			},
			{
				Type: "databricks_app",
				Mode: "managed",
				Name: "test_app",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{Name: "test_app"}},
				},
			},
			{
				Type: "databricks_app",
				Mode: "managed",
				Name: "test_app_old",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{Name: "test_app_old"}},
				},
			},
		},
	}
	err := TerraformToBundle(&tfState, &config)
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

	assert.Equal(t, "test_app", config.Resources.Apps["test_app"].Name)
	assert.Equal(t, resources.ModifiedStatusUpdated, config.Resources.Apps["test_app"].ModifiedStatus)
	assert.Equal(t, "test_app_old", config.Resources.Apps["test_app_old"].Name)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Apps["test_app_old"].ModifiedStatus)
	assert.Equal(t, "test_app_new", config.Resources.Apps["test_app_new"].Name)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Apps["test_app_new"].ModifiedStatus)

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
				"TerraformToBundle should support '%s' (please add it to convert.go and extend the test suite)",
				resources.Type().Field(i).Name,
			)
		}
	}
}
