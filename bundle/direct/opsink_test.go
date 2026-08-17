package direct

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUploader records the uploads it receives and optionally blocks until block is
// closed, so a test can hold the uploader and observe what coalesces behind it.
type fakeUploader struct {
	block   chan struct{}
	started chan string
	err     error

	mu            sync.Mutex
	uploads       []string
	actions       map[string]bundledeployments.OperationActionType
	resourceIDs   map[string]string
	statuses      map[string]bundledeployments.OperationStatus
	errorMessages map[string]string
}

func (f *fakeUploader) upload(ctx context.Context, resourceKey string, op recordedOperation) error {
	if f.started != nil {
		f.started <- resourceKey
	}
	if f.block != nil {
		<-f.block
	}

	f.mu.Lock()
	f.uploads = append(f.uploads, resourceKey+"="+string(op.state))
	if f.actions == nil {
		f.actions = map[string]bundledeployments.OperationActionType{}
		f.resourceIDs = map[string]string{}
		f.statuses = map[string]bundledeployments.OperationStatus{}
		f.errorMessages = map[string]string{}
	}
	f.actions[resourceKey] = op.action
	f.resourceIDs[resourceKey] = op.resourceID
	f.statuses[resourceKey] = op.status
	f.errorMessages[resourceKey] = op.errorMessage
	f.mu.Unlock()

	return f.err
}

func (f *fakeUploader) recorded() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.uploads...)
}

func (f *fakeUploader) actionFor(resourceKey string) bundledeployments.OperationActionType {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.actions[resourceKey]
}

func (f *fakeUploader) resourceIDFor(resourceKey string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.resourceIDs[resourceKey]
}

func (f *fakeUploader) statusFor(resourceKey string) bundledeployments.OperationStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.statuses[resourceKey]
}

func (f *fakeUploader) errorMessageFor(resourceKey string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.errorMessages[resourceKey]
}

// envelope builds the serialized RecordedState the state DB hands the sink.
func envelope(t *testing.T, name string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(dstate.RecordedState{State: json.RawMessage(`{"name":"` + name + `"}`)})
	require.NoError(t, err)
	return raw
}

func recordState(t *testing.T, s *operationSink, resourceKey, name string) {
	t.Helper()
	s.RecordOperation(t.Context(), resourceKey, dstate.OperationInfo{Action: deployplan.Update}, "id-1", envelope(t, name))
}

func TestOperationSinkUploadsEachOperation(t *testing.T) {
	f := &fakeUploader{}
	s := newOperationSink(t.Context(), f, 32)

	for i := range 20 {
		recordState(t, s, "resources.jobs.job"+strconv.Itoa(i), "n")
	}
	require.NoError(t, s.close())

	assert.Len(t, f.recorded(), 20)
}

func TestOperationSinkCoalescesWritesBehindAnUpload(t *testing.T) {
	// Hold the uploader on the first write so the two behind it pile up. They carry
	// the resource's full state, so only the newest needs to go: the resource costs
	// two requests rather than three.
	f := &fakeUploader{block: make(chan struct{}), started: make(chan string, 2)}
	s := newOperationSink(t.Context(), f, 32)

	recordState(t, s, "resources.jobs.foo", "v1")
	assert.Equal(t, "resources.jobs.foo", <-f.started)

	recordState(t, s, "resources.jobs.foo", "v2")
	recordState(t, s, "resources.jobs.foo", "v3")

	close(f.block)
	require.NoError(t, s.close())

	assert.Equal(t, []string{
		`resources.jobs.foo={"state":{"name":"v1"}}`,
		`resources.jobs.foo={"state":{"name":"v3"}}`,
	}, f.recorded())
}

func TestOperationSinkCoalescedFailureKeepsTheStateItReplaces(t *testing.T) {
	// A create writes state and then fails waiting for the resource to come up. If
	// the failure catches the write before it is uploaded, it must not replace that
	// state with its own emptiness: a resource recorded without state is dropped from
	// the deployment, so the next plan would create it a second time.
	f := &fakeUploader{block: make(chan struct{}), started: make(chan string, 2)}
	s := newOperationSink(t.Context(), f, 32)

	// Occupy the uploader with an unrelated resource, so the two writes below both
	// land in pending and coalesce.
	recordState(t, s, "resources.jobs.busy", "v1")
	assert.Equal(t, "resources.jobs.busy", <-f.started)

	s.RecordOperation(t.Context(), "resources.job_runs.my_run", dstate.OperationInfo{Action: deployplan.Create}, "run-1", envelope(t, "the run"))
	// priorState and priorID are empty: the resource was created in this deploy, so
	// there is no pre-deploy record to report.
	s.recordFailure(t.Context(), "resources.job_runs.my_run", deployplan.Create, "", nil, errors.New("run did not succeed: FAILED"))

	close(f.block)
	require.NoError(t, s.close())

	assert.Equal(t, []string{
		`resources.jobs.busy={"state":{"name":"v1"}}`,
		`resources.job_runs.my_run={"state":{"name":"the run"}}`,
	}, f.recorded())
	assert.Equal(t, "run-1", f.resourceIDFor("resources.job_runs.my_run"))
	assert.Equal(t,
		bundledeployments.OperationStatusOperationStatusFailed,
		f.statusFor("resources.job_runs.my_run"))
	assert.Equal(t, "run did not succeed: FAILED", f.errorMessageFor("resources.job_runs.my_run"))
}

func TestOperationSinkCoalescedDeleteStillClearsState(t *testing.T) {
	// A delete legitimately carries no state, and coalescing must let it through: the
	// resource is gone, and keeping the state it replaces would leave it listed.
	f := &fakeUploader{block: make(chan struct{}), started: make(chan string, 2)}
	s := newOperationSink(t.Context(), f, 32)

	recordState(t, s, "resources.jobs.busy", "v1")
	assert.Equal(t, "resources.jobs.busy", <-f.started)

	s.RecordOperation(t.Context(), "resources.jobs.foo", dstate.OperationInfo{Action: deployplan.Update}, "id-1", envelope(t, "before"))
	s.RecordOperation(t.Context(), "resources.jobs.foo", dstate.OperationInfo{Action: deployplan.Delete}, "id-1", nil)

	close(f.block)
	require.NoError(t, s.close())

	assert.Equal(t, []string{
		`resources.jobs.busy={"state":{"name":"v1"}}`,
		`resources.jobs.foo=`,
	}, f.recorded())
	assert.Equal(t,
		bundledeployments.OperationActionTypeOperationActionTypeDelete,
		f.actionFor("resources.jobs.foo"))
}

func TestOperationSinkRecordDuringUploadIsStillUploaded(t *testing.T) {
	f := &fakeUploader{block: make(chan struct{}), started: make(chan string, 2)}
	s := newOperationSink(t.Context(), f, 32)

	recordState(t, s, "resources.jobs.foo", "v1")
	assert.Equal(t, "resources.jobs.foo", <-f.started)

	// The uploader has taken this key off pending and is uploading it right now.
	recordState(t, s, "resources.jobs.foo", "v2")

	close(f.block)
	require.NoError(t, s.close())

	// Two uploads, in order: an in-flight request cannot be recalled, so v2 goes up
	// after v1 rather than replacing it. The service ends up with the newest state.
	assert.Equal(t, []string{
		`resources.jobs.foo={"state":{"name":"v1"}}`,
		`resources.jobs.foo={"state":{"name":"v2"}}`,
	}, f.recorded())
	assert.Empty(t, s.pending)
}

func TestOperationSinkReturnsUploadError(t *testing.T) {
	uploadErr := errors.New("boom")
	f := &fakeUploader{err: uploadErr}
	s := newOperationSink(t.Context(), f, 32)

	recordState(t, s, "resources.jobs.foo", "v1")

	err := s.close()
	require.Error(t, err)
	assert.ErrorIs(t, err, uploadErr)
	assert.ErrorContains(t, err, "resources.jobs.foo")
}

func TestOperationSinkKeepsRecordingAfterUploadError(t *testing.T) {
	// One failed upload must not drop the records for everything behind it, so DMS
	// ends up as close to reality as it can get.
	f := &fakeUploader{err: errors.New("boom")}
	s := newOperationSink(t.Context(), f, 32)

	recordState(t, s, "resources.jobs.foo", "v1")
	recordState(t, s, "resources.jobs.bar", "v1")

	require.Error(t, s.close())
	assert.Len(t, f.recorded(), 2)
}

func TestOperationSinkFirstErrIsWhatStopsTheDeploy(t *testing.T) {
	f := &fakeUploader{err: errors.New("boom")}
	s := newOperationSink(t.Context(), f, 32)

	assert.NoError(t, s.firstErr())

	recordState(t, s, "resources.jobs.foo", "v1")
	require.Error(t, s.close())

	// Reported after the fact too, so the caller can check once more before it
	// completes the version.
	assert.Error(t, s.firstErr())
}

func TestOperationSinkDropsUnsupportedAction(t *testing.T) {
	f := &fakeUploader{}
	s := newOperationSink(t.Context(), f, 32)

	// Skip never reaches a sink; it is dropped with a warning rather than failing a
	// resource that deployed fine.
	s.RecordOperation(t.Context(), "resources.jobs.foo", dstate.OperationInfo{Action: deployplan.Skip}, "id-1", nil)
	require.NoError(t, s.close())

	assert.Empty(t, f.recorded())
}

func TestOperationSinkDropsOversizedState(t *testing.T) {
	f := &fakeUploader{}
	s := newOperationSink(t.Context(), f, 32)

	s.RecordOperation(t.Context(), "resources.jobs.foo", dstate.OperationInfo{Action: deployplan.Create}, "id-1", envelope(t, strings.Repeat("x", maxOperationStateSize)))
	require.NoError(t, s.close())

	assert.Empty(t, f.recorded())
}

func TestOperationSinkCloseIsIdempotent(t *testing.T) {
	f := &fakeUploader{}
	s := newOperationSink(t.Context(), f, 32)

	recordState(t, s, "resources.jobs.foo", "v1")

	require.NoError(t, s.close())
	require.NoError(t, s.close())
	assert.Len(t, f.recorded(), 1)
}

func TestNilOperationSinkIsNoOp(t *testing.T) {
	var s *operationSink
	s.RecordOperation(t.Context(), "resources.jobs.foo", dstate.OperationInfo{Action: deployplan.Create}, "id-1", nil)
	s.recordFailure(t.Context(), "resources.jobs.foo", deployplan.Create, "id-1", nil, errors.New("boom"))
	assert.NoError(t, s.firstErr())
	assert.NoError(t, s.close())
}

func TestNewOperationSinkNilUploaderIsNil(t *testing.T) {
	// Recording off: the sink is nil so the state DB's nil check leaves it unset.
	assert.Nil(t, newOperationSink(t.Context(), nil, 32))
}
