package direct

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBundleClient struct {
	sdkbundle.BundleInterface
	requests []sdkbundle.CreateOperationRequest
	err      error
}

func (c *fakeBundleClient) CreateOperation(_ context.Context, req sdkbundle.CreateOperationRequest) (*sdkbundle.Operation, error) {
	c.requests = append(c.requests, req)
	if c.err != nil {
		return nil, c.err
	}
	return &sdkbundle.Operation{}, nil
}

func TestOperationRecorderRecordsUnderVersionParent(t *testing.T) {
	client := &fakeBundleClient{}
	rec := NewOperationRecorder(client, "dep-1", 7)

	err := rec.record(t.Context(), "resources.jobs.foo", deployplan.Create, "job-123", map[string]string{"name": "foo"})
	require.NoError(t, err)

	require.Len(t, client.requests, 1)
	req := client.requests[0]
	assert.Equal(t, "deployments/dep-1/versions/7", req.Parent)
	assert.Equal(t, "resources.jobs.foo", req.ResourceKey)
	assert.Equal(t, "resources.jobs.foo", req.Operation.ResourceKey)
	assert.Equal(t, "job-123", req.Operation.ResourceId)
	assert.Equal(t, sdkbundle.OperationStatusOperationStatusSucceeded, req.Operation.Status)
}

func TestOperationRecorderSerializesState(t *testing.T) {
	client := &fakeBundleClient{}
	rec := NewOperationRecorder(client, "dep", 1)

	err := rec.record(t.Context(), "resources.jobs.foo", deployplan.Update, "job-1", map[string]any{"name": "foo", "tasks": 2})
	require.NoError(t, err)

	require.Len(t, client.requests, 1)
	state := client.requests[0].Operation.State
	require.NotNil(t, state)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(*state, &decoded))
	assert.Equal(t, "foo", decoded["name"])
}

func TestOperationRecorderOmitsStateForDelete(t *testing.T) {
	client := &fakeBundleClient{}
	rec := NewOperationRecorder(client, "dep", 1)

	err := rec.record(t.Context(), "resources.jobs.foo", deployplan.Delete, "job-1", nil)
	require.NoError(t, err)

	require.Len(t, client.requests, 1)
	assert.Nil(t, client.requests[0].Operation.State)
	assert.Equal(t, sdkbundle.OperationActionTypeOperationActionTypeDelete, client.requests[0].Operation.ActionType)
}

func TestOperationRecorderPropagatesAPIError(t *testing.T) {
	wantErr := errors.New("boom")
	client := &fakeBundleClient{err: wantErr}
	rec := NewOperationRecorder(client, "dep", 1)

	err := rec.record(t.Context(), "resources.jobs.foo", deployplan.Create, "job-1", nil)
	assert.ErrorIs(t, err, wantErr)
}

func TestDeployActionToSDKMapping(t *testing.T) {
	cases := map[deployplan.ActionType]sdkbundle.OperationActionType{
		deployplan.Create:       sdkbundle.OperationActionTypeOperationActionTypeCreate,
		deployplan.Update:       sdkbundle.OperationActionTypeOperationActionTypeUpdate,
		deployplan.UpdateWithID: sdkbundle.OperationActionTypeOperationActionTypeUpdateWithId,
		deployplan.Recreate:     sdkbundle.OperationActionTypeOperationActionTypeRecreate,
		deployplan.Resize:       sdkbundle.OperationActionTypeOperationActionTypeResize,
		deployplan.Delete:       sdkbundle.OperationActionTypeOperationActionTypeDelete,
	}

	for action, want := range cases {
		got, err := deployActionToSDK(action)
		require.NoError(t, err, "action: %s", action)
		assert.Equal(t, want, got, "action: %s", action)
	}
}

func TestDeployActionToSDKRejectsNonMutatingActions(t *testing.T) {
	for _, action := range []deployplan.ActionType{deployplan.Skip, deployplan.Undefined} {
		_, err := deployActionToSDK(action)
		assert.Error(t, err, "action: %s", action)
	}
}
