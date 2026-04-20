package deploy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	libsfiler "github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/deploy"
	ucmfiler "github.com/databricks/cli/ucm/deploy/filer"
	"github.com/databricks/cli/ucm/deploy/lock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixture bundles the inputs every pull/push test needs: a local project dir
// for the Ucm, and two filers for the remote (state + lock) sharing a single
// temp dir so a second client can contend the same logical state root.
type fixture struct {
	t        *testing.T
	projDir  string
	remote   libsfiler.Filer
	backend  deploy.Backend
	u        *ucm.Ucm
	localDir string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	projDir := t.TempDir()
	remoteDir := t.TempDir()

	local, err := libsfiler.NewLocalClient(remoteDir)
	require.NoError(t, err)

	u := &ucm.Ucm{RootPath: projDir}
	u.Config.Ucm = config.Ucm{Name: "test", Target: "dev"}

	return &fixture{
		t:       t,
		projDir: projDir,
		remote:  local,
		backend: deploy.Backend{
			StateFiler: ucmfiler.NewStateFilerFromFiler(local),
			LockFiler:  local,
			User:       "alice@example.com",
		},
		u:        u,
		localDir: filepath.Join(projDir, filepath.FromSlash(deploy.LocalCacheDir), "dev"),
	}
}

// readLocalUcmStateBytes reads the on-disk ucm-state.json from the local
// cache directory. Tests use this instead of exposing readLocalState from
// the production package.
func readLocalUcmStateBytes(t *testing.T, localDir string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(localDir, deploy.UcmStateFileName))
	require.NoError(t, err)
	return data
}

func decodeState(t *testing.T, data []byte) deploy.State {
	t.Helper()
	var s deploy.State
	require.NoError(t, json.NewDecoder(bytes.NewReader(data)).Decode(&s))
	return s
}

// seedRemoteUcmState writes a ucm-state.json at the remote root. Used to
// simulate a remote produced by a peer client (e.g. a contending Push).
func seedRemoteUcmState(t *testing.T, ctx context.Context, remote libsfiler.Filer, s deploy.State) {
	t.Helper()
	blob, err := json.Marshal(s)
	require.NoError(t, err)
	require.NoError(t, remote.Write(ctx, deploy.UcmStateFileName, bytes.NewReader(blob), libsfiler.OverwriteIfExists, libsfiler.CreateParentDirectories))
}

func TestPullFirstRunInitializesFreshLocal(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	require.NoError(t, deploy.Pull(ctx, f.u, f.backend))

	got := decodeState(t, readLocalUcmStateBytes(t, f.localDir))
	assert.Equal(t, 0, got.Seq)
	assert.Equal(t, deploy.StateVersion, got.Version)

	// tfstate must NOT be mirrored locally: the signal to downstream
	// phases that this is a first-run.
	_, err := os.Stat(filepath.Join(f.localDir, deploy.TfStateFileName))
	assert.True(t, os.IsNotExist(err), "unexpected local tfstate on first-run: %v", err)
}

func TestPullMirrorsRemoteStateAndTfstate(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	remoteState := deploy.State{Version: deploy.StateVersion, Seq: 4}
	seedRemoteUcmState(t, ctx, f.remote, remoteState)
	require.NoError(t, f.remote.Write(ctx, deploy.TfStateFileName, bytes.NewReader([]byte(`{"terraform":"blob"}`)), libsfiler.OverwriteIfExists, libsfiler.CreateParentDirectories))

	require.NoError(t, deploy.Pull(ctx, f.u, f.backend))

	got := decodeState(t, readLocalUcmStateBytes(t, f.localDir))
	assert.Equal(t, 4, got.Seq)

	tfData, err := os.ReadFile(filepath.Join(f.localDir, deploy.TfStateFileName))
	require.NoError(t, err)
	assert.Equal(t, `{"terraform":"blob"}`, string(tfData))
}

func TestPullRejectsFutureVersion(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	seedRemoteUcmState(t, ctx, f.remote, deploy.State{Version: deploy.StateVersion + 1, Seq: 1})

	err := deploy.Pull(ctx, f.u, f.backend)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remote version")
}

func TestPullReleasesLockOnSuccess(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	require.NoError(t, deploy.Pull(ctx, f.u, f.backend))

	// A contending client must be able to acquire the lock immediately
	// after Pull returns — i.e. the defer ran.
	contender := lock.NewLockerWithFiler("bob@example.com", ".", f.remote)
	require.NoError(t, contender.Acquire(ctx, false))
	require.NoError(t, contender.Release(ctx, lock.GoalDeploy))
}

func TestPullReleasesLockOnError(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	// Future-version remote causes Pull to fail; the lock must still be
	// released for the next caller.
	seedRemoteUcmState(t, ctx, f.remote, deploy.State{Version: deploy.StateVersion + 99})

	err := deploy.Pull(ctx, f.u, f.backend)
	require.Error(t, err)

	contender := lock.NewLockerWithFiler("bob@example.com", ".", f.remote)
	require.NoError(t, contender.Acquire(ctx, false))
	require.NoError(t, contender.Release(ctx, lock.GoalDeploy))
}

func TestPullFailsWhenLockAlreadyHeld(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	// Peer grabs the lock first.
	peer := lock.NewLockerWithFiler("bob@example.com", ".", f.remote)
	require.NoError(t, peer.Acquire(ctx, false))
	defer peer.Release(ctx, lock.GoalDeploy)

	err := deploy.Pull(ctx, f.u, f.backend)
	require.Error(t, err)

	var held *lock.ErrLockHeld
	require.ErrorAs(t, err, &held)
	assert.Equal(t, "bob@example.com", held.Holder)
}

func TestPullRequiresBackend(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t)

	err := deploy.Pull(ctx, f.u, deploy.Backend{})
	require.Error(t, err)
}

func TestPullNilUcm(t *testing.T) {
	ctx := t.Context()
	err := deploy.Pull(ctx, nil, deploy.Backend{})
	require.Error(t, err)
}
