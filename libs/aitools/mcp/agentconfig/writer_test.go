package agentconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

			entries, err := readAgent(ctx, a)
			require.NoError(t, err)
			require.Contains(t, entries, testName)
			assert.True(t, mcpconfig.IsDatabricksEntry(entries[testName]))
			assert.Contains(t, entries, "other")

			removed, err := removeAgent(ctx, a, testName)
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
			_, err = removeAgent(t.Context(), a, testName)
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
	entries, err := readAgent(t.Context(), a)
	require.NoError(t, err)
	assert.True(t, mcpconfig.IsDatabricksEntry(entries[testName]))
}

// TestUpsertUpdatesOwnedEntry checks that an owned entry with other arguments
// (say, a different profile) is updated in place, and the file is written.
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
			entries, err := readAgent(t.Context(), a)
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

// TestUpsertRefusesCommandThatIsNotDatabricks checks that the entry we would write
// must be recognized as ours, or a later Upsert needs force and Remove can never
// delete it. Nothing is written.
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
	entries, err := readAgent(t.Context(), a)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRemoveRefusesEntryWeDidNotWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	other := `{"mcpServers":{"` + testName + `":{"command":"/usr/bin/other","args":[]}}}`
	writeFile(t, path, []byte(other), 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	_, err := removeAgent(t.Context(), a, testName)
	require.ErrorIs(t, err, ErrNotOwned)
	assert.Equal(t, other, string(readFile(t, path)))
}

func TestRemoveAbsentEntryDoesNotCreateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	removed, err := removeAgent(t.Context(), a, testName)
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
	dirInfo, err := os.Stat(filepath.Dir(created))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())
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
	entries, err := readAgent(t.Context(), a)
	require.NoError(t, err)
	assert.Contains(t, entries, testName)
}

// TestUpsertRefusesDanglingSymlink checks that a link to a missing file is refused
// rather than replaced by a regular file.
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

			entries, err := readAgent(t.Context(), a)
			require.NoError(t, err)
			require.Contains(t, entries, testName)
			assert.Equal(t, winCmd, entries[testName]["command"])
			assert.True(t, mcpconfig.IsDatabricksEntry(entries[testName]))
		})
	}
}

// TestUpsertRefusesInvalidServerName checks that a name that is not a valid server
// name is refused before anything is read or written.
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

// TestUpsertNamesTheWrongPart checks that the error names the part that is
// wrong: the command or the args.
func TestUpsertNamesTheWrongPart(t *testing.T) {
	a := testAgent(filepath.Join(t.TempDir(), "config.json"), mcpconfig.FormatJSON, "mcpServers")

	_, _, err := Upsert(t.Context(), a, testName, "/usr/local/bin/cli", testArgs("test"), nil, false)
	require.ErrorContains(t, err, "must be the databricks binary")

	_, _, err = Upsert(t.Context(), a, testName, testCmd, []string{"aitools", "serve"}, nil, false)
	require.ErrorContains(t, err, "args must start with aitools mcp serve")
	assert.NotContains(t, err.Error(), "binary")
}

// TestUpsertKeepsUserKeysJSON checks that keys the user added to an entry we wrote
// are kept when it is updated. An entry that matches in the keys we write is not
// rewritten, and a changed arg is repaired.
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
	entries, err := readAgent(t.Context(), a)
	require.NoError(t, err)
	assert.Equal(t, testArgs("other"), toStrings(t, entries[testName]["args"]))
	assert.Equal(t, true, entries[testName]["disabled"])

	// --force replaces an entry we did not write, user keys included.
	foreign := filepath.Join(t.TempDir(), "foreign.json")
	writeFile(t, foreign, []byte(`{"mcpServers":{"`+testName+`":{"command":"/usr/bin/other","args":[],"disabled":true}}}`), 0o600)
	_, status, err = Upsert(t.Context(), testAgent(foreign, mcpconfig.FormatJSON, "mcpServers"), testName, testCmd, testArgs("test"), nil, true)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	entries, err = readAgent(t.Context(), testAgent(foreign, mcpconfig.FormatJSON, "mcpServers"))
	require.NoError(t, err)
	assert.NotContains(t, entries[testName], "disabled")
}

// TestUpsertKeepsUserKeysTOML checks that a Codex block keeps the keys the user
// added, such as enabled = false, when it is updated in place.
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
	entries, err := readAgent(t.Context(), a)
	require.NoError(t, err)
	assert.Equal(t, testArgs("other"), toStrings(t, entries[testName]["args"]))
	assert.Equal(t, false, entries[testName]["enabled"])
}

// TestUpsertKeepsToolsNarrowedByUser checks that Copilot narrows the tools of an entry we
// wrote after the first add. The list is the user's from then on: a re-add with
// the same launch keys leaves the file alone, and a changed argument keeps the
// list.
func TestUpsertKeepsToolsNarrowedByUser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp-config.json")
	entry := testEntry(testCmd, testArgs("test"), nil)
	entry["tools"] = []string{"search"}
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
	entries, err := readAgent(t.Context(), a)
	require.NoError(t, err)
	assert.Equal(t, []any{"search"}, entries[testName]["tools"])
}

// TestUpsertPatchesOnlyChangedKeysJSON checks that an update writes only the
// launch keys that changed. The other members keep their bytes: comments, key
// order, number text and the user's keys. Only the command and args are replaced,
// in place.
func TestUpsertPatchesOnlyChangedKeysJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	before := `{
  "mcpServers": {
    "` + testName + `": {
      "disabled": false,
      // written by hand
      "timeout": 1.50,
      "command": "/old/databricks",
      "args": ["aitools", "mcp", "serve", "--profile", "old"],
      "big": 12345678901234567890
    },
    "other": {"command": "npx"}
  }
}
`
	writeFile(t, path, []byte(before), 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	assert.Equal(t, `{
  "mcpServers": {
    "`+testName+`": {
      "disabled": false,
      // written by hand
      "timeout": 1.50,
      "command": "/opt/databricks/bin/databricks",
      "args": [
        "aitools",
        "mcp",
        "serve",
        "--profile",
        "dogfood",
        "--service",
        "system.ai.genie_one_mcp",
        "--agent",
        "test"
      ],
      "big": 12345678901234567890
    },
    "other": {"command": "npx"}
  }
}
`, string(readFile(t, path)))
}

// TestUpsertUpdateAddsAndRemovesLaunchKeys checks that a launch key that the file
// lacks is added on update, and one that the entry no longer has is removed.
// Either way the file stays strict JSON, whether the key was the last member or
// not.
func TestUpsertUpdateAddsAndRemovesLaunchKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, path, []byte(`{"mcpServers":{"`+testName+`":{"command":"`+testCmd+`","args":["aitools","mcp","serve","--agent","test"]}}}`), 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), map[string]string{mcpconfig.ConfigFileEnv: "/cfg"}, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	entries, err := readAgent(t.Context(), a)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{mcpconfig.ConfigFileEnv: "/cfg"}, entries[testName]["env"])
	requireStrictJSON(t, string(readFile(t, path)))

	_, status, err = Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	entries, err = readAgent(t.Context(), a)
	require.NoError(t, err)
	assert.NotContains(t, entries[testName], "env")
	requireStrictJSON(t, string(readFile(t, path)))
}

// TestTOMLUpdateKeepsUserTables checks that keys of an entry we wrote that the
// user keeps in TOML are kept on update, including sub-tables such as
// [mcp_servers.<name>.tools.<tool>].
func TestTOMLUpdateKeepsUserTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	data := testBlock + "\n[mcp_servers." + testName + ".tools.search]\napproval_mode = \"approve\"\n"
	writeFile(t, path, []byte(data), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("other"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	entries, err := readAgent(t.Context(), a)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"search": map[string]any{"approval_mode": "approve"}}, entries[testName]["tools"])
}

// TestUpsertAndRemoveRefuseDuplicateKeysJSON checks that duplicate member names on
// the path to the server are refused. Agents read the last of two equal names
// while an edit would change the first, so the file is left alone.
func TestUpsertAndRemoveRefuseDuplicateKeysJSON(t *testing.T) {
	argsJSON, err := json.Marshal(testArgs("test"))
	require.NoError(t, err)
	ours := `{"command": "` + testCmd + `", "args": ` + string(argsJSON) + `}`
	for _, data := range []string{
		`{"mcpServers": {}, "mcpServers": {"` + testName + `": ` + ours + `}}`,
		`{"mcpServers": {"` + testName + `": ` + ours + `, "` + testName + `": {"command": "/usr/bin/other", "args": []}}}`,
		`{"mcpServers": {"` + testName + `": {"command": "` + testCmd + `", "args": ` + string(argsJSON) + `, "command": "/usr/bin/other"}}}`,
	} {
		t.Run(data, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			writeFile(t, path, []byte(data), 0o600)
			a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

			_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, true)
			require.ErrorIs(t, err, ErrConfigInvalid)
			_, err = removeAgent(t.Context(), a, testName)
			require.ErrorIs(t, err, ErrConfigInvalid)
			assert.Equal(t, data, string(readFile(t, path)))
		})
	}
}

// TestUpsertNonObjectTopLevelNamesProblemOnce checks that a file whose top-level
// value is not an object names the problem once.
func TestUpsertNonObjectTopLevelNamesProblemOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, path, []byte(`[]`), 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.ErrorIs(t, err, ErrConfigInvalid)
	assert.Equal(t, 1, strings.Count(err.Error(), ErrConfigInvalid.Error()), err.Error())
}

// TestRemoveRefusesInvalidServerName checks that Remove validates the name the
// way Upsert does, so an invalid name is an error and nothing is read as a match.
func TestRemoveRefusesInvalidServerName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := `{"mcpServers":{"` + testName + `":{"command":"` + testCmd + `","args":[]}}}`
	writeFile(t, path, []byte(data), 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	_, err := removeAgent(t.Context(), a, "genie.one")
	assert.ErrorContains(t, err, "must contain only letters, digits, '-' and '_'")
	assert.Equal(t, data, string(readFile(t, path)))
}

// entryOf returns the entry of server name in agent a's config.
func entryOf(t *testing.T, ctx context.Context, a *agents.Agent, name string) map[string]any {
	t.Helper()
	entries, err := readAgent(ctx, a)
	require.NoError(t, err)
	require.Contains(t, entries, name)
	return entries[name]
}

// TestUpsertKeepsUserEnvJSON checks that a variable the user added to env
// survives an update, with its bytes. The pinned variable takes the value the
// caller passes, and is removed when the caller passes none.
func TestUpsertKeepsUserEnvJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp_config.json")
	entry := testEntry(testCmd, testArgs("test"), map[string]string{"DATABRICKS_CONFIG_FILE": "/old.cfg", "HTTPS_PROXY": "http://proxy.test"})
	data, err := json.Marshal(map[string]any{"mcpServers": map[string]any{testName: entry}})
	require.NoError(t, err)
	writeFile(t, path, data, 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), map[string]string{"DATABRICKS_CONFIG_FILE": "/new.cfg"}, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	assert.Equal(t, map[string]any{"DATABRICKS_CONFIG_FILE": "/new.cfg", "HTTPS_PROXY": "http://proxy.test"}, entryOf(t, t.Context(), a, testName)["env"])
	assert.Contains(t, string(readFile(t, path)), `"HTTPS_PROXY":"http://proxy.test"`)

	_, status, err = Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	assert.Equal(t, map[string]any{"HTTPS_PROXY": "http://proxy.test"}, entryOf(t, t.Context(), a, testName)["env"])
	requireStrictJSON(t, string(readFile(t, path)))
}

// TestUpsertKeepsUserEnvTOML is TestUpsertKeepsUserEnvJSON for a TOML env table.
func TestUpsertKeepsUserEnvTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	before := testBlock + "\n[mcp_servers." + testName + ".env]\nDATABRICKS_CONFIG_FILE = \"/old.cfg\"\nHTTPS_PROXY = \"http://proxy.test\"\n"
	writeFile(t, path, []byte(before), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), map[string]string{"DATABRICKS_CONFIG_FILE": "/new.cfg"}, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	assert.Equal(t, map[string]any{"DATABRICKS_CONFIG_FILE": "/new.cfg", "HTTPS_PROXY": "http://proxy.test"}, entryOf(t, t.Context(), a, testName)["env"])

	_, status, err = Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	assert.Equal(t, map[string]any{"HTTPS_PROXY": "http://proxy.test"}, entryOf(t, t.Context(), a, testName)["env"])
}

// TestUpsertKeepsUserEnvironmentOpenCode is TestUpsertKeepsUserEnvJSON for
// OpenCode, whose variables are under "environment".
func TestUpsertKeepsUserEnvironmentOpenCode(t *testing.T) {
	ctx := testHome(t)
	a := agents.ByName(agents.NameOpenCode)
	path, err := a.MCP.Path(ctx)
	require.NoError(t, err)
	entry := map[string]any{
		"type":        "local",
		"command":     append([]string{testCmd}, testArgs("test")...),
		"environment": map[string]string{"DATABRICKS_CONFIG_FILE": "/old.cfg", "HTTPS_PROXY": "http://proxy.test"},
	}
	data, err := json.Marshal(map[string]any{"mcp": map[string]any{"servers": map[string]any{testName: entry}}})
	require.NoError(t, err)
	writeFile(t, path, data, 0o600)

	_, status, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	assert.Equal(t, map[string]any{"HTTPS_PROXY": "http://proxy.test"}, entryOf(t, ctx, a, testName)["environment"])
}

// TestUpsertUnpinKeepsEnvClaude checks that unpinning the only variable leaves the
// env table of a Claude Code entry in place, since claude mcp add always writes
// it. The next identical write then reports unchanged.
func TestUpsertUnpinKeepsEnvClaude(t *testing.T) {
	ctx := testHome(t)
	a := agents.ByName(agents.NameClaudeCode)
	_, _, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), map[string]string{mcpconfig.ConfigFileEnv: "/cfg"}, false)
	require.NoError(t, err)

	_, status, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	assert.Equal(t, map[string]any{}, entryOf(t, ctx, a, testName)["env"])

	_, status, err = Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUnchanged, status)
}

// TestUpsertPinIntoEmptyEnvLayout checks that pinning a variable into an existing
// empty env table keeps the closing brace on its own line, as a new env is laid out.
func TestUpsertPinIntoEmptyEnvLayout(t *testing.T) {
	ctx := testHome(t)
	a := agents.ByName(agents.NameClaudeCode)
	path, err := a.MCP.Path(ctx)
	require.NoError(t, err)
	args, err := json.Marshal(testArgs("test"))
	require.NoError(t, err)
	writeFile(t, path, []byte(`{"mcpServers":{"`+testName+`":{"type":"stdio","command":"`+testCmd+`","args":`+string(args)+`,"env":{}}}}`), 0o600)

	_, status, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), map[string]string{mcpconfig.ConfigFileEnv: "/cfg"}, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	assert.Equal(t, map[string]any{mcpconfig.ConfigFileEnv: "/cfg"}, entryOf(t, ctx, a, testName)["env"])
	assert.Regexp(t, `"/cfg"\n\s+}`, string(readFile(t, path)))
	requireStrictJSON(t, string(readFile(t, path)))
}

// TestUpsertRejectsEnvOtherThanPin checks that Upsert refuses an env variable other
// than the pinned one, since such a variable could never be removed by a later write.
func TestUpsertRejectsEnvOtherThanPin(t *testing.T) {
	ctx := testHome(t)
	a := agents.ByName(agents.NameClaudeCode)
	_, _, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), map[string]string{"HTTPS_PROXY": "http://proxy.test"}, false)
	assert.ErrorContains(t, err, "env may only set "+mcpconfig.ConfigFileEnv)
}

// TestUpsertRejectsDuplicateEnvKey checks that a duplicate key in an env table is
// refused before any write, as a duplicate server key is.
func TestUpsertRejectsDuplicateEnvKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp_config.json")
	args, err := json.Marshal(testArgs("test"))
	require.NoError(t, err)
	data := `{"mcpServers":{"` + testName + `":{"command":"` + testCmd + `","args":` + string(args) + `,"env":{"DATABRICKS_CONFIG_FILE":"/a","DATABRICKS_CONFIG_FILE":"/b"}}}}`
	writeFile(t, path, []byte(data), 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	_, _, err = Upsert(t.Context(), a, testName, testCmd, testArgs("test"), map[string]string{mcpconfig.ConfigFileEnv: "/new"}, false)
	assert.ErrorContains(t, err, "duplicate key")
	assert.Equal(t, data, string(readFile(t, path)))
}

// TestRemoveInlineServerTOMLKeepsFile checks that Remove refuses a server defined
// inline under [mcp_servers], as Upsert does, and leaves the table and its comment.
func TestRemoveInlineServerTOMLKeepsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	before := "[mcp_servers]\n# note\n" + testName + ` = { command = "` + testCmd + `", args = ["aitools", "mcp", "serve"] }` + "\n"
	writeFile(t, path, []byte(before), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, err := RemoveAt(a, path, testName)
	assert.ErrorContains(t, err, "edit it by hand")
	assert.Equal(t, before, string(readFile(t, path)))
}

// TestUpsertArrayElementIsNotHeaderTOML checks that an array element that looks like
// a table header, here ["mcp_servers"], does not stand in for the root table. An
// inline root must still refuse a new server.
func TestUpsertArrayElementIsNotHeaderTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	before := "mcp_servers = {}\nother = [\n  [\"mcp_servers\"]\n]\n"
	writeFile(t, path, []byte(before), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	assert.ErrorContains(t, err, "edit it by hand")
	assert.Equal(t, before, string(readFile(t, path)))
}

// TestRemoveArrayElementInsideServerTOML checks that an array inside a server's table,
// whose last element looks like a table header, is removed with the server.
func TestRemoveArrayElementInsideServerTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, path, []byte(testBlock+"extra = [\n  [\"c\"]\n]\n"), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	removed, err := RemoveAt(a, path, testName)
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Empty(t, readFile(t, path))
}

// TestUpsertKeepsTypeWhenBuilderDoesNotWriteIt checks that a "type" key is the
// user's for agents whose builder never writes one, so an update keeps it.
func TestUpsertKeepsTypeWhenBuilderDoesNotWriteIt(t *testing.T) {
	for _, name := range []string{agents.NameCursor, agents.NameGemini, agents.NameAntigravity, agents.NameCodex} {
		t.Run(name, func(t *testing.T) {
			ctx := testHome(t)
			a := agents.ByName(name)
			path, err := a.MCP.Path(ctx)
			require.NoError(t, err)
			var before []byte
			if a.MCP.Format == mcpconfig.FormatTOML {
				before = []byte(testBlock + "type = \"stdio\"\n")
			} else {
				before, err = json.Marshal(map[string]any{"mcpServers": map[string]any{testName: map[string]any{"command": testCmd, "args": testArgs("test"), "type": "stdio"}}})
				require.NoError(t, err)
			}
			writeFile(t, path, before, 0o600)

			_, status, err := Upsert(ctx, a, testName, testCmd, testArgs("other"), nil, false)
			require.NoError(t, err)
			assert.Equal(t, StatusUpdated, status)
			entry := entryOf(t, ctx, a, testName)
			assert.Equal(t, "stdio", entry["type"])
			assert.Contains(t, entry["args"], "other")
		})
	}
}

// TestUpsertRefusesDanglingOpenCodeLink checks that a dangling opencode.jsonc link
// is refused, and that opencode.json is not created beside it.
func TestUpsertRefusesDanglingOpenCodeLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs privileges on Windows")
	}
	ctx := testHome(t)
	a := agents.ByName(agents.NameOpenCode)
	path, err := a.MCP.Path(ctx)
	require.NoError(t, err)
	dir := filepath.Dir(path)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	jsonc := filepath.Join(dir, "opencode.jsonc")
	require.NoError(t, os.Symlink(filepath.Join(dir, "missing.jsonc"), jsonc))

	_, _, err = Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	assert.ErrorContains(t, err, "does not exist")
	_, err = os.Lstat(filepath.Join(dir, "opencode.json"))
	assert.ErrorIs(t, err, os.ErrNotExist)
	info, err := os.Lstat(jsonc)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink, "link is kept")
}

// TestOpenCodeV1LayoutIsRefused checks that a write refuses an OpenCode config
// whose mcp table holds V1 servers. A V2 mcp.servers table added beside them
// would be read by a V1 OpenCode as a server named servers. The V2 mcp.timeout
// object is not a server, so it does not block a write.
func TestOpenCodeV1LayoutIsRefused(t *testing.T) {
	a := agents.ByName(agents.NameOpenCode)
	tests := []struct {
		name    string
		body    string
		refused bool
	}{
		{"v1 local server", `{"mcp":{"playwright":{"type":"local","command":["npx","@playwright/mcp"],"enabled":true}}}`, true},
		{"v1 server with command only", `{"mcp":{"playwright":{"command":["npx"]}}}`, true},
		{"v1 server beside v2 table", `{"mcp":{"servers":{},"playwright":{"type":"remote","url":"https://mcp.example.test/mcp"}}}`, true},
		{"v2 servers and timeout", `{"mcp":{"servers":{"other":{"type":"local","command":["npx"]}},"timeout":{"catalog":30000,"execution":30000}}}`, false},
		{"v2 servers only", `{"mcp":{"servers":{}}}`, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testHome(t)
			path, err := a.MCP.Path(ctx)
			require.NoError(t, err)
			writeFile(t, path, []byte(tc.body), 0o600)

			_, _, upsertErr := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
			_, removeErr := RemoveAt(a, path, testName)
			if !tc.refused {
				require.NoError(t, upsertErr)
				require.NoError(t, removeErr)
				return
			}
			assert.ErrorContains(t, upsertErr, "older layout")
			assert.ErrorContains(t, upsertErr, `"mcp.servers"`)
			assert.ErrorContains(t, removeErr, "older layout")
			assert.Equal(t, tc.body, string(readFile(t, path)))
		})
	}
}

// TestUpsertRefusesEnvThatIsNotATable checks that an env value that is not a table
// is refused and not overwritten.
func TestUpsertRefusesEnvThatIsNotATable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := `{"mcpServers":{"` + testName + `":{"command":"` + testCmd + `","args":["aitools","mcp","serve","--agent","test"],"env":"x"}}}`
	writeFile(t, path, []byte(data), 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), map[string]string{mcpconfig.ConfigFileEnv: "/cfg"}, false)
	require.ErrorIs(t, err, ErrConfigInvalid)
	assert.Equal(t, data, string(readFile(t, path)))
}

// TestWriteRefusesReadOnlyFile checks that an add or a remove does not write a
// config file the user made read-only. The file keeps its bytes and mode. An add
// that needs no write still succeeds.
func TestWriteRefusesReadOnlyFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes read-only files")
	}
	path := filepath.Join(t.TempDir(), "config.json")
	// Windows maps the mode to the read-only attribute. Writable again before the
	// temporary directory is removed.
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	writeFile(t, path, []byte(`{"mcpServers":{"other":{"command":"npx"}}}`), 0o600)
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")
	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)

	require.NoError(t, os.Chmod(path, 0o400))
	before := readFile(t, path)
	refused := filepath.ToSlash(path) + " is read-only; make it writable"

	_, _, err = Upsert(t.Context(), a, testName, testCmd, testArgs("changed"), nil, false)
	require.ErrorIs(t, err, ErrReadOnly)
	require.ErrorIs(t, err, fs.ErrPermission)
	require.ErrorContains(t, err, refused)
	_, err = RemoveAt(a, path, testName)
	require.ErrorIs(t, err, ErrReadOnly)
	require.ErrorContains(t, err, refused)
	assert.Equal(t, before, readFile(t, path))
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o400), info.Mode().Perm())
	}

	_, status, err = Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUnchanged, status)
}

// lockedAgent is an agent whose config is locked the way Claude Code locks its own.
func lockedAgent(path string) *agents.Agent {
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")
	a.MCP.Lock = true
	return a
}

func TestConfigLockPath(t *testing.T) {
	dir := t.TempDir()

	existing := filepath.Join(dir, ".claude.json")
	writeFile(t, existing, []byte(`{}`), 0o600)
	got, err := configLock(existing)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, ".claude.json.lock"), got)

	missing := filepath.Join(dir, "missing", ".claude.json")
	got, err = configLock(missing)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "missing", ".claude.json.lock"), got)
}

// TestConfigLockPathIsTheLinkPath checks that a symlinked config is locked next to the link, where Claude Code locks it,
// not next to its target.
func TestConfigLockPathIsTheLinkPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs privileges on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles", "claude.json")
	writeFile(t, target, []byte(`{}`), 0o600)
	link := filepath.Join(dir, ".claude.json")
	require.NoError(t, os.Symlink(target, link))

	got, err := configLock(link)
	require.NoError(t, err)
	assert.Equal(t, link+".lock", got)
}

// TestSymlinkedConfigWaitsForClaudesLock checks that an edit of a symlinked config waits for the lock Claude Code holds
// next to the link.
func TestSymlinkedConfigWaitsForClaudesLock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs privileges on Windows")
	}
	defer func(prev time.Duration) { lockTimeout = prev }(lockTimeout)
	lockTimeout = 200 * time.Millisecond
	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles", "claude.json")
	writeFile(t, target, []byte(`{}`), 0o600)
	link := filepath.Join(dir, ".claude.json")
	require.NoError(t, os.Symlink(target, link))
	require.NoError(t, os.Mkdir(link+".lock", 0o700)) // what a live Claude Code holds
	t.Cleanup(func() { _ = os.Remove(link + ".lock") })

	_, _, err := Upsert(t.Context(), lockedAgent(link), testName, testCmd, testArgs("test"), nil, false)
	assert.ErrorContains(t, err, "is locked by another process")
}

// TestLockedEditHoldsLockUntilReleased checks that the lock directory exists
// while it is held and is gone once it is released. Upsert has no point between
// the lock and the save where a test could look, so this checks the lock itself.
func TestLockedEditHoldsLockUntilReleased(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	lock, err := configLock(path)
	require.NoError(t, err)

	release, err := lockConfig(path)
	require.NoError(t, err)
	info, err := os.Stat(lock)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	release()
	_, err = os.Stat(lock)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestUpsertAndRemoveReleaseLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	lock, err := configLock(path)
	require.NoError(t, err)
	a := lockedAgent(path)

	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	_, err = os.Stat(lock)
	assert.ErrorIs(t, err, fs.ErrNotExist)

	removed, err := RemoveAt(a, path, testName)
	require.NoError(t, err)
	assert.True(t, removed)
	_, err = os.Stat(lock)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestRemoveWithoutConfigDirCreatesNothing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "claude-home")
	a := lockedAgent(filepath.Join(dir, ".claude.json"))

	removed, err := RemoveAt(a, filepath.Join(dir, ".claude.json"), testName)
	require.NoError(t, err)
	assert.False(t, removed)
	_, err = os.Stat(dir)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestUnlockedAgentIgnoresLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	lock, err := configLock(path)
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(lock, 0o700))
	t.Cleanup(func() { _ = os.Remove(lock) })

	// A lock held by someone else does not stop an agent that does not take one.
	_, status, err := Upsert(t.Context(), testAgent(path, mcpconfig.FormatJSON, "mcpServers"), testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
}

// TestClaudeLegacyConfigIsLockedAndWritten checks that when Claude Code reads its
// legacy config, an edit writes that file under the lock on it, and leaves
// ~/.claude.json alone.
func TestClaudeLegacyConfigIsLockedAndWritten(t *testing.T) {
	defer func(prev time.Duration) { lockTimeout = prev }(lockTimeout)
	lockTimeout = 200 * time.Millisecond

	ctx := testHome(t)
	home, err := env.UserHomeDir(ctx)
	require.NoError(t, err)
	legacy := filepath.Join(home, ".claude", ".config.json")
	writeFile(t, legacy, []byte(`{"mcpServers":{"other":{"command":"npx"}}}`), 0o600)
	a := agents.ByName(agents.NameClaudeCode)

	lock, err := configLock(legacy)
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(lock, 0o700))
	_, _, err = Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.ErrorContains(t, err, filepath.ToSlash(lock)+" is locked by another process")

	require.NoError(t, os.Remove(lock))
	path, status, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, legacy, path)
	assert.Equal(t, StatusAdded, status)
	_, err = os.Stat(filepath.Join(home, ".claude.json"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

// TestLockedEditTimesOutOnLiveLock checks that an edit waits for a lock that
// another process holds, and fails naming the lock once the wait is over. The
// lock is not ours, so it is left in place.
func TestLockedEditTimesOutOnLiveLock(t *testing.T) {
	defer func(prev time.Duration) { lockTimeout = prev }(lockTimeout)
	lockTimeout = 200 * time.Millisecond

	path := filepath.Join(t.TempDir(), ".claude.json")
	writeFile(t, path, []byte(`{"mcpServers":{"other":{"command":"npx"}}}`), 0o600)
	before := readFile(t, path)
	lock, err := configLock(path)
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(lock, 0o700))
	a := lockedAgent(path)

	start := time.Now()
	_, _, err = Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.ErrorContains(t, err, filepath.ToSlash(lock)+" is locked by another process")
	assert.GreaterOrEqual(t, time.Since(start), lockTimeout)

	_, err = RemoveAt(a, path, testName)
	require.ErrorContains(t, err, filepath.ToSlash(lock)+" is locked by another process")

	_, err = os.Stat(lock)
	require.NoError(t, err, "the lock is not ours to remove")
	assert.Equal(t, before, readFile(t, path))
}

// TestLockedEditWaitsForRelease checks that an edit goes ahead once the other
// process lets go of the lock while the edit waits.
func TestLockedEditWaitsForRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	lock, err := configLock(path)
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(lock, 0o700))
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = os.Remove(lock)
	}()

	_, status, err := Upsert(t.Context(), lockedAgent(path), testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
}

// TestLockedEditTakesOverStaleLock checks that a lock left by a process that died
// is removed once its mtime is older than lockStaleAfter.
func TestLockedEditTakesOverStaleLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	lock, err := configLock(path)
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(lock, 0o700))
	old := time.Now().Add(-(lockStaleAfter + time.Second))
	require.NoError(t, os.Chtimes(lock, old, old))

	_, status, err := Upsert(t.Context(), lockedAgent(path), testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	_, err = os.Stat(lock)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

// claudeLikeWriter saves the config the way Claude Code does: it takes the lock,
// reads the file, adds one to numStartups, writes a temp file and renames it over
// the config, then releases the lock. saves counts the renames that landed. It
// runs until stop is closed.
func claudeLikeWriter(path, lock string, stop <-chan struct{}, saves *atomic.Int64) {
	for {
		select {
		case <-stop:
			return
		default:
		}
		if os.Mkdir(lock, 0o700) != nil {
			time.Sleep(100 * time.Microsecond)
			continue
		}
		data, _ := os.ReadFile(path)
		m := map[string]any{}
		_ = json.Unmarshal(data, &m)
		n, _ := m["numStartups"].(float64)
		m["numStartups"] = n + 1
		time.Sleep(2 * time.Millisecond)
		out, _ := json.MarshalIndent(m, "", "  ")
		if os.WriteFile(path+".tmp", out, 0o600) == nil && os.Rename(path+".tmp", path) == nil {
			saves.Add(1)
		}
		_ = os.Remove(lock)
		time.Sleep(time.Millisecond)
	}
}

// TestLockedEditLosesNoUpdates runs edits against a config that Claude Code
// writes at the same time. Neither side may lose the other's update: every save
// by Claude Code is counted in the file, and every upsert is in the file when it
// returns.
func TestLockedEditLosesNoUpdates(t *testing.T) {
	const iterations = 100
	path := filepath.Join(t.TempDir(), ".claude.json")
	// A large config makes each save slow enough for the race to show.
	projects := map[string]any{}
	for i := range 800 {
		projects[fmt.Sprintf("/work/project-%03d", i)] = map[string]any{"history": strings.Repeat("x", 500)}
	}
	seed, err := json.MarshalIndent(map[string]any{"numStartups": 0, "projects": projects}, "", "  ")
	require.NoError(t, err)
	writeFile(t, path, seed, 0o600)

	// Claude Code locks the config's path as given, as configLock does. The test
	// builds the name on its own, so the lock under test is not checked against itself.
	abs, err := filepath.Abs(path)
	require.NoError(t, err)

	var saves atomic.Int64
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		claudeLikeWriter(path, abs+".lock", stop, &saves)
	})
	stopWriter := sync.OnceFunc(func() {
		close(stop)
		wg.Wait()
	})
	defer stopWriter()

	a := lockedAgent(path)
	for i := range iterations {
		agent := strconv.Itoa(i)
		_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs(agent), nil, false)
		require.NoError(t, err)
		entries, err := ReadAt(a, path)
		require.NoError(t, err)
		require.Contains(t, entries, testName, "upsert %d", i)
		assert.Equal(t, testArgs(agent), toStrings(t, entries[testName]["args"]), "upsert %d", i)
	}

	stopWriter()
	var final map[string]any
	require.NoError(t, json.Unmarshal(readFile(t, path), &final))
	numStartups, _ := final["numStartups"].(float64)
	assert.Equal(t, saves.Load(), int64(numStartups), "Claude Code saves that were lost")
}

// TestPathIsTheFileTheSpecNames checks that Path returns the file the spec names, without creating or reading it.
func TestPathIsTheFileTheSpecNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	a := testAgent(path, mcpconfig.FormatJSON, "mcpServers")

	got, err := Path(t.Context(), a)
	require.NoError(t, err)
	assert.Equal(t, path, got)
	_, err = os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestPathFailsForAgentWithoutMCP(t *testing.T) {
	_, err := Path(t.Context(), &agents.Agent{Name: "pi", DisplayName: "Pi"})
	require.Error(t, err)
}

// TestReadAtAndRemoveAtUseTheGivenFile checks that ReadAt and RemoveAt work on the file they are given, not on the
// file the agent's spec names now. The entry is written to one file, and the spec names another.
func TestReadAtAndRemoveAtUseTheGivenFile(t *testing.T) {
	recorded := filepath.Join(t.TempDir(), "recorded.json")
	current := filepath.Join(t.TempDir(), "current.json")
	_, _, err := Upsert(t.Context(), testAgent(recorded, mcpconfig.FormatJSON, "mcpServers"), testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	a := testAgent(current, mcpconfig.FormatJSON, "mcpServers")

	entries, err := ReadAt(a, recorded)
	require.NoError(t, err)
	assert.Contains(t, entries, testName)

	entries, err = readAgent(t.Context(), a)
	require.NoError(t, err)
	assert.Empty(t, entries, "the spec's file does not have the entry")

	removed, err := RemoveAt(a, recorded, testName)
	require.NoError(t, err)
	assert.True(t, removed)
	entries, err = ReadAt(a, recorded)
	require.NoError(t, err)
	assert.Empty(t, entries)
	_, err = os.Stat(current)
	assert.ErrorIs(t, err, os.ErrNotExist, "RemoveAt does not write the spec's file")
}

// TestReadAtMissingFileHasNoEntries checks that a file that no longer exists reads as having no entries.
func TestReadAtMissingFileHasNoEntries(t *testing.T) {
	a := testAgent(filepath.Join(t.TempDir(), "current.json"), mcpconfig.FormatJSON, "mcpServers")
	entries, err := ReadAt(a, filepath.Join(t.TempDir(), "gone.json"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// readAgent reads the entries of agent a's config at the path Path names, as the commands read the config.
func readAgent(ctx context.Context, a *agents.Agent) (map[string]map[string]any, error) {
	path, err := Path(ctx, a)
	if err != nil {
		return nil, err
	}
	return ReadAt(a, path)
}

// removeAgent removes server name from agent a's config at the path Path names.
func removeAgent(ctx context.Context, a *agents.Agent, name string) (bool, error) {
	path, err := Path(ctx, a)
	if err != nil {
		return false, err
	}
	return RemoveAt(a, path, name)
}

// TestCodexRoundTripKeepsOtherTables checks that adding and removing our server leaves every other table of a Codex
// config byte-identical: a server with a comment inside its table, a multi-line array, an enabled key and CRLF line
// endings.
func TestCodexRoundTripKeepsOtherTables(t *testing.T) {
	ctx := testHome(t)
	a := agents.ByName(agents.NameCodex)
	path, err := a.MCP.Path(ctx)
	require.NoError(t, err)
	before := "# Codex settings.\r\n" +
		"model = \"gpt-5\"\r\n" +
		"\r\n" +
		"[mcp_servers.docs]\r\n" +
		"# keep this comment\r\n" +
		"command = \"npx\"\r\n" +
		"args = [\r\n" +
		"    \"-y\",\r\n" +
		"    \"docs-server\",\r\n" +
		"]\r\n" +
		"enabled = true\r\n" +
		"timeout = 10\r\n"
	writeFile(t, path, []byte(before), 0o600)

	_, status, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	added := string(readFile(t, path))
	assert.True(t, strings.HasPrefix(added, before), "the other tables must keep their bytes:\n%s", added)

	removed, err := RemoveAt(a, path, testName)
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Equal(t, before, string(readFile(t, path)))
}
