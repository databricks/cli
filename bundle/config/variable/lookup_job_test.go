package variable

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLookupJob_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockJobsAPI()
	api.EXPECT().
		GetBySettingsName(mock.Anything, "job").
		Return(&jobs.BaseJob{
			JobId: 5678,
		}, nil)

	ctx := context.Background()
	l := lookupJob{name: "job"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "5678", result)
}

func TestLookupJob_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockJobsAPI()
	api.EXPECT().
		GetBySettingsName(mock.Anything, "job").
		Return(nil, &apierr.APIError{StatusCode: 404})

	ctx := context.Background()
	l := lookupJob{name: "job"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorIs(t, err, apierr.ErrNotFound)
}

func TestLookupJob_String(t *testing.T) {
	l := lookupJob{name: "name"}
	assert.Equal(t, "job: name", l.String())
}
