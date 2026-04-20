package deploy_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/databricks/cli/ucm/deploy"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateJSONRoundTrip(t *testing.T) {
	src := deploy.State{
		Version:   deploy.StateVersion,
		Seq:       7,
		ID:        uuid.MustParse("11111111-2222-3333-4444-555555555555"),
		Timestamp: time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC),
	}
	buf, err := json.Marshal(src)
	require.NoError(t, err)

	var got deploy.State
	require.NoError(t, json.Unmarshal(buf, &got))
	assert.Equal(t, src, got)
}

func TestErrStaleStateMessage(t *testing.T) {
	err := &deploy.ErrStaleState{LocalSeq: 3, RemoteSeq: 5}
	msg := err.Error()
	assert.Contains(t, msg, "local seq 3")
	assert.Contains(t, msg, "remote seq 5")
	assert.Contains(t, msg, "pull before pushing")
}

func TestErrStaleStateErrorsAs(t *testing.T) {
	err := error(&deploy.ErrStaleState{LocalSeq: 1, RemoteSeq: 2})
	var target *deploy.ErrStaleState
	require.True(t, errors.As(err, &target))
	assert.Equal(t, 1, target.LocalSeq)
	assert.Equal(t, 2, target.RemoteSeq)
}

func TestErrStaleStateIsReflexive(t *testing.T) {
	// errors.Is on a typed error falls back to == identity via the pointer;
	// this guards against anyone adding a custom Is that breaks that.
	err := &deploy.ErrStaleState{LocalSeq: 1, RemoteSeq: 2}
	assert.True(t, errors.Is(err, err))
}

func TestStateUnmarshalEmptyTimestamp(t *testing.T) {
	// Tolerate blobs written by tests or future CLIs that omit Timestamp.
	var s deploy.State
	require.NoError(t, json.Unmarshal([]byte(`{"version":1,"seq":0,"id":"00000000-0000-0000-0000-000000000000","timestamp":"0001-01-01T00:00:00Z"}`), &s))
	assert.Equal(t, 1, s.Version)
	assert.Equal(t, 0, s.Seq)
	assert.True(t, s.Timestamp.IsZero())
}

func TestStateMarshalIndented(t *testing.T) {
	// Sanity-check that an indented marshal reads back identical. This is
	// the exact shape writeLocalState persists.
	src := deploy.State{Version: 1, Seq: 2, ID: uuid.New(), Timestamp: time.Now().UTC()}
	buf, err := json.MarshalIndent(src, "", "  ")
	require.NoError(t, err)

	var got deploy.State
	require.NoError(t, json.NewDecoder(bytes.NewReader(buf)).Decode(&got))
	assert.Equal(t, src.Seq, got.Seq)
	assert.Equal(t, src.Version, got.Version)
	assert.Equal(t, src.ID, got.ID)
}
