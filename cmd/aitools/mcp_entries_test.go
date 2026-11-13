package aitools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentFilesReadsEachFileOnce checks that a file is read on first use and then served from the cache: a second read
// in the same command returns what the first read saw, even when the file has changed since.
func TestAgentFilesReadsEachFileOnce(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName("claude-code")
	path := filepath.Join(home, ".claude.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"mcpServers":{"first":{"command":"ug","args":["mcp-proxy"]}}}`), 0o600))

	files := newAgentFiles()
	before := files.file(claude, path)
	require.NoError(t, before.err)
	require.Contains(t, before.entries, "first")

	require.NoError(t, os.WriteFile(path, []byte(`{"mcpServers":{"second":{"command":"ug","args":["mcp-proxy"]}}}`), 0o600))
	assert.Equal(t, before, files.file(claude, path), "the same command reads the file once")

	fresh := newAgentFiles().file(claude, path)
	require.NoError(t, fresh.err)
	assert.Contains(t, fresh.entries, "second", "a new command reads the file again")
}

// TestAgentFilesFindsCurrentPathOnce checks that an agent's current config path is found once: the agent's
// directory, changed after the first lookup, does not move the path for the rest of the command.
func TestAgentFilesFindsCurrentPathOnce(t *testing.T) {
	withHome(t)
	claude := agents.ByName("claude-code")
	first := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", first)

	files := newAgentFiles()
	path, err := files.currentPath(t.Context(), claude)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(first, ".claude.json"), path)

	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	again, err := files.currentPath(t.Context(), claude)
	require.NoError(t, err)
	assert.Equal(t, path, again)
}

// TestCheckedBindingKeepsTheCheckError checks that a server whose config file could not be resolved has no binding
// file, and that its ownership check reports the check's error.
func TestCheckedBindingKeepsTheCheckError(t *testing.T) {
	checkErr := assert.AnError
	binding := checkedBinding(mcp.EntryCheck{Err: checkErr})
	_, err := binding.owns(t.Context(), mcp.Entry{}, nil)
	assert.ErrorIs(t, err, checkErr)
}
