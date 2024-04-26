package deploy

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIsAnyResourceRunningWithEmptyState(t *testing.T) {
	mock := mocks.NewMockWorkspaceClient(t)
	err := checkAnyResourceRunning(context.Background(), mock.WorkspaceClient, &config.Resources{})
	require.NoError(t, err)
}

func TestIsAnyResourceRunningWithJob(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	resources := &config.Resources{
		Jobs: map[string]*resources.Job{
			"job1": {ID: "123"},
		},
	}

	jobsApi := m.GetMockJobsAPI()
	jobsApi.EXPECT().ListRunsAll(mock.Anything, jobs.ListRunsRequest{
		JobId:      123,
		ActiveOnly: true,
	}).Return([]jobs.BaseRun{
		{RunId: 1234},
	}, nil).Once()

	err := checkAnyResourceRunning(context.Background(), m.WorkspaceClient, resources)
	require.ErrorContains(t, err, "job 123 is running")

	jobsApi.EXPECT().ListRunsAll(mock.Anything, jobs.ListRunsRequest{
		JobId:      123,
		ActiveOnly: true,
	}).Return([]jobs.BaseRun{}, nil).Once()

	err = checkAnyResourceRunning(context.Background(), m.WorkspaceClient, resources)
	require.NoError(t, err)
}

func TestIsAnyResourceRunningWithPipeline(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	resources := &config.Resources{
		Pipelines: map[string]*resources.Pipeline{
			"pipeline1": {ID: "123"},
		},
	}

	pipelineApi := m.GetMockPipelinesAPI()
	pipelineApi.EXPECT().Get(mock.Anything, pipelines.GetPipelineRequest{
		PipelineId: "123",
	}).Return(&pipelines.GetPipelineResponse{
		PipelineId: "123",
		State:      pipelines.PipelineStateRunning,
	}, nil).Once()

	err := checkAnyResourceRunning(context.Background(), m.WorkspaceClient, resources)
	require.ErrorContains(t, err, "pipeline 123 is running")

	pipelineApi.EXPECT().Get(mock.Anything, pipelines.GetPipelineRequest{
		PipelineId: "123",
	}).Return(&pipelines.GetPipelineResponse{
		PipelineId: "123",
		State:      pipelines.PipelineStateIdle,
	}, nil).Once()
	err = checkAnyResourceRunning(context.Background(), m.WorkspaceClient, resources)
	require.NoError(t, err)
}

func TestIsAnyResourceRunningWithAPIFailure(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	resources := &config.Resources{
		Pipelines: map[string]*resources.Pipeline{
			"pipeline1": {ID: "123"},
		},
	}

	pipelineApi := m.GetMockPipelinesAPI()
	pipelineApi.EXPECT().Get(mock.Anything, pipelines.GetPipelineRequest{
		PipelineId: "123",
	}).Return(nil, errors.New("API failure")).Once()

	err := checkAnyResourceRunning(context.Background(), m.WorkspaceClient, resources)
	require.NoError(t, err)
}
