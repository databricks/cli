package variable

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResolveServicePrincipal_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockServicePrincipalsAPI()
	api.EXPECT().
		GetByDisplayName(mock.Anything, "service-principal").
		Return(&iam.ServicePrincipal{
			ApplicationId: "5678",
		}, nil)

	ctx := context.Background()
	l := resolveServicePrincipal{name: "service-principal"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "5678", result)
}

func TestResolveServicePrincipal_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockServicePrincipalsAPI()
	api.EXPECT().
		GetByDisplayName(mock.Anything, "service-principal").
		Return(nil, &apierr.APIError{StatusCode: 404})

	ctx := context.Background()
	l := resolveServicePrincipal{name: "service-principal"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorIs(t, err, apierr.ErrNotFound)
}

func TestResolveServicePrincipal_String(t *testing.T) {
	l := resolveServicePrincipal{name: "name"}
	assert.Equal(t, "service-principal: name", l.String())
}
