package dms

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// What a recorded deploy puts on the wire - the version it claims, the operations it stages,
// the completion it reports - is asserted end to end by acceptance/bundle/dms. What is left
// here is what only an injected API error or a disabled recording can reach.

// fakeDMS answers the generated calls a BufferedClient makes. It embeds the SDK interface so it
// satisfies it while only overriding those.
type fakeDMS struct {
	bundledeployments.BundleDeploymentsInterface

	getDeployment func(id string) (*bundledeployments.Deployment, error)

	created   []bundledeployments.CreateDeploymentRequest
	completed []bundledeployments.CompleteVersionRequest
	deleted   []string

	// raw captures what the hand-written half of the client sent; see fakeRaw.
	raw *fakeRaw
}

func (f *fakeDMS) CreateDeployment(ctx context.Context, req bundledeployments.CreateDeploymentRequest) (*bundledeployments.Deployment, error) {
	f.created = append(f.created, req)
	return &bundledeployments.Deployment{Name: "deployments/new-id"}, nil
}

func (f *fakeDMS) GetDeployment(ctx context.Context, req bundledeployments.GetDeploymentRequest) (*bundledeployments.Deployment, error) {
	return f.getDeployment(req.Name)
}

func (f *fakeDMS) CompleteVersion(ctx context.Context, req bundledeployments.CompleteVersionRequest) (*bundledeployments.Version, error) {
	f.completed = append(f.completed, req)
	return &bundledeployments.Version{}, nil
}

func (f *fakeDMS) DeleteDeployment(ctx context.Context, req bundledeployments.DeleteDeploymentRequest) error {
	f.deleted = append(f.deleted, req.Name)
	return nil
}

func (f *fakeDMS) Heartbeat(ctx context.Context, req bundledeployments.HeartbeatRequest) (*bundledeployments.HeartbeatResponse, error) {
	return &bundledeployments.HeartbeatResponse{}, nil
}

// deploymentAt answers GetDeployment with a deployment whose last version is lastVersion.
func deploymentAt(lastVersion string) func(string) (*bundledeployments.Deployment, error) {
	return func(name string) (*bundledeployments.Deployment, error) {
		return &bundledeployments.Deployment{Name: name, LastVersionId: lastVersion}, nil
	}
}

// testRecording records the stored deployment through f, failing CreateVersion with
// versionErr when set.
func testClient(t *testing.T, f *fakeDMS, versionType VersionType, versionErr error) *BufferedClient {
	t.Helper()
	f.raw = newFakeRaw("1")
	f.raw.versionErr = versionErr
	c, err := NewBufferedClient(Options{
		Client:        &Client{Service: f, raw: f.raw},
		DeploymentID:  "stored-id",
		LastVersionID: "4",
		VersionType:   versionType,
	})
	require.NoError(t, err)
	return c
}

func TestRecordingStartReportsWhatTheServiceRefused(t *testing.T) {
	aborted := &apierr.APIError{StatusCode: 409, ErrorCode: "ABORTED"}
	exhausted := &apierr.APIError{StatusCode: 429, ErrorCode: "RESOURCE_EXHAUSTED"}

	tests := []struct {
		name         string
		versionErr   error
		wantMessages []string
		wantCause    error
	}{
		{
			name:         "another deploy claimed the version number",
			versionErr:   aborted,
			wantMessages: []string{"another deploy already claimed version 5", "try again"},
			wantCause:    aborted,
		},
		{
			name:         "the bundle stages more operations than a version holds",
			versionErr:   exhausted,
			wantMessages: []string{"this bundle deploys 1 resources"},
			wantCause:    exhausted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeDMS{}
			r := testClient(t, f, VersionTypeDeploy, tt.versionErr)

			err := r.Start(t.Context(), []StagedOperation{{ResourceKey: "jobs.foo"}})

			require.Error(t, err)
			for _, want := range tt.wantMessages {
				assert.ErrorContains(t, err, want)
			}
			if tt.wantCause != nil {
				// Wrapped, so a caller can still match on what the service said.
				assert.ErrorIs(t, err, tt.wantCause)
			}
			assert.Empty(t, f.created, "the deployment already exists")
		})
	}
}

func TestRecordingFinishDeletesTheDeploymentOnlyForACompletedDestroy(t *testing.T) {
	// A failed destroy leaves resources behind, so its deployment has to stay for the next
	// deploy to find.
	tests := []struct {
		name        string
		versionType VersionType
		success     bool
		wantDeleted bool
	}{
		{name: "a destroy that completed", versionType: VersionTypeDestroy, success: true, wantDeleted: true},
		{name: "a destroy that failed", versionType: VersionTypeDestroy, success: false},
		{name: "a deploy", versionType: VersionTypeDeploy, success: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeDMS{}
			r := testClient(t, f, tt.versionType, nil)

			err := r.Start(t.Context(), nil)
			require.NoError(t, err)
			require.NoError(t, r.Close(t.Context(), tt.success))

			wantReason := bundledeployments.VersionCompleteVersionCompleteSuccess
			if !tt.success {
				wantReason = bundledeployments.VersionCompleteVersionCompleteFailure
			}
			require.Len(t, f.completed, 1)
			assert.Equal(t, wantReason, f.completed[0].CompletionReason)

			if tt.wantDeleted {
				assert.Equal(t, []string{"deployments/stored-id"}, f.deleted)
			} else {
				assert.Empty(t, f.deleted)
			}
		})
	}
}

func TestNilClientIsNoOp(t *testing.T) {
	// What a bundle that does not record deployment history gets, so callers call through
	// without checking.
	var r *BufferedClient

	require.NoError(t, r.Start(t.Context(), []StagedOperation{{ResourceKey: "jobs.foo"}}))
	require.NoError(t, r.Close(t.Context(), true))
	require.NoError(t, r.Drain())

	assert.Empty(t, r.DeploymentID())
	assert.Zero(t, r.Version())
	assert.NoError(t, r.Err())
}

// What the service ends up holding is asserted by acceptance/bundle/dms. What is left here is
// what a deploy cannot reach: the queue's coalescing and the size limit. No test here goes
// through the API - the queue is driven directly, so nothing waits on a request.

// queued builds a client whose buffer nothing drains, so a test drives record and take itself.
func queued() *BufferedClient {
	return &BufferedClient{
		queue:   make(chan string, bufferedOperations),
		pending: make(map[string]OperationUpdate),
	}
}

func stateUpdate(t *testing.T, name string) OperationUpdate {
	t.Helper()
	update, err := NewStateUpdate("id-1", json.RawMessage(`{"state":{"name":"`+name+`"}}`), false)
	require.NoError(t, err)
	return update
}

func TestBufferCoalescesWhileAKeyIsPending(t *testing.T) {
	s := queued()

	// Two writes for one resource with nothing draining. They carry the resource's full
	// state, so only the newest needs to go: one slot in the queue, not two.
	s.record("resources.jobs.foo", stateUpdate(t, "v1"))
	s.record("resources.jobs.foo", stateUpdate(t, "v2"))

	assert.Len(t, s.queue, 1)
	update, ok := s.take("resources.jobs.foo")
	require.True(t, ok)
	assert.JSONEq(t, `{"state":{"name":"v2"}}`, string(update.State))

	// Taken means a request has it, and an in-flight request cannot be recalled, so the next
	// write gets its own slot rather than joining it.
	s.record("resources.jobs.foo", stateUpdate(t, "v3"))
	assert.Len(t, s.queue, 2)
	update, ok = s.take("resources.jobs.foo")
	require.True(t, ok)
	assert.JSONEq(t, `{"state":{"name":"v3"}}`, string(update.State))
}

func TestBufferFailsOnOversizedState(t *testing.T) {
	// The service will not take a state this large, so the resource cannot be recorded.
	// Failing here says so, where reporting nothing would leave DMS without the resource
	// and the next plan would create it again.
	s := queued()

	s.RecordOperation(t.Context(), "resources.jobs.foo", false, "id-1", json.RawMessage(strings.Repeat("x", maxStateSize+1)))

	assert.ErrorContains(t, s.Err(), "exceeds the 65536 byte limit")
	assert.Empty(t, s.queue)
	assert.Empty(t, s.pending)
}

func TestDrainIsIdempotent(t *testing.T) {
	// Nothing recorded, so no transport is needed: this is the second drain, which must not
	// panic on an already closed queue.
	s := queued()
	s.done = make(chan struct{})
	s.stopQueue = sync.OnceFunc(func() { close(s.queue) })
	go s.run(t.Context())

	require.NoError(t, s.Drain())
	require.NoError(t, s.Drain())
}

func TestRecordBeforeStartIsAnError(t *testing.T) {
	// The buffer only exists once a version does, so a write before Start is a wiring bug.
	// Reported rather than dropped: the record would otherwise be lost silently.
	c := &BufferedClient{}

	c.RecordOperation(t.Context(), "resources.jobs.foo", false, "id-1", json.RawMessage(`{"state":{}}`))

	assert.ErrorContains(t, c.Err(), "before the deployment version was created")
}
