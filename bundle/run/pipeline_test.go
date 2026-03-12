package run

import (
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/cmdio"
	sdk_config "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPipelineRunnerCancel(t *testing.T) {
	pipeline := &resources.Pipeline{
		BaseResource: resources.BaseResource{ID: "123"},
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"test_pipeline": pipeline,
				},
			},
		},
	}

	runner := pipelineRunner{key: "test", bundle: b, pipeline: pipeline}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	mockWait := &pipelines.WaitGetPipelineIdle[struct{}]{
		Poll: func(time.Duration, func(*pipelines.GetPipelineResponse)) (*pipelines.GetPipelineResponse, error) {
			return nil, nil
		},
	}

	pipelineApi := m.GetMockPipelinesAPI()
	pipelineApi.EXPECT().Stop(t.Context(), pipelines.StopRequest{
		PipelineId: "123",
	}).Return(mockWait, nil)

	err := runner.Cancel(t.Context())
	require.NoError(t, err)
}

func TestPipelineRunnerRestart(t *testing.T) {
	pipeline := &resources.Pipeline{
		BaseResource: resources.BaseResource{ID: "123"},
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"test_pipeline": pipeline,
				},
			},
		},
	}

	runner := pipelineRunner{key: "test", bundle: b, pipeline: pipeline}

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &sdk_config.Config{
		Host: "https://test.com",
	}
	b.SetWorkpaceClient(m.WorkspaceClient)

	ctx := cmdio.MockDiscard(t.Context())

	mockWait := &pipelines.WaitGetPipelineIdle[struct{}]{
		Poll: func(time.Duration, func(*pipelines.GetPipelineResponse)) (*pipelines.GetPipelineResponse, error) {
			return nil, nil
		},
	}

	pipelineApi := m.GetMockPipelinesAPI()
	pipelineApi.EXPECT().Stop(mock.Anything, pipelines.StopRequest{
		PipelineId: "123",
	}).Return(mockWait, nil)

	// Mock runner starting a new update
	pipelineApi.EXPECT().StartUpdate(mock.Anything, pipelines.StartUpdate{
		PipelineId: "123",
	}).Return(&pipelines.StartUpdateResponse{
		UpdateId: "456",
	}, nil)

	// Mock runner polling for events
	pipelineApi.EXPECT().ListPipelineEventsAll(mock.Anything, pipelines.ListPipelineEventsRequest{
		Filter:     `update_id = '456'`,
		MaxResults: 100,
		PipelineId: "123",
	}).Return([]pipelines.PipelineEvent{}, nil)

	// Mock runner polling for update status
	pipelineApi.EXPECT().GetUpdateByPipelineIdAndUpdateId(mock.Anything, "123", "456").
		Return(&pipelines.GetUpdateResponse{
			Update: &pipelines.UpdateInfo{
				State: pipelines.UpdateInfoStateCompleted,
			},
		}, nil)

	_, err := runner.Restart(ctx, &Options{})
	require.NoError(t, err)
}
