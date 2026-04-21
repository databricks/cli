package phases_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/phases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeTfFactory(f *fakeTf) phases.TerraformFactory {
	return func(_ context.Context, _ *ucm.Ucm) (phases.TerraformWrapper, error) {
		return f, nil
	}
}

func TestBuildHappyPath(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	tf := phases.Build(ctx, f.u, phases.Options{TerraformFactory: fakeTfFactory(f.tf)})

	require.False(t, logdiag.HasError(ctx))
	require.NotNil(t, tf)
	assert.Equal(t, 1, f.tf.RenderCalls)
	assert.Equal(t, 0, f.tf.InitCalls, "Build must not call terraform init itself")
}

func TestBuildFactoryError(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	factory := func(_ context.Context, _ *ucm.Ucm) (phases.TerraformWrapper, error) {
		return nil, errSentinel
	}

	tf := phases.Build(ctx, f.u, phases.Options{TerraformFactory: factory})

	assert.Nil(t, tf)
	require.True(t, logdiag.HasError(ctx))
	diags := logdiag.FlushCollected(ctx)
	require.NotEmpty(t, diags)
	assert.Contains(t, diags[0].Summary, errSentinel.Error())
}

func TestBuildRenderError(t *testing.T) {
	f := newFixture(t)
	f.tf.RenderErr = errSentinel

	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	tf := phases.Build(ctx, f.u, phases.Options{TerraformFactory: fakeTfFactory(f.tf)})

	assert.Nil(t, tf)
	require.True(t, logdiag.HasError(ctx))
	diags := logdiag.FlushCollected(ctx)
	require.NotEmpty(t, diags)
	assert.Contains(t, diags[0].Summary, "render terraform config")
}
