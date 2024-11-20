package variable

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLookupQuery_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockQueriesAPI()
	api.EXPECT().
		GetByDisplayName(mock.Anything, "query").
		Return(&sql.ListQueryObjectsResponseQuery{
			Id: "1234",
		}, nil)

	ctx := context.Background()
	l := &lookupQuery{name: "query"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "1234", result)
}

func TestLookupQuery_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockQueriesAPI()
	api.EXPECT().
		GetByDisplayName(mock.Anything, "query").
		Return(nil, &apierr.APIError{StatusCode: 404})

	ctx := context.Background()
	l := &lookupQuery{name: "query"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorIs(t, err, apierr.ErrNotFound)
}

func TestLookupQuery_String(t *testing.T) {
	l := &lookupQuery{name: "name"}
	assert.Equal(t, "query: name", l.String())
}
