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
				RootPath:     "/Users/foo@bar.com",
				ArtifactPath: "/Users/foo@bar.com/artifacts",
				FilePath:     "/Users/foo@bar.com/files",
				StatePath:    "/Users/foo@bar.com/state",
				ResourcePath: "/Users/foo@bar.com/resources",
			},
			Permissions: []resources.Permission{
				{Level: CAN_MANAGE, UserName: "TestUser"},
				{Level: CAN_VIEW, GroupName: "TestGroup"},
				{Level: CAN_RUN, ServicePrincipalName: "TestServicePrincipal"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {JobSettings: jobs.JobSettings{Name: "job_1"}},
					"job_2": {JobSettings: jobs.JobSettings{Name: "job_2"}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline_1": {CreatePipeline: pipelines.CreatePipeline{}},
					"pipeline_2": {CreatePipeline: pipelines.CreatePipeline{}},
				},
				Models: map[string]*resources.MlflowModel{
					"model_1": {},
					"model_2": {},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment_1": {Experiment: ml.Experiment{}},
					"experiment_2": {Experiment: ml.Experiment{}},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint_1": {CreateServingEndpoint: serving.CreateServingEndpoint{}},
					"endpoint_2": {CreateServingEndpoint: serving.CreateServingEndpoint{}},
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
	workspaceApi.EXPECT().SetPermissions(mock.Anything, workspace.WorkspaceObjectPermissionsRequest{
		AccessControlList: []workspace.WorkspaceObjectAccessControlRequest{
			{UserName: "TestUser", PermissionLevel: "CAN_MANAGE"},
			{GroupName: "TestGroup", PermissionLevel: "CAN_READ"},
			{ServicePrincipalName: "TestServicePrincipal", PermissionLevel: "CAN_RUN"},
		},
		WorkspaceObjectId:   "1234",
		WorkspaceObjectType: "directories",
	}).Return(nil, nil)

	diags := bundle.ApplySeq(context.Background(), b, ValidateSharedRootPermissions(), ApplyWorkspaceRootPermissions())
	require.Empty(t, diags)
}

func TestApplyWorkspaceRootPermissionsForAllPaths(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/Some/Root/Path",
				ArtifactPath: "/Users/foo@bar.com/artifacts",
				FilePath:     "/Users/foo@bar.com/files",
				StatePath:    "/Users/foo@bar.com/state",
				ResourcePath: "/Users/foo@bar.com/resources",
			},
			Permissions: []resources.Permission{
				{Level: CAN_MANAGE, UserName: "TestUser"},
				{Level: CAN_VIEW, GroupName: "TestGroup"},
				{Level: CAN_RUN, ServicePrincipalName: "TestServicePrincipal"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {JobSettings: jobs.JobSettings{Name: "job_1"}},
					"job_2": {JobSettings: jobs.JobSettings{Name: "job_2"}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline_1": {CreatePipeline: pipelines.CreatePipeline{}},
					"pipeline_2": {CreatePipeline: pipelines.CreatePipeline{}},
				},
				Models: map[string]*resources.MlflowModel{
					"model_1": {},
					"model_2": {},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment_1": {Experiment: ml.Experiment{}},
					"experiment_2": {Experiment: ml.Experiment{}},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint_1": {CreateServingEndpoint: serving.CreateServingEndpoint{}},
					"endpoint_2": {CreateServingEndpoint: serving.CreateServingEndpoint{}},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	workspaceApi := m.GetMockWorkspaceAPI()
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/Some/Root/Path").Return(&workspace.ObjectInfo{
		ObjectId: 1,
	}, nil)
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/Users/foo@bar.com/artifacts").Return(&workspace.ObjectInfo{
		ObjectId: 2,
	}, nil)
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/Users/foo@bar.com/files").Return(&workspace.ObjectInfo{
		ObjectId: 3,
	}, nil)
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/Users/foo@bar.com/state").Return(&workspace.ObjectInfo{
		ObjectId: 4,
	}, nil)
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/Users/foo@bar.com/resources").Return(&workspace.ObjectInfo{
		ObjectId: 5,
	}, nil)

	workspaceApi.EXPECT().SetPermissions(mock.Anything, workspace.WorkspaceObjectPermissionsRequest{
		AccessControlList: []workspace.WorkspaceObjectAccessControlRequest{
			{UserName: "TestUser", PermissionLevel: "CAN_MANAGE"},
			{GroupName: "TestGroup", PermissionLevel: "CAN_READ"},
			{ServicePrincipalName: "TestServicePrincipal", PermissionLevel: "CAN_RUN"},
		},
		WorkspaceObjectId:   "1",
		WorkspaceObjectType: "directories",
	}).Return(nil, nil)

	workspaceApi.EXPECT().SetPermissions(mock.Anything, workspace.WorkspaceObjectPermissionsRequest{
		AccessControlList: []workspace.WorkspaceObjectAccessControlRequest{
			{UserName: "TestUser", PermissionLevel: "CAN_MANAGE"},
			{GroupName: "TestGroup", PermissionLevel: "CAN_READ"},
			{ServicePrincipalName: "TestServicePrincipal", PermissionLevel: "CAN_RUN"},
		},
		WorkspaceObjectId:   "2",
		WorkspaceObjectType: "directories",
	}).Return(nil, nil)

	workspaceApi.EXPECT().SetPermissions(mock.Anything, workspace.WorkspaceObjectPermissionsRequest{
		AccessControlList: []workspace.WorkspaceObjectAccessControlRequest{
			{UserName: "TestUser", PermissionLevel: "CAN_MANAGE"},
			{GroupName: "TestGroup", PermissionLevel: "CAN_READ"},
			{ServicePrincipalName: "TestServicePrincipal", PermissionLevel: "CAN_RUN"},
		},
		WorkspaceObjectId:   "3",
		WorkspaceObjectType: "directories",
	}).Return(nil, nil)

	workspaceApi.EXPECT().SetPermissions(mock.Anything, workspace.WorkspaceObjectPermissionsRequest{
		AccessControlList: []workspace.WorkspaceObjectAccessControlRequest{
			{UserName: "TestUser", PermissionLevel: "CAN_MANAGE"},
			{GroupName: "TestGroup", PermissionLevel: "CAN_READ"},
			{ServicePrincipalName: "TestServicePrincipal", PermissionLevel: "CAN_RUN"},
		},
		WorkspaceObjectId:   "4",
		WorkspaceObjectType: "directories",
	}).Return(nil, nil)

	workspaceApi.EXPECT().SetPermissions(mock.Anything, workspace.WorkspaceObjectPermissionsRequest{
		AccessControlList: []workspace.WorkspaceObjectAccessControlRequest{
			{UserName: "TestUser", PermissionLevel: "CAN_MANAGE"},
			{GroupName: "TestGroup", PermissionLevel: "CAN_READ"},
			{ServicePrincipalName: "TestServicePrincipal", PermissionLevel: "CAN_RUN"},
		},
		WorkspaceObjectId:   "5",
		WorkspaceObjectType: "directories",
	}).Return(nil, nil)

	diags := bundle.Apply(context.Background(), b, ApplyWorkspaceRootPermissions())
	require.NoError(t, diags.Error())
}
