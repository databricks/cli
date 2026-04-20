package terraform

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanReturnsResultWithChanges(t *testing.T) {
	u, _ := newRenderUcm(t)
	runner := &fakeRunner{PlanHasChanges: true}
	tf, _ := newInitTerraform(t)
	tf.WorkingDir, _ = WorkingDir(u)
	tf.runnerFactory = newFakeRunnerFactory(runner)

	result, err := tf.Plan(t.Context(), u)
	require.NoError(t, err)
	assert.True(t, result.HasChanges)
	assert.Equal(t, filepath.Join(tf.WorkingDir, PlanFileName), result.PlanPath)
	assert.Equal(t, "plan has changes", result.Summary)
	assert.Equal(t, 1, runner.PlanCalls)
}

func TestPlanReturnsResultWithoutChanges(t *testing.T) {
	u, _ := newRenderUcm(t)
	runner := &fakeRunner{PlanHasChanges: false}
	tf, _ := newInitTerraform(t)
	tf.WorkingDir, _ = WorkingDir(u)
	tf.runnerFactory = newFakeRunnerFactory(runner)

	result, err := tf.Plan(t.Context(), u)
	require.NoError(t, err)
	assert.False(t, result.HasChanges)
	assert.Equal(t, "no changes", result.Summary)
}

func TestPlanInitsFirst(t *testing.T) {
	u, _ := newRenderUcm(t)
	runner := &fakeRunner{}
	tf, _ := newInitTerraform(t)
	tf.WorkingDir, _ = WorkingDir(u)
	tf.runnerFactory = newFakeRunnerFactory(runner)

	_, err := tf.Plan(t.Context(), u)
	require.NoError(t, err)
	assert.Equal(t, 1, runner.InitCalls, "Plan must call Init first")
}
