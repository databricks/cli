package phases_test

import (
	"testing"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/phases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDestroyHappyPath(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Destroy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	// Destroy does NOT call Render via Build — Init re-renders.
	assert.Equal(t, 0, f.tf.RenderCalls)
	assert.Equal(t, 1, f.tf.InitCalls)
	assert.Equal(t, 1, f.tf.DestroyCalls)
	assert.Equal(t, 0, f.tf.ApplyCalls)
	// Post-destroy Push must advance the remote Seq.
	assert.Equal(t, 1, readRemoteSeq(t, f))
}

func TestDestroyShortCircuitsOnDirectEngine(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Ucm.Engine = engine.EngineDirect
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Destroy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.True(t, logdiag.HasError(ctx))
	assert.Equal(t, 0, f.tf.DestroyCalls)
}

func TestDestroyBailsOnDestroyError(t *testing.T) {
	f := newFixture(t)
	f.tf.DestroyErr = errSentinel
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Destroy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.True(t, logdiag.HasError(ctx))
	// Destroy failed before Push; remote state file must not have been
	// written (Pull only writes locally on first-run).
	assert.Equal(t, -1, readRemoteSeq(t, f))
}

func TestDestroyBailsOnInitializeError(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Destroy(ctx, f.u, phases.Options{TerraformFactory: fakeTfFactory(f.tf)})

	require.True(t, logdiag.HasError(ctx))
	assert.Equal(t, 0, f.tf.DestroyCalls)
}

func TestDestroyBailsOnInitError(t *testing.T) {
	f := newFixture(t)
	f.tf.InitErr = errSentinel
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Destroy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.True(t, logdiag.HasError(ctx))
	assert.Equal(t, 0, f.tf.DestroyCalls)
}
