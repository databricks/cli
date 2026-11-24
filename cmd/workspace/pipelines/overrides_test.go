package pipelines

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLooksLikeUUID(t *testing.T) {
	assert.True(t, looksLikeUUID("a12cd3e4-0ab1-1abc-1a2b-1a2bcd3e4f05"))
}

func TestLooksLikeUUID_resourceName(t *testing.T) {
	assert.False(t, looksLikeUUID("my-pipeline-key"))
}

func TestWaitForStartedUpdate_failedUpdateWhoseErrorEventsTimeOut(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockPipelinesAPI()
	api.EXPECT().ListPipelineEventsAll(mock.Anything, pipelines.ListPipelineEventsRequest{
		Filter:     `update_id = 'u1'`,
		MaxResults: 100,
		PipelineId: "p1",
	}).Return(nil, nil)
	api.EXPECT().GetUpdateByPipelineIdAndUpdateId(mock.Anything, "p1", "u1").Return(&pipelines.GetUpdateResponse{
		Update: &pipelines.UpdateInfo{State: pipelines.UpdateInfoStateFailed},
	}, nil)
	api.EXPECT().ListPipelineEventsAll(mock.Anything, pipelines.ListPipelineEventsRequest{
		Filter:     `level='ERROR'`,
		MaxResults: 100,
		PipelineId: "p1",
	}).RunAndReturn(func(ctx context.Context, _ pipelines.ListPipelineEventsRequest) ([]pipelines.PipelineEvent, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	ctx := cmdio.MockDiscard(t.Context())

	err := waitForStartedUpdate(ctx, m.WorkspaceClient, "p1", "u1", 10*time.Millisecond)

	assert.EqualError(t, err, "update u1 failed; its error details could not be retrieved within the 10ms timeout")
}
