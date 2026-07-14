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

func TestUndeclaredWriterTypes(t *testing.T) {
	const deployer = "me@example.com"
	self := resources.Permission{Level: CAN_MANAGE, UserName: deployer}
	other := resources.Permission{Level: CAN_MANAGE, UserName: "other@example.com"}
	sp := resources.Permission{Level: CAN_MANAGE, ServicePrincipalName: "sp-1"}
	group := resources.Permission{Level: CAN_MANAGE, GroupName: "team"}

	cases := []struct {
		name                                   string
		undeclared                             []resources.Permission
		wantSelf, wantOther, wantSP, wantGroup bool
	}{
		{"empty", nil, false, false, false, false},
		{"deploying user", []resources.Permission{self}, true, false, false, false},
		{"other user", []resources.Permission{other}, false, true, false, false},
		{"service principal", []resources.Permission{sp}, false, false, true, false},
		{"group", []resources.Permission{group}, false, false, false, true},
		{"all types", []resources.Permission{self, other, sp, group}, true, true, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotSelf, gotOther, gotSP, gotGroup := undeclaredWriterTypes(tc.undeclared, deployer)
			require.Equal(t, tc.wantSelf, gotSelf)
			require.Equal(t, tc.wantOther, gotOther)
			require.Equal(t, tc.wantSP, gotSP)
			require.Equal(t, tc.wantGroup, gotGroup)
		})
	}
}

func TestUserHomeOwner(t *testing.T) {
	cases := []struct {
		path  string
		owner string
		ok    bool
	}{
		{"/Workspace/Users/alice@example.com/.bundle/x/state", "alice@example.com", true},
		{"/Workspace/Users/alice@example.com", "alice@example.com", true},
		{"/Workspace/Shared/state", "", false},
		{"/Workspace/team/state", "", false},
		{"/Workspace/Users/", "", false},
	}
	for _, tc := range cases {
		owner, ok := userHomeOwner(tc.path)
		require.Equal(t, tc.ok, ok, tc.path)
		require.Equal(t, tc.owner, owner, tc.path)
	}
}
