package client

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// setupJobTestMocks configures common mocks for job submission tests.
// Returns the mock client and a pointer that will be set with the captured SubmitRun request.
func setupJobTestMocks(t *testing.T, ctx context.Context, runID int64) (*mocks.MockWorkspaceClient, *jobs.SubmitRun) {
	t.Helper()
	m := mocks.NewMockWorkspaceClient(t)

	m.GetMockCurrentUserAPI().EXPECT().Me(ctx).Return(&iam.User{
		UserName: "testuser@example.com",
	}, nil)

	m.GetMockWorkspaceAPI().EXPECT().MkdirsByPath(ctx, mock.AnythingOfType("string")).Return(nil)
	m.GetMockWorkspaceAPI().EXPECT().Import(ctx, mock.AnythingOfType("workspace.Import")).Return(nil)

	var capturedRequest jobs.SubmitRun
	m.GetMockJobsAPI().EXPECT().Submit(ctx, mock.AnythingOfType("jobs.SubmitRun")).
		Run(func(_ context.Context, req jobs.SubmitRun) {
			capturedRequest = req
		}).
		Return(&jobs.WaitGetRunJobTerminatedOrSkipped[jobs.SubmitRunResponse]{RunId: runID}, nil)

	m.GetMockJobsAPI().EXPECT().GetRun(ctx, jobs.GetRunRequest{RunId: runID}).Return(&jobs.Run{
		Tasks: []jobs.RunTask{
			{
				TaskKey: sshServerTaskKey,
				Status: &jobs.RunStatus{
					State: jobs.RunLifecycleStateV2StateRunning,
				},
			},
		},
	}, nil)

	return m, &capturedRequest
}

func TestSubmitJob_ClassicCluster(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m, captured := setupJobTestMocks(t, ctx, 42)

	opts := ClientOptions{
		ClusterID:          "cluster-123",
		ServerTimeout:      30 * time.Minute,
		TaskStartupTimeout: 5 * time.Minute,
	}

	err := submitSSHTunnelJob(ctx, m.WorkspaceClient, "0.1.0", "test-scope", opts)
	require.NoError(t, err)

	require.Len(t, captured.Tasks, 1)
	task := captured.Tasks[0]
	assert.Equal(t, sshServerTaskKey, task.TaskKey)
	assert.Equal(t, "cluster-123", task.ExistingClusterId)
	assert.Empty(t, task.EnvironmentKey)
	assert.Nil(t, task.Compute)
	assert.Nil(t, captured.Environments)
}

func TestSubmitJob_ServerlessGPU(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m, captured := setupJobTestMocks(t, ctx, 43)

	opts := ClientOptions{
		ConnectionName:     "gpu-conn",
		Accelerator:        "GPU_1xA10",
		ServerTimeout:      30 * time.Minute,
		TaskStartupTimeout: 5 * time.Minute,
	}

	err := submitSSHTunnelJob(ctx, m.WorkspaceClient, "0.1.0", "test-scope", opts)
	require.NoError(t, err)

	require.Len(t, captured.Tasks, 1)
	task := captured.Tasks[0]
	assert.Equal(t, sshServerTaskKey, task.TaskKey)
	assert.Empty(t, task.ExistingClusterId)
	assert.Equal(t, serverlessEnvironmentKey, task.EnvironmentKey)
	require.NotNil(t, task.Compute)
	assert.Equal(t, compute.HardwareAcceleratorType("GPU_1xA10"), task.Compute.HardwareAccelerator)

	require.Len(t, captured.Environments, 1)
	assert.Equal(t, serverlessEnvironmentKey, captured.Environments[0].EnvironmentKey)
	require.NotNil(t, captured.Environments[0].Spec)
	assert.Equal(t, "4", captured.Environments[0].Spec.EnvironmentVersion)
}

func TestSubmitJob_ServerlessCPU(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m, captured := setupJobTestMocks(t, ctx, 44)

	opts := ClientOptions{
		ConnectionName:     "cpu-conn",
		ServerTimeout:      30 * time.Minute,
		TaskStartupTimeout: 5 * time.Minute,
	}

	err := submitSSHTunnelJob(ctx, m.WorkspaceClient, "0.1.0", "test-scope", opts)
	require.NoError(t, err)

	require.Len(t, captured.Tasks, 1)
	task := captured.Tasks[0]
	assert.Equal(t, sshServerTaskKey, task.TaskKey)
	assert.Empty(t, task.ExistingClusterId)
	assert.Equal(t, serverlessEnvironmentKey, task.EnvironmentKey)
	assert.Nil(t, task.Compute)

	require.Len(t, captured.Environments, 1)
	assert.Equal(t, serverlessEnvironmentKey, captured.Environments[0].EnvironmentKey)
}

func TestSubmitJob_NotebookUpload(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	m.GetMockCurrentUserAPI().EXPECT().Me(ctx).Return(&iam.User{
		UserName: "testuser@example.com",
	}, nil)

	m.GetMockWorkspaceAPI().EXPECT().MkdirsByPath(ctx, mock.AnythingOfType("string")).Return(nil)

	var capturedImport workspace.Import
	m.GetMockWorkspaceAPI().EXPECT().Import(ctx, mock.AnythingOfType("workspace.Import")).
		Run(func(_ context.Context, req workspace.Import) {
			capturedImport = req
		}).
		Return(nil)

	m.GetMockJobsAPI().EXPECT().Submit(ctx, mock.AnythingOfType("jobs.SubmitRun")).
		Return(&jobs.WaitGetRunJobTerminatedOrSkipped[jobs.SubmitRunResponse]{RunId: 45}, nil)

	m.GetMockJobsAPI().EXPECT().GetRun(ctx, jobs.GetRunRequest{RunId: 45}).Return(&jobs.Run{
		Tasks: []jobs.RunTask{
			{
				TaskKey: sshServerTaskKey,
				Status:  &jobs.RunStatus{State: jobs.RunLifecycleStateV2StateRunning},
			},
		},
	}, nil)

	opts := ClientOptions{
		ClusterID:          "cluster-123",
		ServerTimeout:      30 * time.Minute,
		TaskStartupTimeout: 5 * time.Minute,
	}

	err := submitSSHTunnelJob(ctx, m.WorkspaceClient, "0.1.0", "test-scope", opts)
	require.NoError(t, err)

	assert.Contains(t, capturedImport.Path, "ssh-server-bootstrap")
	assert.Equal(t, workspace.ImportFormatSource, capturedImport.Format)
	assert.Equal(t, workspace.LanguagePython, capturedImport.Language)
	assert.True(t, capturedImport.Overwrite)
	assert.NotEmpty(t, capturedImport.Content)
}

func TestSubmitJob_TaskParameters(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m, captured := setupJobTestMocks(t, ctx, 46)

	opts := ClientOptions{
		ClusterID:          "cluster-123",
		ShutdownDelay:      10 * time.Minute,
		MaxClients:         5,
		ServerTimeout:      30 * time.Minute,
		TaskStartupTimeout: 5 * time.Minute,
	}

	err := submitSSHTunnelJob(ctx, m.WorkspaceClient, "0.1.0", "test-scope", opts)
	require.NoError(t, err)

	task := captured.Tasks[0]
	params := task.NotebookTask.BaseParameters
	assert.Equal(t, "0.1.0", params["version"])
	assert.Equal(t, "test-scope", params["secretScopeName"])
	assert.Equal(t, "10m0s", params["shutdownDelay"])
	assert.Equal(t, "5", params["maxClients"])
	assert.Equal(t, "cluster-123", params["sessionId"])
	assert.Equal(t, "false", params["serverless"])
}

func TestSubmitJob_ServerlessParameters(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m, captured := setupJobTestMocks(t, ctx, 47)

	opts := ClientOptions{
		ConnectionName:     "gpu-conn",
		Accelerator:        "GPU_8xH100",
		ServerTimeout:      30 * time.Minute,
		TaskStartupTimeout: 5 * time.Minute,
	}

	err := submitSSHTunnelJob(ctx, m.WorkspaceClient, "0.1.0", "test-scope", opts)
	require.NoError(t, err)

	task := captured.Tasks[0]
	params := task.NotebookTask.BaseParameters
	assert.Equal(t, "gpu-conn", params["sessionId"])
	assert.Equal(t, "true", params["serverless"])

	require.NotNil(t, task.Compute)
	assert.Equal(t, compute.HardwareAcceleratorType("GPU_8xH100"), task.Compute.HardwareAccelerator)
}

func TestSubmitJob_CustomEnvironmentVersion(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m, captured := setupJobTestMocks(t, ctx, 48)

	opts := ClientOptions{
		ConnectionName:     "my-conn",
		EnvironmentVersion: 7,
		ServerTimeout:      30 * time.Minute,
		TaskStartupTimeout: 5 * time.Minute,
	}

	err := submitSSHTunnelJob(ctx, m.WorkspaceClient, "0.1.0", "test-scope", opts)
	require.NoError(t, err)

	require.Len(t, captured.Environments, 1)
	assert.Equal(t, "7", captured.Environments[0].Spec.EnvironmentVersion)
}
