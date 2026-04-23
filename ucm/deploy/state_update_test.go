package deploy_test

import (
	"testing"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateUpdateBumpsSeq(t *testing.T) {
	in := &deploy.State{Seq: 5}
	got := deploy.StateUpdate(in)
	assert.Equal(t, 6, got.Seq)
}

func TestStateUpdateStampsCliVersion(t *testing.T) {
	got := deploy.StateUpdate(&deploy.State{})
	assert.Equal(t, build.GetInfo().Version, got.CliVersion)
}

func TestStateUpdateStampsVersion(t *testing.T) {
	got := deploy.StateUpdate(&deploy.State{})
	assert.Equal(t, deploy.StateVersion, got.Version)
}

func TestStateUpdateStampsTimestamp(t *testing.T) {
	before := time.Now().UTC()
	got := deploy.StateUpdate(&deploy.State{})
	after := time.Now().UTC()

	require.False(t, got.Timestamp.IsZero())
	assert.False(t, got.Timestamp.Before(before))
	assert.False(t, got.Timestamp.After(after))
}

func TestStateUpdateAssignsFreshID(t *testing.T) {
	got := deploy.StateUpdate(&deploy.State{})
	assert.NotEqual(t, uuid.Nil, got.ID)
}

func TestStateUpdatePreservesExistingID(t *testing.T) {
	existing := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	got := deploy.StateUpdate(&deploy.State{ID: existing})
	assert.Equal(t, existing, got.ID)
}

func TestStateUpdateDoesNotMutateInput(t *testing.T) {
	in := &deploy.State{Seq: 5}
	_ = deploy.StateUpdate(in)
	assert.Equal(t, 5, in.Seq)
	assert.Equal(t, uuid.Nil, in.ID)
	assert.True(t, in.Timestamp.IsZero())
	assert.Equal(t, "", in.CliVersion)
}
