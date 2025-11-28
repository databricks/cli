package resources

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSecretExists(t *testing.T) {
	ctx := context.Background()

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockSecretsAPI().On("GetSecret", mock.Anything, workspace.GetSecretRequest{
		Scope: "my_scope",
		Key:   "my_key",
	}).Return(&workspace.GetSecretResponse{
		Key: "my_key",
	}, nil)

	s := &Secret{
		Scope: "my_scope",
		Key:   "my_key",
	}
	exists, err := s.Exists(ctx, m.WorkspaceClient, "my_scope/my_key")

	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSecretNotFound(t *testing.T) {
	ctx := context.Background()

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockSecretsAPI().On("GetSecret", mock.Anything, workspace.GetSecretRequest{
		Scope: "my_scope",
		Key:   "my_key",
	}).Return((*workspace.GetSecretResponse)(nil), &apierr.APIError{
		StatusCode: 404,
		ErrorCode:  "RESOURCE_DOES_NOT_EXIST",
	})

	s := &Secret{
		Scope: "my_scope",
		Key:   "my_key",
	}
	exists, err := s.Exists(ctx, m.WorkspaceClient, "my_scope/my_key")

	require.NoError(t, err)
	assert.False(t, exists)
}

func TestSecretExistsError(t *testing.T) {
	ctx := context.Background()

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockSecretsAPI().On("GetSecret", mock.Anything, workspace.GetSecretRequest{
		Scope: "my_scope",
		Key:   "my_key",
	}).Return((*workspace.GetSecretResponse)(nil), &apierr.APIError{
		StatusCode: 500,
		ErrorCode:  "INTERNAL_ERROR",
	})

	s := &Secret{
		Scope: "my_scope",
		Key:   "my_key",
	}
	_, err := s.Exists(ctx, m.WorkspaceClient, "my_scope/my_key")

	require.Error(t, err)
}

func TestSecretGetName(t *testing.T) {
	s := &Secret{
		Scope: "my_scope",
		Key:   "my_key",
	}
	assert.Equal(t, "my_scope/my_key", s.GetName())

	s.ID = "custom_id"
	assert.Equal(t, "custom_id", s.GetName())
}
