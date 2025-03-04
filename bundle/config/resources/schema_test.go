package resources

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSchemaNotFound(t *testing.T) {
	ctx := context.Background()

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockSchemasAPI().On("GetByFullName", mock.Anything, "non-existent-schema").Return(nil, &apierr.APIError{
		StatusCode: 404,
	})

	s := &Schema{}
	exists, err := s.Exists(ctx, m.WorkspaceClient, "non-existent-schema")

	require.Falsef(t, exists, "Exists should return false when getting a 404 response from Workspace")
	require.NoErrorf(t, err, "Exists should not return an error when getting a 404 response from Workspace")
}
