package direct

import (
	"context"
	"sync"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOpClient struct {
	bundledeployments.BundleDeploymentsInterface

	mu       sync.Mutex
	requests []bundledeployments.CreateOperationRequest
}

func (f *fakeOpClient) CreateOperation(ctx context.Context, req bundledeployments.CreateOperationRequest) (*bundledeployments.Operation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)
	return &bundledeployments.Operation{}, nil
}

// uploadOne records a single operation through the given uploader, mirroring what
// an operationQueue worker does.
func uploadOne(t *testing.T, u operationUploader, resourceKey string, action deployplan.ActionType, resourceID string, state any) {
	t.Helper()
	op, err := newRecordedOperation(action, resourceID, state, nil)
	require.NoError(t, err)
	require.NoError(t, u.upload(t.Context(), resourceKey, op))
}

func TestOperationRecorderStripsResourcePrefix(t *testing.T) {
	f := &fakeOpClient{}
	r := NewOperationRecorder(f, "dep-1", 2)

	uploadOne(t, r, "resources.jobs.foo", deployplan.Create, "job-123", map[string]string{"name": "foo"})

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

	uploadOne(t, r, "resources.jobs.foo", deployplan.Delete, "", nil)

	require.Len(t, f.requests, 1)
	assert.Equal(t, bundledeployments.OperationActionTypeOperationActionTypeDelete, f.requests[0].Operation.ActionType)
	// Delete operations carry no serialized state.
	assert.Nil(t, f.requests[0].Operation.State)
}

func TestNewRecordedOperationRedactsSensitiveFields(t *testing.T) {
	state := struct {
		Name  string `json:"name"`
		Token string `json:"token" bundle:"sensitive"`
	}{Name: "foo", Token: "super-secret"}

	op, err := newRecordedOperation(deployplan.Create, "job-123", state, nil)
	require.NoError(t, err)

	// Sensitive fields are redacted before leaving the CLI, matching what
	// dstate.SaveState writes to the local state file.
	assert.JSONEq(t,
		`{"state":{"name":"foo","token":"`+dyn.SensitiveValueRedacted+`"}}`,
		string(op.state))
}

func TestNewRecordedOperationRecordsDependsOn(t *testing.T) {
	// depends_on rides in an envelope alongside the config: it cannot be
	// recomputed from the config, whose references are already resolved.
	dependsOn := []deployplan.DependsOnEntry{{Node: "resources.jobs.bar", Label: "${resources.jobs.bar.id}"}}

	op, err := newRecordedOperation(deployplan.Create, "job-123", map[string]string{"name": "foo"}, dependsOn)
	require.NoError(t, err)

	assert.JSONEq(t,
		`{"state":{"name":"foo"},"depends_on":[{"node":"resources.jobs.bar","label":"${resources.jobs.bar.id}"}]}`,
		string(op.state))
}

func TestNewRecordedOperationRejectsUnsupportedAction(t *testing.T) {
	_, err := newRecordedOperation(deployplan.Skip, "job-123", nil, nil)
	assert.Error(t, err)
}

func TestMergeAction(t *testing.T) {
	const (
		create   = bundledeployments.OperationActionTypeOperationActionTypeCreate
		recreate = bundledeployments.OperationActionTypeOperationActionTypeRecreate
		update   = bundledeployments.OperationActionTypeOperationActionTypeUpdate
		resize   = bundledeployments.OperationActionTypeOperationActionTypeResize
		del      = bundledeployments.OperationActionTypeOperationActionTypeDelete
	)

	cases := []struct {
		queued, next, want bundledeployments.OperationActionType
	}{
		// A queued create is not downgraded: the resource is still new.
		{create, update, create},
		{create, resize, create},
		{recreate, update, recreate},
		{create, create, create},
		// A delete wins: the resource is gone, so the earlier action is moot.
		{create, del, del},
		{update, del, del},
		// Neither side is a create, so the later action stands.
		{update, resize, resize},
		{resize, update, update},
		{del, create, create},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, mergeAction(c.queued, c.next), "queued %s, next %s", c.queued, c.next)
	}
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
