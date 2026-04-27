package deploy_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runStateUpdate is a small helper that drives the StateUpdate mutator and
// returns the resulting on-disk state. Tests use it to avoid threading
// localDir/decode boilerplate through every assertion.
func runStateUpdate(t *testing.T, f *fixture) deploy.State {
	t.Helper()
	diags := ucm.Apply(t.Context(), f.u, deploy.StateUpdate())
	require.Empty(t, diags)
	return decodeState(t, readLocalUcmStateBytes(t, f.localDir))
}

func TestStateUpdateBumpsSeq(t *testing.T) {
	f := newFixture(t)
	require.NoError(t, deploy.Pull(t.Context(), f.u, f.backend))

	// Seed local with Seq=5 so we can assert Seq+1 after the mutator.
	seedLocalUcmState(t, f.localDir, deploy.State{Version: deploy.StateVersion, Seq: 5})

	got := runStateUpdate(t, f)
	assert.Equal(t, 6, got.Seq)
}

func TestStateUpdateStampsCliVersion(t *testing.T) {
	f := newFixture(t)
	require.NoError(t, deploy.Pull(t.Context(), f.u, f.backend))

	got := runStateUpdate(t, f)
	assert.Equal(t, build.GetInfo().Version, got.CliVersion)
}

func TestStateUpdateStampsVersion(t *testing.T) {
	f := newFixture(t)
	require.NoError(t, deploy.Pull(t.Context(), f.u, f.backend))

	got := runStateUpdate(t, f)
	assert.Equal(t, deploy.StateVersion, got.Version)
}

func TestStateUpdateStampsTimestamp(t *testing.T) {
	f := newFixture(t)
	require.NoError(t, deploy.Pull(t.Context(), f.u, f.backend))

	before := time.Now().UTC()
	got := runStateUpdate(t, f)
	after := time.Now().UTC()

	require.False(t, got.Timestamp.IsZero())
	assert.False(t, got.Timestamp.Before(before))
	assert.False(t, got.Timestamp.After(after))
}

func TestStateUpdatePreservesExistingID(t *testing.T) {
	f := newFixture(t)
	require.NoError(t, deploy.Pull(t.Context(), f.u, f.backend))

	existing := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	seedLocalUcmState(t, f.localDir, deploy.State{Version: deploy.StateVersion, ID: existing})

	got := runStateUpdate(t, f)
	assert.Equal(t, existing, got.ID)
}

func TestStateUpdateAssignsFreshIDWhenZero(t *testing.T) {
	f := newFixture(t)
	require.NoError(t, deploy.Pull(t.Context(), f.u, f.backend))

	// Force the local ID to zero so the mutator has to assign a fresh one.
	seedLocalUcmState(t, f.localDir, deploy.State{Version: deploy.StateVersion, ID: uuid.Nil})

	got := runStateUpdate(t, f)
	assert.NotEqual(t, uuid.Nil, got.ID)
}

func TestStateUpdateFailsWhenLocalMissing(t *testing.T) {
	// No prior Pull → no local ucm-state.json → the mutator surfaces the
	// underlying read error as a diagnostic.
	f := newFixture(t)
	diags := ucm.Apply(t.Context(), f.u, deploy.StateUpdate())
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "read local")
}

// seedLocalUcmState writes a ucm-state.json into the local cache directory.
// Used by tests that want a specific baseline before exercising StateUpdate.
func seedLocalUcmState(t *testing.T, localDir string, s deploy.State) {
	t.Helper()
	require.NoError(t, os.MkdirAll(localDir, 0o755))
	blob, err := json.MarshalIndent(s, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(localDir, deploy.UcmStateFileName), blob, 0o600))
}
