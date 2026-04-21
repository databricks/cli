package phases_test

import (
	"testing"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/databricks/cli/ucm/phases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanHappyPath(t *testing.T) {
	f := newFixture(t)
	f.tf.PlanResult = &terraform.PlanResult{HasChanges: true, Summary: "plan has changes"}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	result := phases.Plan(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.False(t, logdiag.HasError(ctx))
	require.NotNil(t, result)
	assert.True(t, result.HasChanges)
	assert.Equal(t, 1, f.tf.RenderCalls)
	assert.Equal(t, 1, f.tf.InitCalls)
	assert.Equal(t, 1, f.tf.PlanCalls)
}

func TestPlanBailsOnInitializeError(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	// Missing Backend — Initialize.Pull fails, Plan should short-circuit.
	result := phases.Plan(ctx, f.u, phases.Options{TerraformFactory: fakeTfFactory(f.tf)})

	assert.Nil(t, result)
	assert.True(t, logdiag.HasError(ctx))
	assert.Equal(t, 0, f.tf.RenderCalls, "Build must not run when Initialize failed")
}

func TestPlanShortCircuitsOnDirectEngine(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Ucm.Engine = engine.EngineDirect
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	result := phases.Plan(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	assert.Nil(t, result)
	assert.True(t, logdiag.HasError(ctx))
	assert.Equal(t, 0, f.tf.RenderCalls)
	assert.Equal(t, 0, f.tf.InitCalls)
	assert.Equal(t, 0, f.tf.PlanCalls)
}

func TestPlanBailsOnInitError(t *testing.T) {
	f := newFixture(t)
	f.tf.InitErr = errSentinel
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	result := phases.Plan(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	assert.Nil(t, result)
	assert.True(t, logdiag.HasError(ctx))
	assert.Equal(t, 0, f.tf.PlanCalls, "Plan should not run when Init fails")
}

func TestPlanPropagatesPlanError(t *testing.T) {
	f := newFixture(t)
	f.tf.PlanErr = errSentinel
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	result := phases.Plan(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	assert.Nil(t, result)
	require.True(t, logdiag.HasError(ctx))
	diags := logdiag.FlushCollected(ctx)
	require.NotEmpty(t, diags)
	assert.Contains(t, diags[0].Summary, "terraform plan")
}
