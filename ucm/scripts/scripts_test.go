package scripts_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/scripts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUcm(t *testing.T) *ucm.Ucm {
	t.Helper()
	return &ucm.Ucm{
		RootPath: t.TempDir(),
		Config: config.Root{
			Scripts: map[string]config.Script{},
		},
	}
}

func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell execution test skipped on windows")
	}
}

func TestExecuteNoopWhenUnset(t *testing.T) {
	skipOnWindows(t)
	u := newUcm(t)
	ctx, _ := cmdio.NewTestContextWithStderr(logdiag.InitContext(t.Context()))

	diags := ucm.Apply(ctx, u, scripts.Execute(config.ScriptPreInit))
	assert.Empty(t, diags)
}

func TestExecuteRunsScriptWhenDefined(t *testing.T) {
	skipOnWindows(t)
	u := newUcm(t)
	u.Config.Scripts[config.ScriptPreDeploy] = config.Script{Content: "echo hello"}
	ctx, stderr := cmdio.NewTestContextWithStderr(logdiag.InitContext(t.Context()))

	diags := ucm.Apply(ctx, u, scripts.Execute(config.ScriptPreDeploy))
	require.Empty(t, diags)
	assert.Contains(t, stderr.String(), "hello")
}

func TestExecuteReturnsErrorOnNonZeroExit(t *testing.T) {
	skipOnWindows(t)
	u := newUcm(t)
	u.Config.Scripts[config.ScriptPostDestroy] = config.Script{Content: "exit 1"}
	ctx, _ := cmdio.NewTestContextWithStderr(logdiag.InitContext(t.Context()))

	diags := ucm.Apply(ctx, u, scripts.Execute(config.ScriptPostDestroy))
	require.NotEmpty(t, diags)
	assert.True(t, diags.HasError())
}

func TestExecuteUnknownHookIsNoop(t *testing.T) {
	skipOnWindows(t)
	u := newUcm(t)
	u.Config.Scripts[config.ScriptPreDeploy] = config.Script{Content: "echo hello"}
	ctx, stderr := cmdio.NewTestContextWithStderr(logdiag.InitContext(t.Context()))

	// Looking up a hook that isn't set must not fire the script bound to a
	// different hook — keys are literal strings, not overlapping ranges.
	diags := ucm.Apply(ctx, u, scripts.Execute(config.ScriptPostDeploy))
	assert.Empty(t, diags)
	assert.False(t, strings.Contains(stderr.String(), "hello"))
}

func TestExecutePropagatesEnvVars(t *testing.T) {
	skipOnWindows(t)
	u := newUcm(t)
	// Echoing the variable exercises env passthrough; we assert the env
	// value set on the parent process is visible to the child shell.
	u.Config.Scripts[config.ScriptPreInit] = config.Script{Content: `printf "HOME=%s\n" "$HOME"`}
	t.Setenv("HOME", "/tmp/ucm-scripts-test-home")
	ctx, stderr := cmdio.NewTestContextWithStderr(logdiag.InitContext(t.Context()))

	diags := ucm.Apply(ctx, u, scripts.Execute(config.ScriptPreInit))
	require.Empty(t, diags)
	assert.Contains(t, stderr.String(), "HOME=/tmp/ucm-scripts-test-home")
}
