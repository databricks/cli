package terraform

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
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
		JobSettings: jobs.JobSettings{
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
		Permissions: []resources.JobPermission{
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
		JobSettings: jobs.JobSettings{
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
		JobSettings: jobs.JobSettings{
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
		CreatePipeline: pipelines.CreatePipeline{
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
		Permissions: []resources.PipelinePermission{
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
		CreateModelRequest: ml.CreateModelRequest{
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
		CreateModelRequest: ml.CreateModelRequest{
			Name: "name",
		},
		Permissions: []resources.MlflowModelPermission{
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
		Experiment: ml.Experiment{
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
		Experiment: ml.Experiment{
			Name: "name",
		},
		Permissions: []resources.MlflowExperimentPermission{
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
		CreateServingEndpoint: serving.CreateServingEndpoint{
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
		CreateServingEndpoint: serving.CreateServingEndpoint{
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
		Permissions: []resources.ModelServingEndpointPermission{
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
		CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{
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
		CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{
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
		JobSettings: jobs.JobSettings{},
	}
	job2 := resources.Job{
		ModifiedStatus: resources.ModifiedStatusDeleted,
		JobSettings:    jobs.JobSettings{},
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
