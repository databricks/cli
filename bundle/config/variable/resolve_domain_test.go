package variable

import (
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/domains"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResolveDomain_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockDomainsAPI()
	api.EXPECT().
		ListDomainsAll(mock.Anything, mock.Anything).
		Return([]domains.Domain{
			{DomainId: "1234", TagKey: "Payments"},
			{DomainId: "5678", TagKey: "Payments/Fraud", ParentDomainId: "1234"},
		}, nil)

	ctx := t.Context()
	l := resolveDomain{name: "Payments/Fraud"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "5678", result)
}

func TestResolveDomain_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockDomainsAPI()
	api.EXPECT().
		ListDomainsAll(mock.Anything, mock.Anything).
		Return([]domains.Domain{
			{DomainId: "1234", TagKey: "Payments"},
		}, nil)

	ctx := t.Context()
	l := resolveDomain{name: "Finance"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorContains(t, err, `domain with tag key "Finance" does not exist`)
}

func TestResolveDomain_ResolveError(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockDomainsAPI()
	api.EXPECT().
		ListDomainsAll(mock.Anything, mock.Anything).
		Return(nil, errors.New("User is not authorized to access domains"))

	ctx := t.Context()
	l := resolveDomain{name: "Payments"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorContains(t, err, "not authorized")
}

func TestResolveDomain_String(t *testing.T) {
	l := resolveDomain{name: "name"}
	assert.Equal(t, "domain: name", l.String())
}
