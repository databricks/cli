package permissions

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/require"
)

type MockWorkspaceClient struct {
	t *testing.T
}

// Delete implements workspace.WorkspaceService.
func (MockWorkspaceClient) Delete(ctx context.Context, request workspace.Delete) error {
	panic("unimplemented")
}

// Export implements workspace.WorkspaceService.
func (MockWorkspaceClient) Export(ctx context.Context, request workspace.ExportRequest) (*workspace.ExportResponse, error) {
	panic("unimplemented")
}

// GetPermissionLevels implements workspace.WorkspaceService.
func (MockWorkspaceClient) GetPermissionLevels(ctx context.Context, request workspace.GetWorkspaceObjectPermissionLevelsRequest) (*workspace.GetWorkspaceObjectPermissionLevelsResponse, error) {
	panic("unimplemented")
}

// GetPermissions implements workspace.WorkspaceService.
func (MockWorkspaceClient) GetPermissions(ctx context.Context, request workspace.GetWorkspaceObjectPermissionsRequest) (*workspace.WorkspaceObjectPermissions, error) {
	panic("unimplemented")
}

// GetStatus implements workspace.WorkspaceService.
func (MockWorkspaceClient) GetStatus(ctx context.Context, request workspace.GetStatusRequest) (*workspace.ObjectInfo, error) {
	panic("unimplemented")
}

// Import implements workspace.WorkspaceService.
func (MockWorkspaceClient) Import(ctx context.Context, request workspace.Import) error {
	panic("unimplemented")
}

// List implements workspace.WorkspaceService.
func (MockWorkspaceClient) List(ctx context.Context, request workspace.ListWorkspaceRequest) (*workspace.ListResponse, error) {
	return &workspace.ListResponse{
		Objects: []workspace.ObjectInfo{
			{ObjectId: 1234, ObjectType: "DIRECTORY", Path: "/Users/foo@bar.com"},
		},
	}, nil
}

// Mkdirs implements workspace.WorkspaceService.
func (MockWorkspaceClient) Mkdirs(ctx context.Context, request workspace.Mkdirs) error {
	panic("unimplemented")
}

// SetPermissions implements workspace.WorkspaceService.
func (MockWorkspaceClient) SetPermissions(ctx context.Context, request workspace.WorkspaceObjectPermissionsRequest) (*workspace.WorkspaceObjectPermissions, error) {
	panic("unimplemented")
}

// UpdatePermissions implements workspace.WorkspaceService.
func (m MockWorkspaceClient) UpdatePermissions(ctx context.Context, request workspace.WorkspaceObjectPermissionsRequest) (*workspace.WorkspaceObjectPermissions, error) {
	require.Equal(m.t, "1234", request.WorkspaceObjectId)
	require.Equal(m.t, "DIRECTORY", request.WorkspaceObjectType)
	require.Contains(m.t, request.AccessControlList, workspace.WorkspaceObjectAccessControlRequest{
		UserName:        "TestUser",
		PermissionLevel: "CAN_MANAGE",
	})

	return nil, nil
}

func TestApplyBundlePermissions(t *testing.T) {
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

	b.WorkspaceClient().Workspace.WithImpl(MockWorkspaceClient{t})

	err := bundle.Apply(context.Background(), b, ApplyBundlePermissions())
	require.NoError(t, err)

	require.Len(t, b.Config.Resources.Jobs["job_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_MANAGE_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Jobs["job_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_MANAGE_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.Permission{Level: "CAN_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.Permission{Level: "CAN_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Models["model_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Models["model_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Models["model_1"].Permissions, resources.Permission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Models["model_2"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Models["model_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Models["model_2"].Permissions, resources.Permission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Experiments["experiment_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Experiments["experiment_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Experiments["experiment_1"].Permissions, resources.Permission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Experiments["experiment_2"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Experiments["experiment_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Experiments["experiment_2"].Permissions, resources.Permission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.Permission{Level: "CAN_QUERY", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.Permission{Level: "CAN_QUERY", ServicePrincipalName: "TestServicePrincipal"})
}

func TestWarningOnOverlapPermission(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/foo@bar.com",
			},
			Permissions: []resources.Permission{
				{Level: CAN_MANAGE, UserName: "TestUser"},
				{Level: CAN_VIEW, GroupName: "TestGroup"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {
						Permissions: []resources.Permission{
							{Level: CAN_VIEW, UserName: "TestUser"},
						},
						JobSettings: &jobs.JobSettings{},
					},
					"job_2": {
						Permissions: []resources.Permission{
							{Level: CAN_VIEW, UserName: "TestUser2"},
						},
						JobSettings: &jobs.JobSettings{},
					},
				},
			},
		},
	}

	b.WorkspaceClient().Workspace.WithImpl(MockWorkspaceClient{t})

	err := bundle.Apply(context.Background(), b, ApplyBundlePermissions())
	require.NoError(t, err)

	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_VIEW", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_VIEW", UserName: "TestUser2"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})

}
