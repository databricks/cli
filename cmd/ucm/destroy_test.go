package ucm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_Destroy_RequiresAutoApprove(t *testing.T) {
	_ = newVerbHarness(t)

	_, _, err := runVerb(t, validFixtureDir(t), "destroy")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "auto-approve")
}

func TestCmd_Destroy_HappyPathWithAutoApprove(t *testing.T) {
	h := newVerbHarness(t)

	stdout, stderr, err := runVerb(t, validFixtureDir(t), "destroy", "--auto-approve")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Destroy OK!")
	assert.Equal(t, 0, h.tf.RenderCalls, "destroy should not call Render")
	assert.Equal(t, 1, h.tf.InitCalls)
	assert.Equal(t, 1, h.tf.DestroyCalls)
	assert.Equal(t, 0, h.tf.ApplyCalls)
}

func TestCmd_Destroy_PropagatesDestroyError(t *testing.T) {
	h := newVerbHarness(t)
	h.tf.DestroyErr = assertSentinel

	_, _, err := runVerb(t, validFixtureDir(t), "destroy", "--auto-approve")

	require.Error(t, err)
	assert.Equal(t, 1, h.tf.DestroyCalls)
}
