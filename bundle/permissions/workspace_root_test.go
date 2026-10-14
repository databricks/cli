package permissions

import (
	"encoding/json"
	"github.com/databricks/cli/libs/sdktypes/experiments/v1"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/sdktypes/modelserving/v1"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"sync"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/cli/libs/workspaceapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var expectedRootACL = []workspaceapi.WorkspaceObjectAccessControlRequest{
	{UserName: "TestUser", PermissionLevel: "CAN_MANAGE"},
	{GroupName: "TestGroup", PermissionLevel: "CAN_READ"},
	{ServicePrincipalName: "TestServicePrincipal", PermissionLevel: "CAN_RUN"},
}

// fakeDirs records the ACLs set on workspace directories.
type fakeDirs struct {
	mu  sync.Mutex
	set map[string][]workspaceapi.WorkspaceObjectAccessControlRequest
}

// fakeDirectories serves get-status for the given directories (path -> object id)
// and records permission updates on them.
func fakeDirectories(t *testing.T, b *bundle.Bundle, ids map[string]int64) *fakeDirs {
	w, server := bundletest.NewClients(t)
	b.SetWorkpaceClient(w)
	dirs := &fakeDirs{set: map[string][]workspaceapi.WorkspaceObjectAccessControlRequest{}}
	server.Handle("GET", "/api/2.0/workspace/get-status", func(req testserver.Request) any {
		path := req.URL.Query().Get("path")
		id, ok := ids[path]
		if !ok {
			return testserver.Response{StatusCode: 404, Body: map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": path}}
		}
		return workspaceapi.ObjectInfo{ObjectId: id, ObjectType: workspaceapi.ObjectTypeDirectory, Path: path}
	})
	server.Handle("PUT", "/api/2.0/permissions/directories/{id}", func(req testserver.Request) any {
		var body workspaceapi.WorkspaceObjectPermissionsRequest
		assert.NoError(t, json.Unmarshal(req.Body, &body))
		dirs.mu.Lock()
		defer dirs.mu.Unlock()
		dirs.set[req.Vars["id"]] = body.AccessControlList
		return workspaceapi.WorkspaceObjectPermissions{}
	})
	return dirs
}

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
					"job_1": {JobSettings: jobs.JobSettings{Name: new("job_1")}},
					"job_2": {JobSettings: jobs.JobSettings{Name: new("job_2")}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline_1": {CreatePipelineRequest: pipelines.CreatePipelineRequest{}},
					"pipeline_2": {CreatePipelineRequest: pipelines.CreatePipelineRequest{}},
				},
				Models: map[string]*resources.MlflowModel{
					"model_1": {},
					"model_2": {},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment_1": {CreateExperimentRequest: experiments.CreateExperimentRequest{}},
					"experiment_2": {CreateExperimentRequest: experiments.CreateExperimentRequest{}},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint_1": {CreateInferenceEndpointRequest: modelserving.CreateInferenceEndpointRequest{}},
					"endpoint_2": {CreateInferenceEndpointRequest: modelserving.CreateInferenceEndpointRequest{}},
				},
			},
		},
	}

	dirs := fakeDirectories(t, b, map[string]int64{"/Users/foo@bar.test": 1234})

	diags := bundle.ApplySeq(t.Context(), b, ValidateSharedRootPermissions(), ApplyWorkspaceRootPermissions())
	require.Empty(t, diags)
	assert.Equal(t, map[string][]workspaceapi.WorkspaceObjectAccessControlRequest{"1234": expectedRootACL}, dirs.set)
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
					"job_1": {JobSettings: jobs.JobSettings{Name: new("job_1")}},
					"job_2": {JobSettings: jobs.JobSettings{Name: new("job_2")}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline_1": {CreatePipelineRequest: pipelines.CreatePipelineRequest{}},
					"pipeline_2": {CreatePipelineRequest: pipelines.CreatePipelineRequest{}},
				},
				Models: map[string]*resources.MlflowModel{
					"model_1": {},
					"model_2": {},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment_1": {CreateExperimentRequest: experiments.CreateExperimentRequest{}},
					"experiment_2": {CreateExperimentRequest: experiments.CreateExperimentRequest{}},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint_1": {CreateInferenceEndpointRequest: modelserving.CreateInferenceEndpointRequest{}},
					"endpoint_2": {CreateInferenceEndpointRequest: modelserving.CreateInferenceEndpointRequest{}},
				},
			},
		},
	}

	dirs := fakeDirectories(t, b, map[string]int64{
		"/Some/Root/Path":               1,
		"/Users/foo@bar.test/artifacts": 2,
		"/Users/foo@bar.test/files":     3,
		"/Users/foo@bar.test/state":     4,
		"/Users/foo@bar.test/resources": 5,
	})

	diags := bundle.Apply(t.Context(), b, ApplyWorkspaceRootPermissions())
	require.NoError(t, diags.Error())
	assert.Equal(t, map[string][]workspaceapi.WorkspaceObjectAccessControlRequest{
		"1": expectedRootACL,
		"2": expectedRootACL,
		"3": expectedRootACL,
		"4": expectedRootACL,
		"5": expectedRootACL,
	}, dirs.set)
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
