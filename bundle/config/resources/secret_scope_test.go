package resources

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSecretScopeExists(t *testing.T) {
	ctx := context.Background()

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockSecretsAPI().On("ListScopesAll", mock.Anything).Return([]workspace.SecretScope{
		{Name: "my_scope"},
		{Name: "other_scope"},
	}, nil)

	s := &SecretScope{
		Name: "my_scope",
	}
	exists, err := s.Exists(ctx, m.WorkspaceClient, "my_scope")

	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSecretScopeNotFound(t *testing.T) {
	ctx := context.Background()

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockSecretsAPI().On("ListScopesAll", mock.Anything).Return([]workspace.SecretScope{
		{Name: "other_scope"},
	}, nil)

	s := &SecretScope{
		Name: "my_scope",
	}
	exists, err := s.Exists(ctx, m.WorkspaceClient, "my_scope")

	require.NoError(t, err)
	assert.False(t, exists)
}

func TestSecretScopeGetName(t *testing.T) {
	s := &SecretScope{
		Name: "my_scope",
	}
	assert.Equal(t, "my_scope", s.GetName())

	s.ID = "custom_id"
	assert.Equal(t, "custom_id", s.GetName())
}

func TestSecretScopeResourceDescription(t *testing.T) {
	s := &SecretScope{}
	desc := s.ResourceDescription()

	assert.Equal(t, "secret_scope", desc.SingularName)
	assert.Equal(t, "secret_scopes", desc.PluralName)
	assert.Equal(t, "Secret Scope", desc.SingularTitle)
	assert.Equal(t, "Secret Scopes", desc.PluralTitle)
}
