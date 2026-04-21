package terraform

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	libsfiler "github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/lock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sharedLockerFactory produces Lockers that all share the same underlying
// filer.Filer. The shared filer is what makes the two Apply calls race on
// the same lock file — required for the contention test below.
func sharedLockerFactory(t *testing.T, user string) (lockerFactory, libsfiler.Filer) {
	t.Helper()
	f, err := libsfiler.NewLocalClient(t.TempDir())
	require.NoError(t, err)
	factory := func(_ context.Context, _ *ucm.Ucm, who string) (*lock.Locker, error) {
		holder := who
		if holder == "" {
			holder = user
		}
		return lock.NewLockerWithFiler(holder, "/state", f), nil
	}
	return factory, f
}

func newApplyTerraform(t *testing.T, u *ucm.Ucm, runner *fakeRunner, lf lockerFactory, user string) *Terraform {
	t.Helper()
	workingDir, err := WorkingDir(u)
	require.NoError(t, err)
	return &Terraform{
		ExecPath:      "/stub/terraform",
		WorkingDir:    workingDir,
		Env:           map[string]string{"DATABRICKS_HOST": "https://example.cloud.databricks.com"},
		runnerFactory: newFakeRunnerFactory(runner),
		lockerFactory: lf,
		user:          user,
	}
}

func TestApplyRunsUnderLock(t *testing.T) {
	u, _ := newRenderUcm(t)
	runner := &fakeRunner{}
	factory, _ := sharedLockerFactory(t, "alice")
	tf := newApplyTerraform(t, u, runner, factory, "alice")

	require.NoError(t, tf.Apply(t.Context(), u))
	assert.Equal(t, 1, runner.ApplyCalls)
	// Lock is released on defer — next Apply should succeed too.
	require.NoError(t, tf.Apply(t.Context(), u))
	assert.Equal(t, 2, runner.ApplyCalls)
}

func TestApplyLockContentionReturnsErrLockHeld(t *testing.T) {
	u, _ := newRenderUcm(t)
	factory, _ := sharedLockerFactory(t, "shared")

	// First Apply holds the lock via an ApplyHook that blocks on a channel.
	// While it is blocked, a second Apply runs and should surface *ErrLockHeld.
	hold := make(chan struct{})
	release := make(chan struct{})
	firstRunner := &fakeRunner{
		ApplyHook: func(_ context.Context) {
			close(hold)
			<-release
		},
	}
	firstTf := newApplyTerraform(t, u, firstRunner, factory, "alice")

	errCh := make(chan error, 1)
	go func() {
		errCh <- firstTf.Apply(t.Context(), u)
	}()

	// Wait until the first Apply is holding the lock.
	<-hold

	secondRunner := &fakeRunner{}
	secondTf := newApplyTerraform(t, u, secondRunner, factory, "bob")

	err := secondTf.Apply(t.Context(), u)
	require.Error(t, err)
	var held *lock.ErrLockHeld
	require.True(t, errors.As(err, &held), "expected ErrLockHeld, got %T: %v", err, err)
	assert.Equal(t, "alice", held.Holder)
	assert.Equal(t, 0, secondRunner.ApplyCalls, "second Apply must not invoke the runner")

	// Let the first Apply finish.
	close(release)
	require.NoError(t, <-errCh)
}

func TestApplyUsesPlanPathWhenAvailable(t *testing.T) {
	u, _ := newRenderUcm(t)
	runner := &fakeRunner{PlanHasChanges: true}
	factory, _ := sharedLockerFactory(t, "alice")
	tf := newApplyTerraform(t, u, runner, factory, "alice")

	planResult, err := tf.Plan(t.Context(), u)
	require.NoError(t, err)
	require.NotNil(t, planResult)
	assert.True(t, planResult.HasChanges)

	require.NoError(t, tf.Apply(t.Context(), u))
	require.Len(t, runner.LastApplyOpts, 1, "Apply should have received the plan-path option")

	// The plan path should be the one returned by Plan.
	assert.Equal(t, filepath.Join(tf.WorkingDir, PlanFileName), planResult.PlanPath)
}
