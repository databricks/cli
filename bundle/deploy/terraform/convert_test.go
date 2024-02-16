package terraform

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleToTerraformJob(t *testing.T) {
	var src = resources.Job{
		JobSettings: &jobs.JobSettings{
			Name: "my job",
			JobClusters: []jobs.JobCluster{
				{
					JobClusterKey: "key",
					NewCluster: &compute.ClusterSpec{
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

	var config = config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	resource := out.Resource.Job["my_job"].(*schema.ResourceJob)

	assert.Equal(t, "my job", resource.Name)
	assert.Len(t, resource.JobCluster, 1)
	assert.Equal(t, "https://github.com/foo/bar", resource.GitSource.Url)
	assert.Len(t, resource.Parameter, 2)
	assert.Equal(t, "param1", resource.Parameter[0].Name)
	assert.Equal(t, "param2", resource.Parameter[1].Name)
	assert.Nil(t, out.Data)
}

func TestBundleToTerraformJobPermissions(t *testing.T) {
	var src = resources.Job{
		Permissions: []resources.Permission{
			{
				Level:    "CAN_VIEW",
				UserName: "jane@doe.com",
			},
		},
	}

	var config = config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	resource := out.Resource.Permissions["job_my_job"].(*schema.ResourcePermissions)

	assert.NotEmpty(t, resource.JobId)
	assert.Len(t, resource.AccessControl, 1)
	assert.Equal(t, "jane@doe.com", resource.AccessControl[0].UserName)
	assert.Equal(t, "CAN_VIEW", resource.AccessControl[0].PermissionLevel)
}

func TestBundleToTerraformJobTaskLibraries(t *testing.T) {
	var src = resources.Job{
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

	var config = config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	resource := out.Resource.Job["my_job"].(*schema.ResourceJob)

	assert.Equal(t, "my job", resource.Name)
	require.Len(t, resource.Task, 1)
	require.Len(t, resource.Task[0].Library, 1)
	assert.Equal(t, "mlflow", resource.Task[0].Library[0].Pypi.Package)
}

func TestBundleToTerraformPipeline(t *testing.T) {
	var src = resources.Pipeline{
		PipelineSpec: &pipelines.PipelineSpec{
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

	var config = config.Root{
		Resources: config.Resources{
			Pipelines: map[string]*resources.Pipeline{
				"my_pipeline": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	resource := out.Resource.Pipeline["my_pipeline"].(*schema.ResourcePipeline)

	assert.Equal(t, "my pipeline", resource.Name)
	assert.Len(t, resource.Library, 2)
	assert.Len(t, resource.Notification, 2)
	assert.Equal(t, resource.Notification[0].Alerts, []string{"on-update-fatal-failure"})
	assert.Equal(t, resource.Notification[0].EmailRecipients, []string{"jane@doe.com"})
	assert.Equal(t, resource.Notification[1].Alerts, []string{"on-update-failure", "on-flow-failure"})
	assert.Equal(t, resource.Notification[1].EmailRecipients, []string{"jane@doe.com", "john@doe.com"})
	assert.Nil(t, out.Data)
}

func TestBundleToTerraformPipelinePermissions(t *testing.T) {
	var src = resources.Pipeline{
		Permissions: []resources.Permission{
			{
				Level:    "CAN_VIEW",
				UserName: "jane@doe.com",
			},
		},
	}

	var config = config.Root{
		Resources: config.Resources{
			Pipelines: map[string]*resources.Pipeline{
				"my_pipeline": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	resource := out.Resource.Permissions["pipeline_my_pipeline"].(*schema.ResourcePermissions)

	assert.NotEmpty(t, resource.PipelineId)
	assert.Len(t, resource.AccessControl, 1)
	assert.Equal(t, "jane@doe.com", resource.AccessControl[0].UserName)
	assert.Equal(t, "CAN_VIEW", resource.AccessControl[0].PermissionLevel)
}

func TestBundleToTerraformModel(t *testing.T) {
	var src = resources.MlflowModel{
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

	var config = config.Root{
		Resources: config.Resources{
			Models: map[string]*resources.MlflowModel{
				"my_model": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	resource := out.Resource.MlflowModel["my_model"].(*schema.ResourceMlflowModel)

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
	var src = resources.MlflowModel{
		Permissions: []resources.Permission{
			{
				Level:    "CAN_READ",
				UserName: "jane@doe.com",
			},
		},
	}

	var config = config.Root{
		Resources: config.Resources{
			Models: map[string]*resources.MlflowModel{
				"my_model": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	resource := out.Resource.Permissions["mlflow_model_my_model"].(*schema.ResourcePermissions)

	assert.NotEmpty(t, resource.RegisteredModelId)
	assert.Len(t, resource.AccessControl, 1)
	assert.Equal(t, "jane@doe.com", resource.AccessControl[0].UserName)
	assert.Equal(t, "CAN_READ", resource.AccessControl[0].PermissionLevel)
}

func TestBundleToTerraformExperiment(t *testing.T) {
	var src = resources.MlflowExperiment{
		Experiment: &ml.Experiment{
			Name: "name",
		},
	}

	var config = config.Root{
		Resources: config.Resources{
			Experiments: map[string]*resources.MlflowExperiment{
				"my_experiment": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	resource := out.Resource.MlflowExperiment["my_experiment"].(*schema.ResourceMlflowExperiment)

	assert.Equal(t, "name", resource.Name)
	assert.Nil(t, out.Data)
}

func TestBundleToTerraformExperimentPermissions(t *testing.T) {
	var src = resources.MlflowExperiment{
		Permissions: []resources.Permission{
			{
				Level:    "CAN_READ",
				UserName: "jane@doe.com",
			},
		},
	}

	var config = config.Root{
		Resources: config.Resources{
			Experiments: map[string]*resources.MlflowExperiment{
				"my_experiment": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	resource := out.Resource.Permissions["mlflow_experiment_my_experiment"].(*schema.ResourcePermissions)

	assert.NotEmpty(t, resource.ExperimentId)
	assert.Len(t, resource.AccessControl, 1)
	assert.Equal(t, "jane@doe.com", resource.AccessControl[0].UserName)
	assert.Equal(t, "CAN_READ", resource.AccessControl[0].PermissionLevel)

}

func TestBundleToTerraformModelServing(t *testing.T) {
	var src = resources.ModelServingEndpoint{
		CreateServingEndpoint: &serving.CreateServingEndpoint{
			Name: "name",
			Config: serving.EndpointCoreConfigInput{
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

	var config = config.Root{
		Resources: config.Resources{
			ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
				"my_model_serving_endpoint": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	resource := out.Resource.ModelServing["my_model_serving_endpoint"].(*schema.ResourceModelServing)

	assert.Equal(t, "name", resource.Name)
	assert.Equal(t, "model_name", resource.Config.ServedModels[0].ModelName)
	assert.Equal(t, "1", resource.Config.ServedModels[0].ModelVersion)
	assert.Equal(t, true, resource.Config.ServedModels[0].ScaleToZeroEnabled)
	assert.Equal(t, "Small", resource.Config.ServedModels[0].WorkloadSize)
	assert.Equal(t, "model_name-1", resource.Config.TrafficConfig.Routes[0].ServedModelName)
	assert.Equal(t, 100, resource.Config.TrafficConfig.Routes[0].TrafficPercentage)
	assert.Nil(t, out.Data)
}

func TestBundleToTerraformModelServingPermissions(t *testing.T) {
	var src = resources.ModelServingEndpoint{
		Permissions: []resources.Permission{
			{
				Level:    "CAN_VIEW",
				UserName: "jane@doe.com",
			},
		},
	}

	var config = config.Root{
		Resources: config.Resources{
			ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
				"my_model_serving_endpoint": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	resource := out.Resource.Permissions["model_serving_my_model_serving_endpoint"].(*schema.ResourcePermissions)

	assert.NotEmpty(t, resource.ServingEndpointId)
	assert.Len(t, resource.AccessControl, 1)
	assert.Equal(t, "jane@doe.com", resource.AccessControl[0].UserName)
	assert.Equal(t, "CAN_VIEW", resource.AccessControl[0].PermissionLevel)

}

func TestBundleToTerraformRegisteredModel(t *testing.T) {
	var src = resources.RegisteredModel{
		CreateRegisteredModelRequest: &catalog.CreateRegisteredModelRequest{
			Name:        "name",
			CatalogName: "catalog",
			SchemaName:  "schema",
			Comment:     "comment",
		},
	}

	var config = config.Root{
		Resources: config.Resources{
			RegisteredModels: map[string]*resources.RegisteredModel{
				"my_registered_model": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	resource := out.Resource.RegisteredModel["my_registered_model"].(*schema.ResourceRegisteredModel)

	assert.Equal(t, "name", resource.Name)
	assert.Equal(t, "catalog", resource.CatalogName)
	assert.Equal(t, "schema", resource.SchemaName)
	assert.Equal(t, "comment", resource.Comment)
	assert.Nil(t, out.Data)
}

func TestBundleToTerraformRegisteredModelGrants(t *testing.T) {
	var src = resources.RegisteredModel{
		Grants: []resources.Grant{
			{
				Privileges: []string{"EXECUTE"},
				Principal:  "jane@doe.com",
			},
		},
	}

	var config = config.Root{
		Resources: config.Resources{
			RegisteredModels: map[string]*resources.RegisteredModel{
				"my_registered_model": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	resource := out.Resource.Grants["registered_model_my_registered_model"].(*schema.ResourceGrants)

	assert.NotEmpty(t, resource.Function)
	assert.Len(t, resource.Grant, 1)
	assert.Equal(t, "jane@doe.com", resource.Grant[0].Principal)
	assert.Equal(t, "EXECUTE", resource.Grant[0].Privileges[0])
}

func TestTerraformToBundleEmptyLocalResources(t *testing.T) {
	var config = config.Root{
		Resources: config.Resources{},
	}
	var tfState = tfjson.State{
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Type:            "databricks_job",
						Mode:            "managed",
						Name:            "test_job",
						AttributeValues: map[string]interface{}{"id": "1"},
					},
					{
						Type:            "databricks_pipeline",
						Mode:            "managed",
						Name:            "test_pipeline",
						AttributeValues: map[string]interface{}{"id": "1"},
					},
					{
						Type:            "databricks_mlflow_model",
						Mode:            "managed",
						Name:            "test_mlflow_model",
						AttributeValues: map[string]interface{}{"id": "1"},
					},
					{
						Type:            "databricks_mlflow_experiment",
						Mode:            "managed",
						Name:            "test_mlflow_experiment",
						AttributeValues: map[string]interface{}{"id": "1"},
					},
					{
						Type:            "databricks_model_serving",
						Mode:            "managed",
						Name:            "test_model_serving",
						AttributeValues: map[string]interface{}{"id": "1"},
					},
					{
						Type:            "databricks_registered_model",
						Mode:            "managed",
						Name:            "test_registered_model",
						AttributeValues: map[string]interface{}{"id": "1"},
					},
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

	AssertFullResourceCoverage(t, &config)
}

func TestTerraformToBundleEmptyRemoteResources(t *testing.T) {
	var config = config.Root{
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
					PipelineSpec: &pipelines.PipelineSpec{
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
		},
	}
	var tfState = tfjson.State{
		Values: nil,
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

	AssertFullResourceCoverage(t, &config)
}

func TestTerraformToBundleModifiedResources(t *testing.T) {
	var config = config.Root{
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
					PipelineSpec: &pipelines.PipelineSpec{
						Name: "test_pipeline",
					},
				},
				"test_pipeline_new": {
					PipelineSpec: &pipelines.PipelineSpec{
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
		},
	}
	var tfState = tfjson.State{
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Type:            "databricks_job",
						Mode:            "managed",
						Name:            "test_job",
						AttributeValues: map[string]interface{}{"id": "1"},
					},
					{
						Type:            "databricks_job",
						Mode:            "managed",
						Name:            "test_job_old",
						AttributeValues: map[string]interface{}{"id": "2"},
					},
					{
						Type:            "databricks_pipeline",
						Mode:            "managed",
						Name:            "test_pipeline",
						AttributeValues: map[string]interface{}{"id": "1"},
					},
					{
						Type:            "databricks_pipeline",
						Mode:            "managed",
						Name:            "test_pipeline_old",
						AttributeValues: map[string]interface{}{"id": "2"},
					},
					{
						Type:            "databricks_mlflow_model",
						Mode:            "managed",
						Name:            "test_mlflow_model",
						AttributeValues: map[string]interface{}{"id": "1"},
					},
					{
						Type:            "databricks_mlflow_model",
						Mode:            "managed",
						Name:            "test_mlflow_model_old",
						AttributeValues: map[string]interface{}{"id": "2"},
					},
					{
						Type:            "databricks_mlflow_experiment",
						Mode:            "managed",
						Name:            "test_mlflow_experiment",
						AttributeValues: map[string]interface{}{"id": "1"},
					},
					{
						Type:            "databricks_mlflow_experiment",
						Mode:            "managed",
						Name:            "test_mlflow_experiment_old",
						AttributeValues: map[string]interface{}{"id": "2"},
					},
					{
						Type:            "databricks_model_serving",
						Mode:            "managed",
						Name:            "test_model_serving",
						AttributeValues: map[string]interface{}{"id": "1"},
					},
					{
						Type:            "databricks_model_serving",
						Mode:            "managed",
						Name:            "test_model_serving_old",
						AttributeValues: map[string]interface{}{"id": "2"},
					},
					{
						Type:            "databricks_registered_model",
						Mode:            "managed",
						Name:            "test_registered_model",
						AttributeValues: map[string]interface{}{"id": "1"},
					},
					{
						Type:            "databricks_registered_model",
						Mode:            "managed",
						Name:            "test_registered_model_old",
						AttributeValues: map[string]interface{}{"id": "2"},
					},
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

	AssertFullResourceCoverage(t, &config)
}

func AssertFullResourceCoverage(t *testing.T, config *config.Root) {
	resources := reflect.ValueOf(config.Resources)
	for i := 0; i < resources.NumField(); i++ {
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
