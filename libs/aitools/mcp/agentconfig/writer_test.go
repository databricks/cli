package agentconfig

import (
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
	"github.com/databricks/cli/libs/testdiff"
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
// variables set to empty so the test does not depend on the developer's shell.
func testHome(t *testing.T) context.Context {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	for _, key := range []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "COPILOT_HOME", "GEMINI_CLI_HOME", "XDG_CONFIG_HOME", "APPDATA"} {
		ctx = env.Set(ctx, key, "") //nolint:fatcontext // each key is set on the same context
	}
	return ctx
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

// TestRoundTripPerAgent compares against the goldens with newlines normalized,
// so a checkout that turns the fixtures into CRLF still passes. Run with -update
// to regenerate the after files, then review them by hand.
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
			goldenAfter := filepath.Join("testdata", a.Name, "after."+ext)
			if testdiff.OverwriteMode {
				require.NoError(t, os.WriteFile(goldenAfter, after, 0o644))
			}
			assert.Equal(t, testdiff.NormalizeNewlines(string(readFile(t, goldenAfter))), testdiff.NormalizeNewlines(string(after)))

			// The same entry again must not touch the file.
			_, status, err = Upsert(ctx, a, testName, testCmd, testArgs(a.Name), nil, false)
			require.NoError(t, err)
			assert.Equal(t, StatusUnchanged, status)
			assert.Equal(t, after, readFile(t, path))

			entries, err := Read(ctx, a)
			require.NoError(t, err)
			require.Contains(t, entries, testName)
			assert.True(t, mcpconfig.IsDatabricksEntry(entries[testName]))
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
			assert.ErrorContains(t, err, filepath.ToSlash(path))
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
	assert.True(t, mcpconfig.IsDatabricksEntry(entries[testName]))
}

// An owned entry with other arguments (say, a different profile) is updated in
// place, and the file is written.
func TestUpsertUpdatesOwnedEntry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		format mcpconfig.Format
		root   string
		data   string
	}{
		{"json", mcpconfig.FormatJSON, "mcpServers", `{"mcpServers": {"` + testName + `": {"command": "` + testCmd + `", "args": ["aitools", "mcp", "serve", "--profile", "old"]}}}`},
		{"toml", mcpconfig.FormatTOML, "mcp_servers", "[mcp_servers." + testName + "]\ncommand = \"" + testCmd + "\"\nargs = [\"aitools\", \"mcp\", \"serve\", \"--profile\", \"old\"]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config")
			writeFile(t, path, []byte(tc.data), 0o600)
			a := testAgent(path, tc.format, tc.root)

			_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
			require.NoError(t, err)
			assert.Equal(t, StatusUpdated, status)
			entries, err := Read(t.Context(), a)
			require.NoError(t, err)
			assert.Equal(t, testArgs("test"), toStrings(t, entries[testName]["args"]))
		})
	}
}

func toStrings(t *testing.T, v any) []string {
	t.Helper()
	list, ok := v.([]any)
	require.True(t, ok, "args is a list")
	out := make([]string, 0, len(list))
	for _, item := range list {
		s, ok := item.(string)
		require.True(t, ok)
		out = append(out, s)
	}
	return out
}

// The entry we would write must be recognized as ours, or a later Upsert
// needs force and Remove can never delete it. Nothing is written.
func TestUpsertRefusesCommandThatIsNotDatabricks(t *testing.T) {
	for _, cmd := range []string{"/usr/local/bin/cli", "/x/mydatabricks"} {
		t.Run(cmd, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

			_, _, err := Upsert(t.Context(), a, testName, cmd, testArgs("test"), nil, true)
			assert.ErrorContains(t, err, "must be the databricks binary")
			_, err = os.Stat(path)
			assert.ErrorIs(t, err, os.ErrNotExist)
		})
	}

	// Windows writes the binary in any case.
	path := filepath.Join(t.TempDir(), "config.json")
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")
	_, status, err := Upsert(t.Context(), a, testName, `C:\Tools\Databricks.EXE`, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
}

func TestReadMissingFileHasNoEntries(t *testing.T) {
	a := testAgent(filepath.Join(t.TempDir(), "missing.toml"), mcpconfig.FormatTOML, "mcp_servers")
	entries, err := Read(t.Context(), a)
	require.NoError(t, err)
	assert.Empty(t, entries)
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

// A link to a missing file is refused rather than replaced by a regular file.
func TestUpsertRefusesDanglingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs privileges on Windows")
	}
	dir := t.TempDir()
	link := filepath.Join(dir, "config.json")
	require.NoError(t, os.Symlink(filepath.Join(dir, "missing", "config.json"), link))
	a := testAgent(link, mcpconfig.FormatJSON, "mcpServers")

	_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	assert.ErrorContains(t, err, "does not exist")
	info, err := os.Lstat(link)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink, "link is kept")
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

// A name that is not a valid server name is refused before anything is read or
// written.
func TestUpsertRefusesInvalidServerName(t *testing.T) {
	for _, name := range []string{"genie.one", "genie one", "genie/one", ""} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

			_, _, err := Upsert(t.Context(), a, name, testCmd, testArgs("test"), nil, false)
			assert.ErrorContains(t, err, "must contain only letters, digits, '-' and '_'")
			_, err = os.Stat(path)
			assert.ErrorIs(t, err, os.ErrNotExist)
		})
	}
}

// The error names the part that is wrong: the command or the args.
func TestUpsertNamesTheWrongPart(t *testing.T) {
	a := testAgent(filepath.Join(t.TempDir(), "config.json"), mcpconfig.FormatJSON, "mcpServers")

	_, _, err := Upsert(t.Context(), a, testName, "/usr/local/bin/cli", testArgs("test"), nil, false)
	assert.ErrorContains(t, err, "must be the databricks binary")

	_, _, err = Upsert(t.Context(), a, testName, testCmd, []string{"aitools", "serve"}, nil, false)
	assert.ErrorContains(t, err, "args must start with aitools mcp serve")
	assert.NotContains(t, err.Error(), "binary")
}

// Keys the user added to an entry we wrote are kept when it is updated. An entry
// that matches in the keys we write is not rewritten, and a changed arg is
// repaired.
func TestUpsertKeepsUserKeysJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp_config.json")
	entry := testEntry(testCmd, testArgs("test"), nil)
	entry["disabled"] = true
	data, err := json.Marshal(map[string]any{"mcpServers": map[string]any{testName: entry}})
	require.NoError(t, err)
	writeFile(t, path, data, 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUnchanged, status)
	assert.Equal(t, data, readFile(t, path))

	_, status, err = Upsert(t.Context(), a, testName, testCmd, testArgs("other"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	entries, err := Read(t.Context(), a)
	require.NoError(t, err)
	assert.Equal(t, testArgs("other"), toStrings(t, entries[testName]["args"]))
	assert.Equal(t, true, entries[testName]["disabled"])

	// --force replaces an entry we did not write, user keys included.
	foreign := filepath.Join(t.TempDir(), "foreign.json")
	writeFile(t, foreign, []byte(`{"mcpServers":{"`+testName+`":{"command":"/usr/bin/other","args":[],"disabled":true}}}`), 0o600)
	_, status, err = Upsert(t.Context(), testAgent(foreign, mcpconfig.FormatJSON, "mcpServers"), testName, testCmd, testArgs("test"), nil, true)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	entries, err = Read(t.Context(), testAgent(foreign, mcpconfig.FormatJSON, "mcpServers"))
	require.NoError(t, err)
	assert.NotContains(t, entries[testName], "disabled")
}

// A Codex block keeps the keys the user added, such as enabled = false, when
// it is updated in place.
func TestUpsertKeepsUserKeysTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	data := testBlock + "enabled = false\n"
	writeFile(t, path, []byte(data), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUnchanged, status)
	assert.Equal(t, data, string(readFile(t, path)))

	_, status, err = Upsert(t.Context(), a, testName, testCmd, testArgs("other"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	entries, err := Read(t.Context(), a)
	require.NoError(t, err)
	assert.Equal(t, testArgs("other"), toStrings(t, entries[testName]["args"]))
	assert.Equal(t, false, entries[testName]["enabled"])
}
