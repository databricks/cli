package direct

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOpClient struct {
	bundledeployments.BundleDeploymentsInterface
	requests []bundledeployments.CreateOperationRequest
}

func (f *fakeOpClient) CreateOperation(ctx context.Context, req bundledeployments.CreateOperationRequest) (*bundledeployments.Operation, error) {
	f.requests = append(f.requests, req)
	return &bundledeployments.Operation{}, nil
}

func TestOperationRecorderStripsResourcePrefix(t *testing.T) {
	f := &fakeOpClient{}
	r := NewOperationRecorder(f, "dep-1", 2)

	err := r.record(t.Context(), "resources.jobs.foo", deployplan.Create, "job-123", map[string]string{"name": "foo"})
	require.NoError(t, err)

	require.Len(t, f.requests, 1)
	req := f.requests[0]
	// The wire key drops the CLI-internal "resources." prefix, both in the query
	// param and the operation body.
	assert.Equal(t, "jobs.foo", req.ResourceKey)
	assert.Equal(t, "jobs.foo", req.Operation.ResourceKey)
	assert.Equal(t, "deployments/dep-1/versions/2", req.Parent)
	assert.Equal(t, bundledeployments.OperationActionTypeOperationActionTypeCreate, req.Operation.ActionType)
	assert.Equal(t, "job-123", req.Operation.ResourceId)
	require.NotNil(t, req.Operation.State)
}

func TestOperationRecorderDeleteHasNoState(t *testing.T) {
	f := &fakeOpClient{}
	r := NewOperationRecorder(f, "dep-1", 3)

	err := r.record(t.Context(), "resources.jobs.foo", deployplan.Delete, "", nil)
	require.NoError(t, err)

	require.Len(t, f.requests, 1)
	assert.Equal(t, bundledeployments.OperationActionTypeOperationActionTypeDelete, f.requests[0].Operation.ActionType)
	// Delete operations carry no serialized state.
	assert.Nil(t, f.requests[0].Operation.State)
}

func TestDeployActionToSDK(t *testing.T) {
	cases := []struct {
		action deployplan.ActionType
		want   bundledeployments.OperationActionType
	}{
		{deployplan.Create, bundledeployments.OperationActionTypeOperationActionTypeCreate},
		{deployplan.Update, bundledeployments.OperationActionTypeOperationActionTypeUpdate},
		{deployplan.UpdateWithID, bundledeployments.OperationActionTypeOperationActionTypeUpdateWithId},
		{deployplan.Recreate, bundledeployments.OperationActionTypeOperationActionTypeRecreate},
		{deployplan.Resize, bundledeployments.OperationActionTypeOperationActionTypeResize},
		{deployplan.Delete, bundledeployments.OperationActionTypeOperationActionTypeDelete},
	}
	for _, c := range cases {
		got, err := deployActionToSDK(c.action)
		require.NoError(t, err)
		assert.Equal(t, c.want, got)
	}

	// Skip and Undefined never reach a recorder and are rejected.
	_, err := deployActionToSDK(deployplan.Skip)
	assert.Error(t, err)
	_, err = deployActionToSDK(deployplan.Undefined)
	assert.Error(t, err)
}

func TestRecordOperationNoOpWithoutRecorder(t *testing.T) {
	b := &DeploymentBundle{}
	// No OpRec set: recording is a no-op.
	assert.NoError(t, b.recordOperation(t.Context(), "resources.jobs.foo", deployplan.Create, "id", struct{}{}))
}
