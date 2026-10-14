package variable

import (
	"testing"

	"github.com/databricks/cli/libs/sdktypes/scim/v1"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResolveServicePrincipal_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockServicePrincipalsV2API()
	iterator := listing.SliceIterator[scim.ServicePrincipal]([]scim.ServicePrincipal{
		{
			ApplicationId: "5678",
		},
	})
	api.EXPECT().
		List(mock.Anything, scim.ListServicePrincipalsRequest{
			Filter: "displayName eq 'service-principal'",
		}).
		Return(&iterator)

	ctx := t.Context()
	l := resolveServicePrincipal{name: "service-principal"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "5678", result)
}

func TestResolveServicePrincipal_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockServicePrincipalsV2API()
	iterator := listing.SliceIterator[scim.ServicePrincipal]([]scim.ServicePrincipal{})
	api.EXPECT().
		List(mock.Anything, scim.ListServicePrincipalsRequest{
			Filter: "displayName eq 'service-principal'",
		}).
		Return(&iterator)

	ctx := t.Context()
	l := resolveServicePrincipal{name: "service-principal"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorContains(t, err, "service principal named \"service-principal\" does not exist")
}

func TestResolveServicePrincipal_String(t *testing.T) {
	l := resolveServicePrincipal{name: "name"}
	assert.Equal(t, "service-principal: name", l.String())
}
