package phases_test

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/phases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeHappyPath(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	setting := phases.Initialize(ctx, f.u, phases.Options{Backend: f.backend})

	require.False(t, logdiag.HasError(ctx), "unexpected error diagnostics: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, engine.EngineTerraform, setting.Type)
	assert.Equal(t, "default", setting.Source)
}

func TestInitializeDirectEngineIsStubbed(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Ucm.Engine = engine.EngineDirect

	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	setting := phases.Initialize(ctx, f.u, phases.Options{Backend: f.backend})

	assert.True(t, logdiag.HasError(ctx))
	diags := logdiag.FlushCollected(ctx)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "direct engine is not yet implemented")
	assert.Equal(t, engine.EngineDirect, setting.Type)
}

func TestInitializeDirectEngineViaEnv(t *testing.T) {
	f := newFixture(t)
	ctx := env.Set(t.Context(), engine.EnvVar, "direct")
	ctx = logdiag.InitContext(ctx)
	logdiag.SetCollect(ctx, true)

	setting := phases.Initialize(ctx, f.u, phases.Options{Backend: f.backend})

	assert.True(t, logdiag.HasError(ctx))
	assert.Equal(t, engine.EngineDirect, setting.Type)
	assert.Equal(t, "env", setting.Source)
}

func TestInitializeInvalidEngineEnv(t *testing.T) {
	f := newFixture(t)
	ctx := env.Set(t.Context(), engine.EnvVar, "bogus")
	ctx = logdiag.InitContext(ctx)
	logdiag.SetCollect(ctx, true)

	phases.Initialize(ctx, f.u, phases.Options{Backend: f.backend})

	require.True(t, logdiag.HasError(ctx))
	diags := logdiag.FlushCollected(ctx)
	require.NotEmpty(t, diags)
	assert.Contains(t, diags[0].Summary, engine.EnvVar)
}

func TestInitializeMissingBackendFails(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	// Zero-valued Backend — Pull refuses to run.
	phases.Initialize(ctx, f.u, phases.Options{})

	require.True(t, logdiag.HasError(ctx))
	diags := logdiag.FlushCollected(ctx)
	require.NotEmpty(t, diags)
	assert.Contains(t, diags[0].Summary, "pull remote state")
}
