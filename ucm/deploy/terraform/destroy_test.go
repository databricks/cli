package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDestroyRunsUnderLock(t *testing.T) {
	u, _ := newRenderUcm(t)
	runner := &fakeRunner{}
	factory, _ := sharedLockerFactory(t, "alice")
	tf := newApplyTerraform(t, u, runner, factory, "alice")

	require.NoError(t, tf.Destroy(t.Context(), u, false))
	assert.Equal(t, 1, runner.DestroyCalls)

	// Lock released on defer: a second Destroy should succeed and re-run.
	require.NoError(t, tf.Destroy(t.Context(), u, false))
	assert.Equal(t, 2, runner.DestroyCalls)
}

func TestDestroyPropagatesRunnerError(t *testing.T) {
	u, _ := newRenderUcm(t)
	runner := &fakeRunner{DestroyErr: assert.AnError}
	factory, _ := sharedLockerFactory(t, "alice")
	tf := newApplyTerraform(t, u, runner, factory, "alice")

	err := tf.Destroy(t.Context(), u, false)
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}
