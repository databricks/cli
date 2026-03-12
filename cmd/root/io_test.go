package root

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestIsTruthy(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"yes", true},
		{"YES", true},
		{"0", false},
		{"false", false},
		{"no", false},
		{"", false},
		{"random", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, isTruthy(tt.input))
		})
	}
}

func TestInitInteractionFlagsFromEnv(t *testing.T) {
	ctx := t.Context()

	ctx = env.Set(ctx, "DATABRICKS_QUIET", "1")
	ctx = env.Set(ctx, "DATABRICKS_NO_INPUT", "true")
	ctx = env.Set(ctx, "DATABRICKS_YES", "YES")

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(ctx)

	f := initInteractionFlags(cmd)
	assert.True(t, f.quiet)
	assert.True(t, f.noInput)
	assert.True(t, f.yes)
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
	f.applyToContext(ctx)

	assert.True(t, cmdio.IsQuiet(ctx))
	assert.True(t, cmdio.IsNoInput(ctx))
	assert.True(t, cmdio.IsYes(ctx))
}
