package run

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/require"
)

func TestPipelineRunnerCancel(t *testing.T) {
	pipeline := &resources.Pipeline{
		ID: "123",
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
	pipelineApi.EXPECT().Stop(context.Background(), pipelines.StopRequest{
		PipelineId: "123",
	}).Return(mockWait, nil)

	err := runner.Cancel(context.Background())
	require.NoError(t, err)
}
