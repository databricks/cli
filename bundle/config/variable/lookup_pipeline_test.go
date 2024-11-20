package variable

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLookupPipeline_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockPipelinesAPI()
	api.EXPECT().
		GetByName(mock.Anything, "pipeline").
		Return(&pipelines.PipelineStateInfo{
			PipelineId: "abcd",
		}, nil)

	ctx := context.Background()
	l := lookupPipeline{name: "pipeline"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "abcd", result)
}

func TestLookupPipeline_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockPipelinesAPI()
	api.EXPECT().
		GetByName(mock.Anything, "pipeline").
		Return(nil, &apierr.APIError{StatusCode: 404})

	ctx := context.Background()
	l := lookupPipeline{name: "pipeline"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorIs(t, err, apierr.ErrNotFound)
}

func TestLookupPipeline_String(t *testing.T) {
	l := lookupPipeline{name: "name"}
	assert.Equal(t, "pipeline: name", l.String())
}
