package scripts_test

import (
	"runtime"
	"testing"

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

func TestExecuteNoScript(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{},
	}

	ctx, _ := cmdio.NewTestContextWithStderr(t.Context())
	diags := bundle.Apply(ctx, b, scripts.Execute(config.ScriptPreInit))
	require.NoError(t, diags.Error())
}
