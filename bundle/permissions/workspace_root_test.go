package permissions

import (
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
				RootPath:     "/Users/foo@bar.test",
				ArtifactPath: "/Users/foo@bar.test/artifacts",
				FilePath:     "/Users/foo@bar.test/files",
				StatePath:    "/Users/foo@bar.test/state",
				ResourcePath: "/Users/foo@bar.test/resources",
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
					"experiment_1": {CreateExperiment: ml.CreateExperiment{}},
					"experiment_2": {CreateExperiment: ml.CreateExperiment{}},
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
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/Users/foo@bar.test").Return(&workspace.ObjectInfo{
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
	}).Return(&workspace.WorkspaceObjectPermissions{}, nil)

	diags := bundle.ApplySeq(t.Context(), b, ValidateSharedRootPermissions(), ApplyWorkspaceRootPermissions())
	require.Empty(t, diags)
}

func TestApplyWorkspaceRootPermissionsForAllPaths(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/Some/Root/Path",
				ArtifactPath: "/Users/foo@bar.test/artifacts",
				FilePath:     "/Users/foo@bar.test/files",
				StatePath:    "/Users/foo@bar.test/state",
				ResourcePath: "/Users/foo@bar.test/resources",
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
					"experiment_1": {CreateExperiment: ml.CreateExperiment{}},
					"experiment_2": {CreateExperiment: ml.CreateExperiment{}},
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
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/Users/foo@bar.test/artifacts").Return(&workspace.ObjectInfo{
		ObjectId: 2,
	}, nil)
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/Users/foo@bar.test/files").Return(&workspace.ObjectInfo{
		ObjectId: 3,
	}, nil)
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/Users/foo@bar.test/state").Return(&workspace.ObjectInfo{
		ObjectId: 4,
	}, nil)
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/Users/foo@bar.test/resources").Return(&workspace.ObjectInfo{
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
	}).Return(&workspace.WorkspaceObjectPermissions{}, nil)

	workspaceApi.EXPECT().SetPermissions(mock.Anything, workspace.WorkspaceObjectPermissionsRequest{
		AccessControlList: []workspace.WorkspaceObjectAccessControlRequest{
			{UserName: "TestUser", PermissionLevel: "CAN_MANAGE"},
			{GroupName: "TestGroup", PermissionLevel: "CAN_READ"},
			{ServicePrincipalName: "TestServicePrincipal", PermissionLevel: "CAN_RUN"},
		},
		WorkspaceObjectId:   "2",
		WorkspaceObjectType: "directories",
	}).Return(&workspace.WorkspaceObjectPermissions{}, nil)

	workspaceApi.EXPECT().SetPermissions(mock.Anything, workspace.WorkspaceObjectPermissionsRequest{
		AccessControlList: []workspace.WorkspaceObjectAccessControlRequest{
			{UserName: "TestUser", PermissionLevel: "CAN_MANAGE"},
			{GroupName: "TestGroup", PermissionLevel: "CAN_READ"},
			{ServicePrincipalName: "TestServicePrincipal", PermissionLevel: "CAN_RUN"},
		},
		WorkspaceObjectId:   "3",
		WorkspaceObjectType: "directories",
	}).Return(&workspace.WorkspaceObjectPermissions{}, nil)

	workspaceApi.EXPECT().SetPermissions(mock.Anything, workspace.WorkspaceObjectPermissionsRequest{
		AccessControlList: []workspace.WorkspaceObjectAccessControlRequest{
			{UserName: "TestUser", PermissionLevel: "CAN_MANAGE"},
			{GroupName: "TestGroup", PermissionLevel: "CAN_READ"},
			{ServicePrincipalName: "TestServicePrincipal", PermissionLevel: "CAN_RUN"},
		},
		WorkspaceObjectId:   "4",
		WorkspaceObjectType: "directories",
	}).Return(&workspace.WorkspaceObjectPermissions{}, nil)

	workspaceApi.EXPECT().SetPermissions(mock.Anything, workspace.WorkspaceObjectPermissionsRequest{
		AccessControlList: []workspace.WorkspaceObjectAccessControlRequest{
			{UserName: "TestUser", PermissionLevel: "CAN_MANAGE"},
			{GroupName: "TestGroup", PermissionLevel: "CAN_READ"},
			{ServicePrincipalName: "TestServicePrincipal", PermissionLevel: "CAN_RUN"},
		},
		WorkspaceObjectId:   "5",
		WorkspaceObjectType: "directories",
	}).Return(&workspace.WorkspaceObjectPermissions{}, nil)

	diags := bundle.Apply(t.Context(), b, ApplyWorkspaceRootPermissions())
	require.NoError(t, diags.Error())
}

func TestPathContains(t *testing.T) {
	testCases := []struct {
		parent   string
		child    string
		expected bool
	}{
		{"/Workspace/u/bundle", "/Workspace/u/bundle/state", true},
		{"/Workspace/u/bundle", "/Workspace/u/bundle", true},
		{"/Workspace/u/bundle/", "/Workspace/u/bundle", true},
		{"/Workspace/u/bundle/", "/Workspace/u/bundle/state", true},
		{"/Workspace/u/bundle", "/Workspace/u/bundle-2/state", false},
		{"/Workspace/u/bundle", "/Workspace/Shared/state", false},
		{"/Workspace/u/bundle/state", "/Workspace/u/bundle", false},
		{"", "/Workspace/u/bundle", true},
		{"/Workspace/u/bundle", "", true},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.expected, pathContains(tc.parent, tc.child), "parent=%q child=%q", tc.parent, tc.child)
	}
}
