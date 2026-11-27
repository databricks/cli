//go:build windows

package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProbeRealCmdShimPrintedHelp runs a real .cmd shim that prints the plugin
// help and exits 1, so the probe sees the exit code and stdout that cmd.exe
// actually reports rather than a stub.
func TestProbeRealCmdShimPrintedHelp(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "help.txt"), []byte(copilotPluginHelp), 0o644))
	shim := filepath.Join(dir, "copilot.cmd")
	require.NoError(t, os.WriteFile(shim, []byte("@type \"%~dp0help.txt\"\r\n@exit /b 1\r\n"), 0o755))

	origLook := agents.LookPath
	t.Cleanup(func() { agents.LookPath = origLook })
	agents.LookPath = func(string) (string, error) { return shim, nil }

	_, err := InstallPluginForAgent(t.Context(), pluginAgent(agents.NameCopilot, "GitHub Copilot", "copilot"), "", "v0.2.6")
	var be *BlockedError
	require.ErrorAs(t, err, &be)
	assert.Equal(t, ReasonShimExitedWithError, be.Reason)
	assert.True(t, strings.HasPrefix(be.Detail, "printed its plugin help but exited with an error (exit status 1)"), be.Detail)
}
