package direct

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeOpCall is one recorded call to the operations API.
type fakeOpCall struct {
	method      string
	parent      string
	resourceKey string
	op          bundledeployments.Operation
	update      updateOperationRequest
	fields      []string
}

type fakeOpClient struct {
	mu    sync.Mutex
	calls []fakeOpCall
	// sequence is what the service reports back; a string, as the service sends it.
	sequence string
}

func (f *fakeOpClient) CreateOperation(ctx context.Context, parent, resourceKey string, op bundledeployments.Operation) (operationResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fakeOpCall{method: "create", parent: parent, resourceKey: resourceKey, op: op})
	return operationResponse{SequenceId: f.sequence}, nil
}

func (f *fakeOpClient) UpdateOperation(ctx context.Context, parent, resourceKey string, fields []string, body updateOperationRequest) (operationResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fakeOpCall{method: "update", parent: parent, resourceKey: resourceKey, update: body, fields: fields})
	return operationResponse{SequenceId: f.sequence}, nil
}

// uploadOne records a single operation through the given uploader, mirroring what
// an operationQueue worker does.
func uploadOne(t *testing.T, u operationUploader, resourceKey string, action deployplan.ActionType, resourceID string, state json.RawMessage) {
	t.Helper()
	op, err := newStateOperation(dstate.OperationInfo{Action: action}, resourceID, state)
	require.NoError(t, err)
	require.NoError(t, u.upload(t.Context(), resourceKey, op))
}

func TestOperationRecorderStripsResourcePrefix(t *testing.T) {
	f := &fakeOpClient{sequence: "1"}
	r := newOperationRecorder(f, "dep-1", 2)

	uploadOne(t, r, "resources.jobs.foo", deployplan.Create, "job-123", envelope(t, "foo"))

	require.Len(t, f.calls, 1)
	c := f.calls[0]
	// The wire key drops the CLI-internal "resources." prefix, both in the query
	// param and the operation body.
	assert.Equal(t, "create", c.method)
	assert.Equal(t, "jobs.foo", c.resourceKey)
	assert.Equal(t, "jobs.foo", c.op.ResourceKey)
	assert.Equal(t, "deployments/dep-1/versions/2", c.parent)
	assert.Equal(t, bundledeployments.OperationActionTypeOperationActionTypeCreate, c.op.ActionType)
	assert.Equal(t, "job-123", c.op.ResourceId)
	require.NotNil(t, c.op.State)
}

func TestOperationRecorderUpdatesSecondWriteForSameResource(t *testing.T) {
	// One operation per resource per version: the second write has to update the
	// first, echoing the sequence_id the service returned as its precondition.
	f := &fakeOpClient{sequence: "7"}
	r := newOperationRecorder(f, "dep-1", 2)

	uploadOne(t, r, "resources.jobs.foo", deployplan.Recreate, "", nil)
	uploadOne(t, r, "resources.jobs.foo", deployplan.Recreate, "job-456", envelope(t, "new"))

	require.Len(t, f.calls, 2)
	assert.Equal(t, "create", f.calls[0].method)

	assert.Equal(t, "update", f.calls[1].method)
	assert.Equal(t, "jobs.foo", f.calls[1].resourceKey)
	assert.Equal(t, "7", f.calls[1].update.SequenceId)
	assert.Equal(t, "job-456", f.calls[1].update.ResourceId)
	require.NotNil(t, f.calls[1].update.State)
}

func TestOperationRecorderFailureAfterAStateWriteKeepsTheState(t *testing.T) {
	// A create whose WaitAfterCreate fails has already written state for a resource
	// that exists remotely. Updating the operation must only mark it failed: sending
	// the failure's own empty state and id would clear both, and a resource with no
	// state is dropped from the deployment, so the next plan would re-create it.
	f := &fakeOpClient{sequence: "3"}
	r := newOperationRecorder(f, "dep-1", 2)

	uploadOne(t, r, "resources.job_runs.my_run", deployplan.Create, "run-1", envelope(t, "the run"))

	failed, err := newFailedOperation(deployplan.Create, "", nil, errors.New("run did not succeed: FAILED"))
	require.NoError(t, err)
	require.NoError(t, r.upload(t.Context(), "resources.job_runs.my_run", failed))

	require.Len(t, f.calls, 2)
	assert.Equal(t, "create", f.calls[0].method)

	update := f.calls[1]
	assert.Equal(t, "update", update.method)
	assert.Equal(t, failureFields, update.fields)
	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, update.update.Status)
	assert.Equal(t, "run did not succeed: FAILED", update.update.ErrorMessage)
	// Neither is in the mask, so what the create recorded stands.
	assert.Nil(t, update.update.State)
	assert.Empty(t, update.update.ResourceId)
}

func TestOperationRecorderFailureBeforeAnyWriteCarriesPriorState(t *testing.T) {
	// Nothing has been recorded for the resource, so the failure creates the operation
	// and has to carry the prior state itself - the resource still exists, and the
	// service rejects an operation that records state without an id.
	f := &fakeOpClient{sequence: "1"}
	r := newOperationRecorder(f, "dep-1", 2)

	failed, err := newFailedOperation(deployplan.Update, "main.some_schema", envelope(t, "before"), errors.New("boom"))
	require.NoError(t, err)
	require.NoError(t, r.upload(t.Context(), "resources.schemas.foo", failed))

	require.Len(t, f.calls, 1)
	assert.Equal(t, "create", f.calls[0].method)
	assert.Equal(t, "main.some_schema", f.calls[0].op.ResourceId)
	require.NotNil(t, f.calls[0].op.State)
}

func TestOperationRecorderTracksSequencePerResource(t *testing.T) {
	// A different resource has its own operation, so its first write creates.
	f := &fakeOpClient{sequence: "1"}
	r := newOperationRecorder(f, "dep-1", 2)

	uploadOne(t, r, "resources.jobs.foo", deployplan.Create, "id-1", envelope(t, "foo"))
	uploadOne(t, r, "resources.jobs.bar", deployplan.Create, "id-2", envelope(t, "bar"))

	require.Len(t, f.calls, 2)
	assert.Equal(t, "create", f.calls[0].method)
	assert.Equal(t, "create", f.calls[1].method)
}

func TestNewStateOperationRecordsEnvelopeAsIs(t *testing.T) {
	// The state DB serializes the envelope (see dstate.SaveState); the operation
	// carries it through untouched, sensitive fields and all.
	state := json.RawMessage(`{"state":{"name":"foo","token":"super-secret"}}`)

	op, err := newStateOperation(dstate.OperationInfo{Action: deployplan.Create}, "job-123", state)
	require.NoError(t, err)

	assert.JSONEq(t, string(state), string(op.state))
	assert.Equal(t, bundledeployments.OperationStatusOperationStatusSucceeded, op.status)
}

func TestNewStateOperationRejectsUnsupportedAction(t *testing.T) {
	_, err := newStateOperation(dstate.OperationInfo{Action: deployplan.Skip}, "job-123", nil)
	assert.Error(t, err)
}

func TestNewStateOperationRejectsOversizedState(t *testing.T) {
	big := json.RawMessage(strings.Repeat("x", maxOperationStateSize+1))

	_, err := newStateOperation(dstate.OperationInfo{Action: deployplan.Create}, "job-123", big)
	assert.ErrorContains(t, err, "exceeds the 65536 byte limit")
}

func TestNewFailedOperationRecordsError(t *testing.T) {
	op, err := newFailedOperation(deployplan.Create, "", nil, errors.New("cluster spec is invalid"))
	require.NoError(t, err)

	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, op.status)
	assert.Equal(t, "cluster spec is invalid", op.errorMessage)
	// The resource was never written, so there is no state to serve back for it.
	assert.Nil(t, op.state)
}

func TestNewFailedOperationRecordsPriorStateWithID(t *testing.T) {
	// A failed recreate has already deleted the resource, so the id must come from
	// the pre-deploy record alongside the state: the service rejects state without
	// an id, since state describes a resource that exists.
	op, err := newFailedOperation(deployplan.Recreate, "main.some_schema", json.RawMessage(`{"state":{"catalog_name":"main"}}`), errors.New("Catalog 'mainx' does not exist"))
	require.NoError(t, err)

	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, op.status)
	assert.Equal(t, "main.some_schema", op.resourceID)
	assert.JSONEq(t, `{"state":{"catalog_name":"main"}}`, string(op.state))
}

func TestNewFailedOperationTruncatesLongError(t *testing.T) {
	// Truncated rather than rejected: a message over the limit would make recording
	// fail and hide the error it is reporting.
	op, err := newFailedOperation(deployplan.Update, "job-123", nil, errors.New(strings.Repeat("x", maxOperationErrorMessageSize+100)))
	require.NoError(t, err)

	assert.Len(t, op.errorMessage, maxOperationErrorMessageSize)
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
