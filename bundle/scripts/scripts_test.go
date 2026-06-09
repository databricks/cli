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
					// Write well over the ~64KiB pipe buffer to stderr before
					// touching stdout. With sequential draining (stdout to EOF
					// first) the script blocks on the stderr write and never
					// closes stdout, deadlocking the mutator.
					config.ScriptPreInit: "seq 100000 | tr -d '\\n' >&2; echo stdout-after-stderr",
				},
			},
		},
	}

	// Bound the test so a reintroduced deadlock fails fast (the context kills
	// the script) instead of hanging until the go test timeout.
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	ctx, stderr := cmdio.NewTestContextWithStderr(ctx)
	diags := bundle.Apply(ctx, b, scripts.Execute(config.ScriptPreInit))
	require.NoError(t, diags.Error())

	output := stderr.String()
	assert.Contains(t, output, "99999100000")
	assert.Contains(t, output, "stdout-after-stderr")
	// stderr is spooled and logged only after stdout reaches EOF, so the
	// historical stdout-then-stderr output order is preserved.
	assert.Less(t, strings.Index(output, "stdout-after-stderr"), strings.Index(output, "99999100000"))
}
