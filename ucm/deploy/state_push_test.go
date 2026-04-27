package deploy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	libsfiler "github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/lock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// applyStateUpdate runs the StateUpdate mutator against f.u so subsequent
// Push calls operate on a bumped local state — Push no longer bumps inline.
func applyStateUpdate(t *testing.T, ctx context.Context, f *fixture) {
	t.Helper()
	diags := ucm.Apply(ctx, f.u, deploy.StateUpdate())
	require.Empty(t, diags)
}

// readRemoteState round-trips the remote ucm-state.json for assertion
// purposes. Returns nil if the remote hasn't been written yet.
func readRemoteState(t *testing.T, f libsfiler.Filer) *deploy.State {
	t.Helper()
	rc, err := f.Read(t.Context(), deploy.UcmStateFileName)
	if err != nil {
		return nil
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	var s deploy.State
	require.NoError(t, json.Unmarshal(data, &s))
	return &s
}

func TestPushFirstWriteAfterPull(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	// First Pull on an empty remote establishes a Seq=0 local; StateUpdate
	// bumps it to 1 in the local cache. Push only mirrors that to remote.
	require.NoError(t, deploy.Pull(ctx, f.u, f.backend))
	applyStateUpdate(t, ctx, f)

	// Drop a tfstate blob locally so Push has something to upload.
	writeLocalTf(t, f, []byte(`{"tf":"v1"}`))

	require.NoError(t, deploy.Push(ctx, f.u, f.backend))

	remote := readRemoteState(t, f.remote)
	require.NotNil(t, remote)
	assert.Equal(t, 1, remote.Seq)

	// Local Seq matches remote because StateUpdate already advanced it.
	localData, err := os.ReadFile(filepath.Join(f.localDir, deploy.UcmStateFileName))
	require.NoError(t, err)
	var local deploy.State
	require.NoError(t, json.Unmarshal(localData, &local))
	assert.Equal(t, 1, local.Seq)
}

func TestPushPullRoundTripIsIdentical(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	require.NoError(t, deploy.Pull(ctx, f.u, f.backend))
	applyStateUpdate(t, ctx, f)
	tfBlob := []byte(`{"tf":"roundtrip"}`)
	writeLocalTf(t, f, tfBlob)
	require.NoError(t, deploy.Push(ctx, f.u, f.backend))

	// Simulate a second clone of the project: wipe the local cache and
	// re-Pull. The pulled tfstate must byte-for-byte match what was Pushed.
	require.NoError(t, os.RemoveAll(f.localDir))
	require.NoError(t, deploy.Pull(ctx, f.u, f.backend))

	got, err := os.ReadFile(deploy.LocalTfStatePath(f.u))
	require.NoError(t, err)
	assert.Equal(t, tfBlob, got)

	// Seq must round-trip too.
	state, err := os.ReadFile(filepath.Join(f.localDir, deploy.UcmStateFileName))
	require.NoError(t, err)
	var s deploy.State
	require.NoError(t, json.Unmarshal(state, &s))
	assert.Equal(t, 1, s.Seq)
}

func TestPushDetectsStaleState(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	// Client A pulls and prepares to push (Seq goes 0→1 locally).
	require.NoError(t, deploy.Pull(ctx, f.u, f.backend))
	applyStateUpdate(t, ctx, f)

	// Meanwhile, client B advances the remote Seq out from under A.
	seedRemoteUcmState(t, ctx, f.remote, deploy.State{Version: deploy.StateVersion, Seq: 5})

	err := deploy.Push(ctx, f.u, f.backend)
	require.Error(t, err)

	var stale *deploy.ErrStaleState
	require.ErrorAs(t, err, &stale)
	assert.Equal(t, 1, stale.LocalSeq)
	assert.Equal(t, 5, stale.RemoteSeq)
}

func TestPushSecondPushAfterRemoteBumpFails(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	require.NoError(t, deploy.Pull(ctx, f.u, f.backend))
	applyStateUpdate(t, ctx, f)
	require.NoError(t, deploy.Push(ctx, f.u, f.backend))

	// Peer bumps remote Seq.
	seedRemoteUcmState(t, ctx, f.remote, deploy.State{Version: deploy.StateVersion, Seq: 99})

	err := deploy.Push(ctx, f.u, f.backend)
	require.Error(t, err)
	var stale *deploy.ErrStaleState
	require.ErrorAs(t, err, &stale)
	assert.Equal(t, 99, stale.RemoteSeq)
}

func TestPushWithoutPriorPullFails(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	// No Pull before Push → no local ucm-state.json → read error.
	err := deploy.Push(ctx, f.u, f.backend)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read local")
}

func TestPushReleasesLockOnStaleError(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	require.NoError(t, deploy.Pull(ctx, f.u, f.backend))

	// Force a stale-state failure.
	seedRemoteUcmState(t, ctx, f.remote, deploy.State{Version: deploy.StateVersion, Seq: 5})

	err := deploy.Push(ctx, f.u, f.backend)
	require.Error(t, err)

	// Lock must be released after the failure.
	contender := lock.NewLockerWithFiler("bob@example.com", ".", f.remote)
	require.NoError(t, contender.Acquire(ctx, false))
	require.NoError(t, contender.Release(ctx, lock.GoalDeploy))
}

func TestPushReleasesLockOnSuccess(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	require.NoError(t, deploy.Pull(ctx, f.u, f.backend))
	applyStateUpdate(t, ctx, f)
	require.NoError(t, deploy.Push(ctx, f.u, f.backend))

	contender := lock.NewLockerWithFiler("bob@example.com", ".", f.remote)
	require.NoError(t, contender.Acquire(ctx, false))
	require.NoError(t, contender.Release(ctx, lock.GoalDeploy))
}

func TestPushFailsWhenLockHeldByPeer(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	require.NoError(t, deploy.Pull(ctx, f.u, f.backend))

	peer := lock.NewLockerWithFiler("bob@example.com", ".", f.remote)
	require.NoError(t, peer.Acquire(ctx, false))
	defer peer.Release(ctx, lock.GoalDeploy)

	err := deploy.Push(ctx, f.u, f.backend)
	require.Error(t, err)
	var held *lock.ErrLockHeld
	require.ErrorAs(t, err, &held)
}

func TestPushMirrorsTfstateBytesExactly(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	require.NoError(t, deploy.Pull(ctx, f.u, f.backend))
	applyStateUpdate(t, ctx, f)
	tfBlob := bytes.Repeat([]byte{0xAB}, 2048)
	writeLocalTf(t, f, tfBlob)

	require.NoError(t, deploy.Push(ctx, f.u, f.backend))

	rc, err := f.remote.Read(ctx, deploy.TfStateFileName)
	require.NoError(t, err)
	defer rc.Close()
	remoteTf, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, tfBlob, remoteTf)
}

func TestPushNilUcmRequiresBackend(t *testing.T) {
	ctx := t.Context()
	err := deploy.Push(ctx, nil, deploy.Backend{})
	require.Error(t, err)

	f := newFixture(t)
	err = deploy.Push(ctx, f.u, deploy.Backend{})
	require.Error(t, err)
}

func TestPushErrStaleStateIsAssignableToPointer(t *testing.T) {
	// Guards against regressions in ErrStaleState pointer receiver.
	var stale *deploy.ErrStaleState
	var e error = &deploy.ErrStaleState{LocalSeq: 1, RemoteSeq: 2}
	require.True(t, errors.As(e, &stale))
}
