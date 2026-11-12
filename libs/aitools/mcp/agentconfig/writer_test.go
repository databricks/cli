package agentconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testName = "genie_one_mcp-dogfood"
	testCmd  = "/opt/databricks/bin/databricks"
)

func testArgs(agent string) []string {
	return []string{"aitools", "mcp", "serve", "--profile", "dogfood", "--service", "system.ai.genie_one_mcp", "--agent", agent}
}

// testHome returns a context whose home is a fresh temp dir, with the override
// variables cleared so the test does not depend on the developer's shell.
func testHome(t *testing.T) context.Context {
	clearAgentOverrides(t)
	return env.WithUserHomeDir(t.Context(), t.TempDir())
}

// clearAgentOverrides empties the variables the agent resolvers read. An empty
// value is treated as unset, and t.Setenv restores the originals afterwards.
func clearAgentOverrides(t *testing.T) {
	for _, key := range []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "COPILOT_HOME", "GEMINI_CLI_HOME", "XDG_CONFIG_HOME", "APPDATA"} {
		t.Setenv(key, "")
	}
}

func testEntry(cmd string, args []string, vars map[string]string) map[string]any {
	entry := map[string]any{"command": cmd, "args": args}
	if len(vars) > 0 {
		entry["env"] = vars
	}
	return entry
}

// testAgent is an agent whose config lives at path, for tests that need a
// format or a command that no real agent uses.
func testAgent(path string, format mcpconfig.Format, root string) *agents.Agent {
	return &agents.Agent{
		Name:        "test",
		DisplayName: "Test",
		MCP: &mcpconfig.Spec{
			Path:   func(context.Context) (string, error) { return path, nil },
			Format: format,
			Root:   root,
			Entry:  testEntry,
			IsOurs: mcpconfig.IsDatabricksEntry,
		},
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func writeFile(t *testing.T, path string, data []byte, perm os.FileMode) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, data, perm))
}

// semantic decodes data into plain Go values, so two files compare equal when
// they hold the same data regardless of formatting, key order or comments.
func semantic(t *testing.T, format mcpconfig.Format, data []byte) map[string]any {
	t.Helper()
	out := map[string]any{}
	if format == mcpconfig.FormatTOML {
		_, err := toml.Decode(string(data), &out)
		require.NoError(t, err)
		return out
	}
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

func TestRoundTripPerAgent(t *testing.T) {
	for _, a := range agents.MCPAgents() {
		t.Run(a.Name, func(t *testing.T) {
			ctx := testHome(t)
			path, err := a.MCP.Path(ctx)
			require.NoError(t, err)
			ext := "json"
			if a.MCP.Format == mcpconfig.FormatTOML {
				ext = "toml"
			}
			before := readFile(t, filepath.Join("testdata", a.Name, "before."+ext))
			writeFile(t, path, before, 0o600)

			gotPath, status, err := Upsert(ctx, a, testName, testCmd, testArgs(a.Name), nil, false)
			require.NoError(t, err)
			assert.Equal(t, path, gotPath)
			assert.Equal(t, StatusAdded, status)
			after := readFile(t, path)
			assert.Equal(t, string(readFile(t, filepath.Join("testdata", a.Name, "after."+ext))), string(after))

			// The same entry again must not touch the file.
			_, status, err = Upsert(ctx, a, testName, testCmd, testArgs(a.Name), nil, false)
			require.NoError(t, err)
			assert.Equal(t, StatusUnchanged, status)
			assert.Equal(t, after, readFile(t, path))

			entries, err := Read(ctx, a)
			require.NoError(t, err)
			require.Contains(t, entries, testName)
			assert.True(t, a.MCP.IsOurs(entries[testName]))
			assert.Contains(t, entries, "other")

			removed, err := Remove(ctx, a, testName)
			require.NoError(t, err)
			assert.True(t, removed)
			assert.Equal(t, semantic(t, a.MCP.Format, before), semantic(t, a.MCP.Format, readFile(t, path)))
		})
	}
}

func TestUpsertRefusesUnparsableFile(t *testing.T) {
	for _, tc := range []struct {
		name   string
		format mcpconfig.Format
		data   string
	}{
		{"json syntax", mcpconfig.FormatJSON, "{not json"},
		{"json top level array", mcpconfig.FormatJSON, `[]`},
		{"json root not object", mcpconfig.FormatJSON, `{"mcpServers": 5}`},
		{"json entry not object", mcpconfig.FormatJSON, `{"mcpServers": {"` + testName + `": 5}}`},
		{"json trailing data", mcpconfig.FormatJSON, `{} {}`},
		{"toml syntax", mcpconfig.FormatTOML, "a = ["},
		{"toml root not table", mcpconfig.FormatTOML, "mcp_servers = 5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config")
			writeFile(t, path, []byte(tc.data), 0o600)
			a := testAgent(path, tc.format, "mcpServers")
			if tc.format == mcpconfig.FormatTOML {
				a = testAgent(path, tc.format, "mcp_servers")
			}

			_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, true)
			require.ErrorIs(t, err, ErrConfigInvalid)
			assert.ErrorContains(t, err, path)
			_, err = Remove(t.Context(), a, testName)
			require.ErrorIs(t, err, ErrConfigInvalid)

			assert.Equal(t, tc.data, string(readFile(t, path)))
		})
	}
}

func TestUpsertRefusesEntryWeDidNotWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	other := `{"mcpServers":{"` + testName + `":{"command":"/usr/bin/other","args":[]}}}`
	writeFile(t, path, []byte(other), 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.ErrorIs(t, err, ErrNotOwned)
	assert.Equal(t, other, string(readFile(t, path)))

	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, true)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	entries, err := Read(t.Context(), a)
	require.NoError(t, err)
	assert.True(t, a.MCP.IsOurs(entries[testName]))
}

func TestRemoveRefusesEntryWeDidNotWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	other := `{"mcpServers":{"` + testName + `":{"command":"/usr/bin/other","args":[]}}}`
	writeFile(t, path, []byte(other), 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	_, err := Remove(t.Context(), a, testName)
	require.ErrorIs(t, err, ErrNotOwned)
	assert.Equal(t, other, string(readFile(t, path)))
}

func TestRemoveAbsentEntryDoesNotCreateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	removed, err := Remove(t.Context(), a, testName)
	require.NoError(t, err)
	assert.False(t, removed)
	_, err = os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestUpsertUnchangedDoesNotRewriteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	// Compact formatting that a rewrite would change, holding the exact entry.
	entry, err := json.Marshal(testEntry(testCmd, testArgs("test"), nil))
	require.NoError(t, err)
	data := []byte(`{"mcpServers":{"` + testName + `":` + string(entry) + `}}`)
	writeFile(t, path, data, 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUnchanged, status)
	assert.Equal(t, data, readFile(t, path))
}

func TestUpsertKeepsFileModeAndCreatesPrivateFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not honor unix file modes")
	}
	dir := t.TempDir()

	existing := filepath.Join(dir, "existing.json")
	writeFile(t, existing, []byte(`{}`), 0o640)
	a := testAgent(existing, mcpconfig.FormatJSON, "mcpServers")
	_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	info, err := os.Stat(existing)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o640), info.Mode().Perm())

	// A missing file and its missing parent directory are created; the file gets 0600.
	created := filepath.Join(dir, "new", "config.json")
	a = testAgent(created, mcpconfig.FormatJSON, "mcpServers")
	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	info, err = os.Stat(created)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestUpsertWritesThroughSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs privileges on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles", "config.json")
	writeFile(t, target, []byte(`{}`), 0o600)
	link := filepath.Join(dir, "config.json")
	require.NoError(t, os.Symlink(target, link))
	a := testAgent(link, mcpconfig.FormatJSON, "mcpServers")

	_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	info, err := os.Lstat(link)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink, "link is replaced by a regular file")
	entries, err := Read(t.Context(), a)
	require.NoError(t, err)
	assert.Contains(t, entries, testName)
}

func TestWindowsCommandPathRoundTrip(t *testing.T) {
	winCmd := `C:\Program Files\databricks\databricks.exe`
	for _, tc := range []struct {
		name   string
		format mcpconfig.Format
		root   string
		want   string
	}{
		{"json", mcpconfig.FormatJSON, "mcpServers", `"command": "C:\\Program Files\\databricks\\databricks.exe"`},
		{"toml", mcpconfig.FormatTOML, "mcp_servers", `command = "C:\\Program Files\\databricks\\databricks.exe"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config")
			a := testAgent(path, tc.format, tc.root)
			_, _, err := Upsert(t.Context(), a, testName, winCmd, testArgs("test"), nil, false)
			require.NoError(t, err)
			assert.Contains(t, string(readFile(t, path)), tc.want)

			entries, err := Read(t.Context(), a)
			require.NoError(t, err)
			require.Contains(t, entries, testName)
			assert.Equal(t, winCmd, entries[testName]["command"])
			assert.True(t, mcpconfig.IsDatabricksEntry(entries[testName]))
		})
	}
}

func TestJSONKeepsOtherValuesByteForByte(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, path, []byte(`{"big": 12345678901234567890, "ratio": 1.50, "text": "caf\u00e9", "mcpServers": {}}`), 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	got := readFile(t, path)
	assert.Contains(t, string(got), `"big": 12345678901234567890`)
	assert.Contains(t, string(got), `"ratio": 1.50`)
	assert.Contains(t, string(got), `"text": "caf\u00e9"`)
	assert.True(t, bytes.HasSuffix(got, []byte("}\n")), "file ends with a newline")
}

// TOML comments are lost on rewrite. This test pins that behavior so a change to
// it is deliberate.
func TestTOMLRewriteDropsComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, path, []byte("# keep me?\nmodel = \"gpt-5\"\n"), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	got := string(readFile(t, path))
	assert.NotContains(t, got, "# keep me?")
	assert.Contains(t, got, `model = "gpt-5"`)
}
