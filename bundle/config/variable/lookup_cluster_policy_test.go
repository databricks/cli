package variable

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLookupClusterPolicy_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockClusterPoliciesAPI()
	api.EXPECT().
		GetByName(mock.Anything, "policy").
		Return(&compute.Policy{
			PolicyId: "1234",
		}, nil)

	ctx := context.Background()
	l := lookupClusterPolicy{name: "policy"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "1234", result)
}

func TestLookupClusterPolicy_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockClusterPoliciesAPI()
	api.EXPECT().
		GetByName(mock.Anything, "policy").
		Return(nil, &apierr.APIError{StatusCode: 404})

	ctx := context.Background()
	l := lookupClusterPolicy{name: "policy"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorIs(t, err, apierr.ErrNotFound)
}

func TestLookupClusterPolicy_String(t *testing.T) {
	l := lookupClusterPolicy{name: "name"}
	assert.Equal(t, "cluster-policy: name", l.String())
}
