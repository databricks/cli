//go:build unix

package agentconfig

import (
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// groupEnv names the file that TestWriteProcessGroup writes its process group to.
const groupEnv = "AGENTCONFIG_TEST_GROUP"

// TestWriteProcessGroup is the child process of TestRunWithTimeoutKeepsCLIInForegroundGroup. It writes its process group.
func TestWriteProcessGroup(t *testing.T) {
	out := os.Getenv(groupEnv)
	if out == "" {
		t.Skip("runs only as the child process of TestRunWithTimeoutKeepsCLIInForegroundGroup")
	}
	require.NoError(t, os.WriteFile(out, []byte(strconv.Itoa(syscall.Getpgrp())), 0o600))
}

// TestRunWithTimeoutKeepsCLIInForegroundGroup checks that the CLI stays in the process group of databricks, so that a
// Ctrl-C at the terminal reaches it too.
func TestRunWithTimeoutKeepsCLIInForegroundGroup(t *testing.T) {
	out := filepath.Join(t.TempDir(), "group")
	ctx := env.Set(t.Context(), groupEnv, out)
	argv := []string{os.Args[0], "-test.run=^TestWriteProcessGroup$"}

	require.NoError(t, runWithTimeout(ctx, 30*time.Second, cliCommand{argv: argv}))
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, strconv.Itoa(syscall.Getpgrp()), string(data))
}
