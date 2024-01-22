package permissions

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestApplyWorkspaceRootPermissions(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/foo@bar.com",
			},
			Permissions: []resources.Permission{
				{Level: CAN_MANAGE, UserName: "TestUser"},
				{Level: CAN_VIEW, GroupName: "TestGroup"},
				{Level: CAN_RUN, ServicePrincipalName: "TestServicePrincipal"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {JobSettings: &jobs.JobSettings{}},
					"job_2": {JobSettings: &jobs.JobSettings{}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline_1": {PipelineSpec: &pipelines.PipelineSpec{}},
					"pipeline_2": {PipelineSpec: &pipelines.PipelineSpec{}},
				},
				Models: map[string]*resources.MlflowModel{
					"model_1": {Model: &ml.Model{}},
					"model_2": {Model: &ml.Model{}},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment_1": {Experiment: &ml.Experiment{}},
					"experiment_2": {Experiment: &ml.Experiment{}},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint_1": {CreateServingEndpoint: &serving.CreateServingEndpoint{}},
					"endpoint_2": {CreateServingEndpoint: &serving.CreateServingEndpoint{}},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	workspaceApi := m.GetMockWorkspaceAPI()
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/Users/foo@bar.com").Return(&workspace.ObjectInfo{
		ObjectId: 1234,
	}, nil)
	workspaceApi.EXPECT().UpdatePermissions(mock.Anything, workspace.WorkspaceObjectPermissionsRequest{
		AccessControlList: []workspace.WorkspaceObjectAccessControlRequest{
			{UserName: "TestUser", PermissionLevel: "CAN_MANAGE"},
			{GroupName: "TestGroup", PermissionLevel: "CAN_READ"},
			{ServicePrincipalName: "TestServicePrincipal", PermissionLevel: "CAN_RUN"},
		},
		WorkspaceObjectId:   "1234",
		WorkspaceObjectType: "directories",
	}).Return(nil, nil)

	err := bundle.Apply(context.Background(), b, ApplyWorkspaceRootPermissions())
	require.NoError(t, err)
}
