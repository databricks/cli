package deploy

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIsAnyResourceRunningWithEmptyState(t *testing.T) {
	mock := mocks.NewMockWorkspaceClient(t)
	state := &tfjson.State{}
	err := checkAnyResourceRunning(context.Background(), mock.WorkspaceClient, state)
	require.NoError(t, err)
}

func TestIsAnyResourceRunningWithJob(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	state := &tfjson.State{
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Type: "databricks_job",
						AttributeValues: map[string]interface{}{
							"id": "123",
						},
						Mode: tfjson.ManagedResourceMode,
					},
				},
			},
		},
	}

	jobsApi := m.GetMockJobsAPI()
	jobsApi.EXPECT().ListRunsAll(mock.Anything, jobs.ListRunsRequest{
		JobId:      123,
		ActiveOnly: true,
	}).Return([]jobs.BaseRun{
		{RunId: 1234},
	}, nil).Once()

	err := checkAnyResourceRunning(context.Background(), m.WorkspaceClient, state)
	require.ErrorContains(t, err, "job 123 is running")

	jobsApi.EXPECT().ListRunsAll(mock.Anything, jobs.ListRunsRequest{
		JobId:      123,
		ActiveOnly: true,
	}).Return([]jobs.BaseRun{}, nil).Once()

	err = checkAnyResourceRunning(context.Background(), m.WorkspaceClient, state)
	require.NoError(t, err)
}

func TestIsAnyResourceRunningWithPipeline(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	state := &tfjson.State{
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Type: "databricks_pipeline",
						AttributeValues: map[string]interface{}{
							"id": "123",
						},
						Mode: tfjson.ManagedResourceMode,
					},
				},
			},
		},
	}

	pipelineApi := m.GetMockPipelinesAPI()
	pipelineApi.EXPECT().Get(mock.Anything, pipelines.GetPipelineRequest{
		PipelineId: "123",
	}).Return(&pipelines.GetPipelineResponse{
		PipelineId: "123",
		State:      pipelines.PipelineStateRunning,
	}, nil).Once()

	err := checkAnyResourceRunning(context.Background(), m.WorkspaceClient, state)
	require.ErrorContains(t, err, "pipeline 123 is running")

	pipelineApi.EXPECT().Get(mock.Anything, pipelines.GetPipelineRequest{
		PipelineId: "123",
	}).Return(&pipelines.GetPipelineResponse{
		PipelineId: "123",
		State:      pipelines.PipelineStateIdle,
	}, nil).Once()
	err = checkAnyResourceRunning(context.Background(), m.WorkspaceClient, state)
	require.NoError(t, err)
}

func TestIsAnyResourceRunningWithAPIFailure(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	state := &tfjson.State{
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Type: "databricks_pipeline",
						AttributeValues: map[string]interface{}{
							"id": "123",
						},
						Mode: tfjson.ManagedResourceMode,
					},
				},
			},
		},
	}

	pipelineApi := m.GetMockPipelinesAPI()
	pipelineApi.EXPECT().Get(mock.Anything, pipelines.GetPipelineRequest{
		PipelineId: "123",
	}).Return(nil, errors.New("API failure")).Once()

	err := checkAnyResourceRunning(context.Background(), m.WorkspaceClient, state)
	require.NoError(t, err)
}
