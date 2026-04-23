package terraform

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/lock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newImportTerraform(t *testing.T, u *ucm.Ucm, runner *fakeRunner, lf lockerFactory, user string) *Terraform {
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

func TestImportRunsUnderLock(t *testing.T) {
	u, _ := newRenderUcm(t)
	runner := &fakeRunner{}
	factory, _ := sharedLockerFactory(t, "alice")
	tf := newImportTerraform(t, u, runner, factory, "alice")

	require.NoError(t, tf.Import(t.Context(), u, "databricks_catalog.sales", "sales_prod"))
	assert.Equal(t, 1, runner.ImportCalls)
	assert.Equal(t, "databricks_catalog.sales", runner.LastImportAddress)
	assert.Equal(t, "sales_prod", runner.LastImportId)
	// Lock is released on defer — next Import should succeed too.
	require.NoError(t, tf.Import(t.Context(), u, "databricks_catalog.sales", "sales_prod"))
	assert.Equal(t, 2, runner.ImportCalls)
}

func TestImportLockContentionReturnsErrLockHeld(t *testing.T) {
	u, _ := newRenderUcm(t)
	factory, _ := sharedLockerFactory(t, "shared")

	hold := make(chan struct{})
	release := make(chan struct{})
	firstRunner := &fakeRunner{
		ImportHook: func(_ context.Context) {
			close(hold)
			<-release
		},
	}
	firstTf := newImportTerraform(t, u, firstRunner, factory, "alice")

	errCh := make(chan error, 1)
	go func() {
		errCh <- firstTf.Import(t.Context(), u, "databricks_catalog.sales", "sales_prod")
	}()

	<-hold

	secondRunner := &fakeRunner{}
	secondTf := newImportTerraform(t, u, secondRunner, factory, "bob")

	err := secondTf.Import(t.Context(), u, "databricks_catalog.sales", "sales_prod")
	require.Error(t, err)
	var held *lock.ErrLockHeld
	require.True(t, errors.As(err, &held), "expected ErrLockHeld, got %T: %v", err, err)
	assert.Equal(t, "alice", held.Holder)
	assert.Equal(t, 0, secondRunner.ImportCalls, "second Import must not invoke the runner")

	close(release)
	require.NoError(t, <-errCh)
}

func TestImportPropagatesRunnerError(t *testing.T) {
	u, _ := newRenderUcm(t)
	sentinel := errors.New("import boom")
	runner := &fakeRunner{ImportErr: sentinel}
	factory, _ := sharedLockerFactory(t, "alice")
	tf := newImportTerraform(t, u, runner, factory, "alice")

	err := tf.Import(t.Context(), u, "databricks_catalog.sales", "sales_prod")
	require.Error(t, err)
	require.ErrorIs(t, err, sentinel)
	assert.Equal(t, 1, runner.ImportCalls)
}
