package direct

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

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
	update      updateOperationRequest
	fields      []string
}

type fakeOpClient struct {
	mu    sync.Mutex
	calls []fakeOpCall
	// sequence is what the service reports back; a string, as the service sends it.
	sequence string
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
	// The version already staged this operation, so the first write updates it and echoes
	// the sequence id staging left. The wire key drops the CLI-internal "resources." prefix.
	assert.Equal(t, "update", c.method)
	assert.Equal(t, "jobs.foo", c.resourceKey)
	assert.Equal(t, "deployments/dep-1/versions/2", c.parent)
	assert.Equal(t, stagedSequenceID, c.update.SequenceId)
	assert.Equal(t, "job-123", c.update.ResourceId)
	require.NotEmpty(t, c.update.State)
}

func TestOperationRecorderUpdatesSecondWriteForSameResource(t *testing.T) {
	// One operation per resource per version: the second write has to update the
	// first, echoing the sequence_id the service returned as its precondition.
	f := &fakeOpClient{sequence: "7"}
	r := newOperationRecorder(f, "dep-1", 2)

	uploadOne(t, r, "resources.jobs.foo", deployplan.Recreate, "", nil)
	uploadOne(t, r, "resources.jobs.foo", deployplan.Recreate, "job-456", envelope(t, "new"))

	require.Len(t, f.calls, 2)
	assert.Equal(t, stagedSequenceID, f.calls[0].update.SequenceId)

	assert.Equal(t, "update", f.calls[1].method)
	assert.Equal(t, "jobs.foo", f.calls[1].resourceKey)
	assert.Equal(t, "7", f.calls[1].update.SequenceId)
	assert.Equal(t, "job-456", f.calls[1].update.ResourceId)
	require.NotEmpty(t, f.calls[1].update.State)
}

func TestOperationRecorderFailureAfterAStateWriteKeepsTheState(t *testing.T) {
	// The create wrote state for a resource that exists, then the wait failed. The update
	// only marks it failed: sending empty state would drop the resource from the deployment.
	f := &fakeOpClient{sequence: "3"}
	r := newOperationRecorder(f, "dep-1", 2)

	uploadOne(t, r, "resources.job_runs.my_run", deployplan.Create, "run-1", envelope(t, "the run"))

	failed, err := newFailedOperation(deployplan.Create, "", errors.New("run did not succeed: FAILED"))
	require.NoError(t, err)
	require.NoError(t, r.upload(t.Context(), "resources.job_runs.my_run", failed))

	require.Len(t, f.calls, 2)
	assert.Equal(t, "update", f.calls[0].method)

	update := f.calls[1]
	assert.Equal(t, "update", update.method)
	assert.Equal(t, []string{"error_message", "status"}, update.fields)
	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, update.update.Status)
	assert.Equal(t, "run did not succeed: FAILED", update.update.ErrorMessage)
	// Neither is in the mask, so what the create recorded stands.
	assert.Empty(t, update.update.State)
	assert.Empty(t, update.update.ResourceId)
}

func TestOperationRecorderFailedRecreateKeepsTheResourceGone(t *testing.T) {
	// The recreate's delete is recorded with no state, and then the create fails. The failure
	// must not fill that gap with the pre-deploy state: the resource really is gone.
	f := &fakeOpClient{sequence: "2"}
	r := newOperationRecorder(f, "dep-1", 2)

	uploadOne(t, r, "resources.jobs.foo", deployplan.Recreate, "old-id", nil)

	failed, err := newFailedOperation(deployplan.Recreate, "old-id", errors.New("boom"))
	require.NoError(t, err)
	require.NoError(t, r.upload(t.Context(), "resources.jobs.foo", failed))

	require.Len(t, f.calls, 2)
	update := f.calls[1]
	assert.Equal(t, "update", update.method)
	assert.Equal(t, []string{"error_message", "status"}, update.fields)
	assert.Empty(t, update.update.State)
	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, update.update.Status)
	assert.Equal(t, "boom", update.update.ErrorMessage)
}

func TestOperationRecorderFailureCarryingALaterWriteSendsIt(t *testing.T) {
	// Two writes, the first uploaded and the second still waiting when the resource failed,
	// so the failure took it over. The update must name state or the first write's stands.
	f := &fakeOpClient{sequence: "4"}
	r := newOperationRecorder(f, "dep-1", 2)

	uploadOne(t, r, "resources.jobs.foo", deployplan.Update, "id-1", envelope(t, "first write"))

	second, err := newStateOperation(dstate.OperationInfo{Action: deployplan.Update}, "id-1", envelope(t, "second write"))
	require.NoError(t, err)
	failed, err := newFailedOperation(deployplan.Update, "id-old", errors.New("boom"))
	require.NoError(t, err)
	require.NoError(t, r.upload(t.Context(), "resources.jobs.foo", coalesce(second, failed)))

	require.Len(t, f.calls, 2)
	update := f.calls[1]
	assert.Equal(t, []string{"state", "error_message", "resource_id", "status"}, update.fields)
	require.NotEmpty(t, update.update.State)
	assert.Contains(t, update.update.State, "second write")
	assert.Equal(t, "id-1", update.update.ResourceId)
}

func TestOperationRecorderFailureBeforeAnyWriteNarrowsTheStagedOperation(t *testing.T) {
	// Nothing was written for the resource, so the failure updates the operation the version
	// staged, at the sequence id staging left. It sends no state: the resource was not
	// touched, and the staged operation already holds whatever the deployment knows.
	f := &fakeOpClient{sequence: "1"}
	r := newOperationRecorder(f, "dep-1", 2)

	failed, err := newFailedOperation(deployplan.Update, "main.some_schema", errors.New("boom"))
	require.NoError(t, err)
	require.NoError(t, r.upload(t.Context(), "resources.schemas.foo", failed))

	require.Len(t, f.calls, 1)
	assert.Equal(t, "update", f.calls[0].method)
	assert.Equal(t, stagedSequenceID, f.calls[0].update.SequenceId)
	assert.Equal(t, []string{"error_message", "status"}, f.calls[0].fields)
	assert.Empty(t, f.calls[0].update.State)
}

func TestOperationRecorderTracksSequencePerResource(t *testing.T) {
	// Each resource has its own staged operation, so each one's first write echoes the staged
	// sequence id rather than a sequence another resource earned.
	f := &fakeOpClient{sequence: "1"}
	r := newOperationRecorder(f, "dep-1", 2)

	uploadOne(t, r, "resources.jobs.foo", deployplan.Create, "id-1", envelope(t, "foo"))
	uploadOne(t, r, "resources.jobs.bar", deployplan.Create, "id-2", envelope(t, "bar"))

	require.Len(t, f.calls, 2)
	assert.Equal(t, stagedSequenceID, f.calls[0].update.SequenceId)
	assert.Equal(t, stagedSequenceID, f.calls[1].update.SequenceId)
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
	op, err := newFailedOperation(deployplan.Create, "", errors.New("cluster spec is invalid"))
	require.NoError(t, err)

	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, op.status)
	assert.Equal(t, "cluster spec is invalid", op.errorMessage)
	// The resource was never written, so there is no state to serve back for it.
	assert.Nil(t, op.state)
	// An update only marks the operation failed; see failedKeepingState.
	assert.Equal(t, failedKeepingState, op.updateFields)
}

func TestNewFailedOperationTruncatesLongError(t *testing.T) {
	// Truncated rather than rejected: a message over the limit would make recording
	// fail and hide the error it is reporting.
	op, err := newFailedOperation(deployplan.Update, "job-123", errors.New(strings.Repeat("x", maxOperationErrorMessageSize+100)))
	require.NoError(t, err)

	assert.Len(t, op.errorMessage, maxOperationErrorMessageSize)
}

func TestNewFailedOperationPreservesUTF8OnTruncation(t *testing.T) {
	// The cut lands one byte into the emoji, so a byte-wise truncation would leave a partial
	// rune behind and the service stores state and messages as strings.
	msg := strings.Repeat("a", maxOperationErrorMessageSize-1) + "❌" + "x"

	op, err := newFailedOperation(deployplan.Update, "job-123", errors.New(msg))
	require.NoError(t, err)

	assert.True(t, utf8.ValidString(op.errorMessage))
	// The whole emoji went, so the message is shorter than the limit rather than exactly it.
	assert.Equal(t, strings.Repeat("a", maxOperationErrorMessageSize-1), op.errorMessage)
}

func TestOperationRecorderReturnsAPIErrors(t *testing.T) {
	// A failed upload returns its error and leaves the recorded sequence id alone, so a later
	// write for the same resource still updates the operation with the precondition the
	// service last gave us rather than trying to create a second one.
	failingClient := &failingOpClient{sequence: "9", failOn: 1}
	r := newOperationRecorder(failingClient, "dep-1", 2)

	uploadOne(t, r, "resources.jobs.foo", deployplan.Create, "job-1", envelope(t, "first"))

	second, err := newStateOperation(dstate.OperationInfo{Action: deployplan.Update}, "job-2", envelope(t, "second"))
	require.NoError(t, err)
	err = r.upload(t.Context(), "resources.jobs.foo", second)
	require.Error(t, err)
	assert.Equal(t, "injected error", err.Error())

	// The third write is what proves the sequence id survived the failure.
	uploadOne(t, r, "resources.jobs.foo", deployplan.Update, "job-3", envelope(t, "third"))

	require.Len(t, failingClient.calls, 3)
	assert.Equal(t, "update", failingClient.calls[0].method)
	assert.Equal(t, "update", failingClient.calls[1].method)
	assert.Equal(t, "update", failingClient.calls[2].method)
	assert.Equal(t, "9", failingClient.calls[2].update.SequenceId)
}

// failingOpClient fails the call at index failOn and reports sequence on the rest.
type failingOpClient struct {
	mu       sync.Mutex
	calls    []fakeOpCall
	sequence string
	failOn   int
}

func (f *failingOpClient) UpdateOperation(ctx context.Context, parent, resourceKey string, fields []string, body updateOperationRequest) (operationResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	callNum := len(f.calls)
	f.calls = append(f.calls, fakeOpCall{method: "update", parent: parent, resourceKey: resourceKey, update: body, fields: fields})
	if callNum == f.failOn {
		return operationResponse{}, errors.New("injected error")
	}
	return operationResponse{SequenceId: f.sequence}, nil
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
		got, err := DeployActionToSDK(c.action)
		require.NoError(t, err)
		assert.Equal(t, c.want, got)
	}

	// Skip and Undefined never reach a recorder and are rejected.
	_, err := DeployActionToSDK(deployplan.Skip)
	assert.Error(t, err)
	_, err = DeployActionToSDK(deployplan.Undefined)
	assert.Error(t, err)
}
