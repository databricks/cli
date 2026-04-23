package phases_test

import (
	"io"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/config/resources"
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

// TestDestroyDirectEngineSkipsTerraform asserts Destroy with engine=direct
// uses direct.Destroy instead of the terraform wrapper. Empty state means
// zero SDK calls fire on the fake client.
func TestDestroyDirectEngineSkipsTerraform(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Ucm.Engine = engine.EngineDirect
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Destroy(ctx, f.u, phases.Options{
		TerraformFactory:    fakeTfFactory(f.tf),
		DirectClientFactory: fakeDirectClientFactory(),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
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

// TestDestroySkipsPromptWhenNoResources asserts that an empty configuration
// produces an empty destroy plan, which approvalForDestroy must treat as
// "nothing to confirm" — tf.Destroy still runs (terraform own-state may
// carry orphaned resources) and remote state advances.
func TestDestroySkipsPromptWhenNoResources(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Destroy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 1, f.tf.DestroyCalls)
}

// TestDestroySkipsPromptWhenAutoApprove asserts --auto-approve bypasses the
// prompt even when u.Config.Resources carries catalogs slated for deletion.
func TestDestroySkipsPromptWhenAutoApprove(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Resources.Catalogs = map[string]*resources.Catalog{
		"main": {Name: "main"},
	}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	ctx, _ = cmdio.NewTestContextWithStderr(ctx)

	phases.Destroy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
		AutoApprove:      true,
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 1, f.tf.DestroyCalls)
}

// TestDestroyErrorsWhenPromptingNotSupported asserts Destroy surfaces the
// --auto-approve guidance when stdin is not a TTY and the config contains
// resources that would be destroyed. Destroy must not fire.
func TestDestroyErrorsWhenPromptingNotSupported(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Resources.Catalogs = map[string]*resources.Catalog{
		"main": {Name: "main"},
	}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	ctx, _ = cmdio.NewTestContextWithStderr(ctx)

	phases.Destroy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.True(t, logdiag.HasError(ctx))
	diags := logdiag.FlushCollected(ctx)
	require.NotEmpty(t, diags)
	assert.Contains(t, diags[0].Summary, "--auto-approve")
	assert.Equal(t, 0, f.tf.DestroyCalls)
}

// TestDestroyPromptsAndAccepts drives the interactive prompt path: with a
// catalog in the config and "y\n" on stdin, Destroy must call tf.Destroy and
// push state.
func TestDestroyPromptsAndAccepts(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Resources.Catalogs = map[string]*resources.Catalog{
		"main": {Name: "main"},
	}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	ctx, tio := cmdio.SetupTest(ctx, cmdio.TestOptions{PromptSupported: true})
	defer tio.Done()
	go func() {
		_, _ = tio.Stdin.WriteString("y\n")
		_ = tio.Stdin.Flush()
	}()
	go func() { _, _ = io.Copy(io.Discard, tio.Stderr) }()

	phases.Destroy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 1, f.tf.DestroyCalls)
}

// TestDestroyAbortsWhenPromptDeclined drives the same interactive path with
// "n\n" on stdin and asserts Destroy is never called.
func TestDestroyAbortsWhenPromptDeclined(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Resources.Catalogs = map[string]*resources.Catalog{
		"main": {Name: "main"},
	}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	ctx, tio := cmdio.SetupTest(ctx, cmdio.TestOptions{PromptSupported: true})
	defer tio.Done()
	go func() {
		_, _ = tio.Stdin.WriteString("n\n")
		_ = tio.Stdin.Flush()
	}()
	go func() { _, _ = io.Copy(io.Discard, tio.Stderr) }()

	phases.Destroy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 0, f.tf.DestroyCalls)
}
