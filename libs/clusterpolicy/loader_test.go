package clusterpolicy

import (
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLoaderCachesDefinitions(t *testing.T) {
	api := mocks.NewMockWorkspaceClient(t).GetMockClusterPoliciesAPI()
	api.EXPECT().GetByPolicyId(mock.Anything, "policy").Return(&compute.Policy{
		Definition: `{"spark_version": {"type": "fixed", "value": "15.4.x-scala2.12"}}`,
	}, nil).Once()

	loader := NewLoader(api)
	for range 2 {
		definition, err := loader.Get(t.Context(), "policy")
		require.NoError(t, err)
		assert.Equal(t, Definition{"spark_version": {Type: "fixed", Value: "15.4.x-scala2.12"}}, definition)
	}
}

func TestLoaderCachesFailures(t *testing.T) {
	for _, tt := range []struct {
		name   string
		policy *compute.Policy
		err    error
	}{
		{"api error", nil, apierr.ErrResourceDoesNotExist},
		{"invalid definition", &compute.Policy{Definition: `{"spark_version": `}, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			api := mocks.NewMockWorkspaceClient(t).GetMockClusterPoliciesAPI()
			api.EXPECT().GetByPolicyId(mock.Anything, "policy").Return(tt.policy, tt.err).Once()

			loader := NewLoader(api)
			for range 2 {
				definition, err := loader.Get(t.Context(), "policy")
				assert.Nil(t, definition)
				lookupErr, ok := errors.AsType[*LookupError](err)
				require.True(t, ok, "expected *LookupError, got %T", err)
				assert.Equal(t, "policy", lookupErr.PolicyID)
				if tt.err != nil {
					assert.ErrorIs(t, err, apierr.ErrNotFound)
					assert.EqualError(t, err, "reading cluster policy policy: "+tt.err.Error())
				}
			}
		})
	}
}
