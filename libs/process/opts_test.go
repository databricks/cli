package process

import (
	"os/exec"
	"runtime"
	"sort"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
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
	sort.Strings(vars)

	assert.GreaterOrEqual(t, len(vars), 2)
	assert.Equal(t, "CCC=DDD", vars[0])
	assert.Equal(t, "EEE=FFF", vars[1])
}
