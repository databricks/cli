package phases_test

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/phases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readRemoteSeq returns the Seq of the remote ucm-state.json, or -1 when the
// file has not been written yet. Used to assert Push advanced the remote.
func readRemoteSeq(t *testing.T, f *fixture) int {
	t.Helper()
	rc, err := f.remote.Read(t.Context(), deploy.UcmStateFileName)
	if err != nil {
		return -1
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	var s deploy.State
	require.NoError(t, json.Unmarshal(data, &s))
	return s.Seq
}

func TestDeployHappyPath(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Deploy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 1, f.tf.RenderCalls)
	assert.Equal(t, 1, f.tf.InitCalls)
	assert.Equal(t, 1, f.tf.ApplyCalls)
	// Post-apply Push must advance the remote Seq from 0 to 1.
	assert.Equal(t, 1, readRemoteSeq(t, f))
}

// TestDeployDirectEngineSkipsTerraform asserts that when the direct engine
// is selected Deploy routes through direct.Apply rather than the terraform
// wrapper: the terraform fake's counters stay at zero, no remote state is
// pushed, and no error is raised by the happy empty-config path.
func TestDeployDirectEngineSkipsTerraform(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Ucm.Engine = engine.EngineDirect
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Deploy(ctx, f.u, phases.Options{
		TerraformFactory:    fakeTfFactory(f.tf),
		DirectClientFactory: fakeDirectClientFactory(),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 0, f.tf.ApplyCalls)
	assert.Equal(t, -1, readRemoteSeq(t, f), "direct engine must never touch remote state")
}

func TestDeployBailsOnApplyError(t *testing.T) {
	f := newFixture(t)
	f.tf.ApplyErr = errSentinel
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Deploy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.True(t, logdiag.HasError(ctx))
	// Apply failed → Push must NOT run; remote ucm-state.json must stay
	// absent because Pull only writes locally on first-run.
	assert.Equal(t, -1, readRemoteSeq(t, f))
}

func TestDeployBailsOnInitializeError(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	// Zero Backend → Pull fails → the rest of Deploy short-circuits.
	phases.Deploy(ctx, f.u, phases.Options{TerraformFactory: fakeTfFactory(f.tf)})

	require.True(t, logdiag.HasError(ctx))
	assert.Equal(t, 0, f.tf.ApplyCalls)
}

func TestDeployBailsOnBuildError(t *testing.T) {
	f := newFixture(t)
	f.tf.RenderErr = errSentinel
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Deploy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.True(t, logdiag.HasError(ctx))
	assert.Equal(t, 0, f.tf.ApplyCalls)
}
