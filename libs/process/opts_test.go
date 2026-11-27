package process

import (
	"os/exec"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithEnvs(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Skipping test on windows for now because of the following error:
		// /bin/sh -c echo $FOO $BAR:  exec: "/bin/sh": file does not exist
		t.SkipNow()
	}
	ctx := t.Context()
	ctx2 := env.Set(ctx, "FOO", "foo")
	res, err := Background(ctx2, []string{"/bin/sh", "-c", "echo $FOO $BAR"}, WithEnvs(map[string]string{
		"BAR": "delirium",
	}))
	assert.NoError(t, err)
	assert.Equal(t, "foo delirium\n", res)
}

func TestWithoutEnvRemovesVariable(t *testing.T) {
	cmd := &exec.Cmd{Env: []string{"FOO=", "FOOBAR=bar", "BAZ=baz"}}
	require.NoError(t, WithoutEnv("FOO")(t.Context(), cmd))
	assert.Equal(t, []string{"FOOBAR=bar", "BAZ=baz"}, cmd.Env)
}

func TestEnvEntryHasKey(t *testing.T) {
	assert.True(t, envEntryHasKey("linux", "CODEX_HOME=/a", "CODEX_HOME"))
	assert.False(t, envEntryHasKey("linux", "Codex_Home=/a", "CODEX_HOME"), "variable names are case-sensitive off Windows")
	assert.False(t, envEntryHasKey("linux", "CODEX_HOMEX=/a", "CODEX_HOME"))
	assert.True(t, envEntryHasKey("windows", "Codex_Home=/a", "CODEX_HOME"))
	assert.False(t, envEntryHasKey("windows", "CODEX_HOMEX=/a", "CODEX_HOME"))
}

func TestWithWaitDelayReturnsWhileChildHoldsOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Skipping test on windows for now, as TestWithEnvs does: /bin/sh is not there.
		t.Skip()
	}
	start := time.Now()
	_, err := Background(t.Context(), []string{"/bin/sh", "-c", "sleep 3 & echo started"}, WithWaitDelay(100*time.Millisecond))
	assert.ErrorIs(t, err, exec.ErrWaitDelay, "the child keeps the output open, so the wait ends at the delay")
	assert.Less(t, time.Since(start), 2*time.Second)
}

func TestWorksWithLibsEnv(t *testing.T) {
	testutil.CleanupEnvironment(t)
	ctx := t.Context()

	cmd := &exec.Cmd{}
	err := WithEnvs(map[string]string{
		"CCC": "DDD",
		"EEE": "FFF",
	})(ctx, cmd)
	assert.NoError(t, err)

	vars := cmd.Environ()
	slices.Sort(vars)

	assert.GreaterOrEqual(t, len(vars), 2)
	assert.Equal(t, "CCC=DDD", vars[0])
	assert.Equal(t, "EEE=FFF", vars[1])
}
