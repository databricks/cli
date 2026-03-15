package root

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestInitInteractionFlagsFromEnv(t *testing.T) {
	ctx := t.Context()

	ctx = env.Set(ctx, envQuiet, "on")
	ctx = env.Set(ctx, envNoInput, "T")
	ctx = env.Set(ctx, envYes, "YES")

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(ctx)

	f := initInteractionFlags(cmd)
	assert.True(t, f.quiet)
	assert.True(t, f.noInput)
	assert.True(t, f.yes)
}

func TestInitInteractionFlagsFromFalseOrInvalidEnv(t *testing.T) {
	ctx := t.Context()

	ctx = env.Set(ctx, envQuiet, "off")
	ctx = env.Set(ctx, envNoInput, "")
	ctx = env.Set(ctx, envYes, "maybe")

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(ctx)

	f := initInteractionFlags(cmd)
	assert.False(t, f.quiet)
	assert.False(t, f.noInput)
	assert.False(t, f.yes)
}

func TestInitInteractionFlagsDefaultFalse(t *testing.T) {
	ctx := t.Context()

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(ctx)

	f := initInteractionFlags(cmd)
	assert.False(t, f.quiet)
	assert.False(t, f.noInput)
	assert.False(t, f.yes)
}

func TestApplyToContextSetsFlags(t *testing.T) {
	ctx := t.Context()

	// Create a minimal cmdio context
	cmdIO := cmdio.NewIO(ctx, flags.OutputText, nil, nil, nil, "", "")
	ctx = cmdio.InContext(ctx, cmdIO)

	f := &interactionFlags{quiet: true, noInput: true, yes: true}
	flagCtx := f.applyToContext(ctx)

	assert.False(t, cmdio.IsQuiet(ctx))
	assert.False(t, cmdio.IsNoInput(ctx))
	assert.False(t, cmdio.IsYes(ctx))
	assert.True(t, cmdio.IsQuiet(flagCtx))
	assert.True(t, cmdio.IsNoInput(flagCtx))
	assert.True(t, cmdio.IsYes(flagCtx))
}
