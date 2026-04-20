package ucm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_Deploy_HappyPath(t *testing.T) {
	h := newVerbHarness(t)

	stdout, stderr, err := runVerb(t, validFixtureDir(t), "deploy")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Deploy OK!")
	assert.Equal(t, 1, h.tf.RenderCalls)
	assert.Equal(t, 1, h.tf.InitCalls)
	assert.Equal(t, 1, h.tf.ApplyCalls)
	assert.Equal(t, 0, h.tf.DestroyCalls)
}

func TestCmd_Deploy_PropagatesApplyError(t *testing.T) {
	h := newVerbHarness(t)
	h.tf.ApplyErr = assertSentinel

	_, _, err := runVerb(t, validFixtureDir(t), "deploy")

	require.Error(t, err)
	assert.Equal(t, 1, h.tf.ApplyCalls)
}
