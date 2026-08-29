package scripts_test

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/scripts"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteOutputWithoutTrailingNewline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	dir := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: dir,
		Config: config.Root{
			Experimental: &config.Experimental{
				Scripts: map[config.ScriptHook]config.Command{
					config.ScriptPreInit: "printf 'line1\nline2\nlast line without newline'",
				},
			},
		},
	}

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	diags := bundle.Apply(ctx, b, scripts.Execute(config.ScriptPreInit))
	require.NoError(t, diags.Error())

	output := stderr.String()
	assert.Contains(t, output, "line1")
	assert.Contains(t, output, "line2")
	assert.Contains(t, output, "last line without newline")
}

func TestExecuteLargeStderrOutputDoesNotDeadlock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	dir := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: dir,
		Config: config.Root{
			Experimental: &config.Experimental{
				Scripts: map[config.ScriptHook]config.Command{
					// Overflows the ~64KiB stderr pipe buffer before stdout
					// closes, which deadlocked the old sequential draining.
					config.ScriptPreInit: "seq 100000 | tr -d '\\n' >&2; echo stdout-after-stderr",
				},
			},
		},
	}

	// A reintroduced deadlock fails fast: the context timeout kills the script.
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	ctx, stderr := cmdio.NewTestContextWithStderr(ctx)
	diags := bundle.Apply(ctx, b, scripts.Execute(config.ScriptPreInit))
	require.NoError(t, diags.Error())

	output := stderr.String()
	assert.Contains(t, output, "99999100000")
	assert.Contains(t, output, "stdout-after-stderr")
	// The script writes stderr first, but spooling preserves the stdout-then-stderr order.
	assert.Less(t, strings.Index(output, "stdout-after-stderr"), strings.Index(output, "99999100000"))
}
