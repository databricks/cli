package agentconfig

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testBlock is the table block Upsert writes for testName with testArgs("test").
const testBlock = "[mcp_servers." + testName + "]\n" +
	`args = ["aitools", "mcp", "serve", "--profile", "dogfood", "--service", "system.ai.genie_one_mcp", "--agent", "test"]` + "\n" +
	`command = "` + testCmd + `"` + "\n"

func TestTOMLAddKeepsCommentsAndLayout(t *testing.T) {
	before := "# Codex settings.\nmodel = \"gpt-5\" # default model\n\n[projects.\"/work\"]\ntrust_level = \"trusted\"\n"
	path := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, path, []byte(before), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	assert.Equal(t, before+"\n"+testBlock, string(readFile(t, path)))
}

func TestTOMLUpdateReplacesOnlyOwnedTables(t *testing.T) {
	before := "[mcp_servers." + testName + "]\n" +
		"# old settings\n" +
		"command = \"/old/databricks\"\n" +
		"args = [\"aitools\", \"mcp\", \"serve\", \"--profile\", \"old\"]\n" +
		"\n" +
		"# about other\n" +
		"[mcp_servers.other]\n" +
		"command = \"npx\"\n"
	path := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, path, []byte(before), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	assert.Equal(t, testBlock+"\n# about other\n[mcp_servers.other]\ncommand = \"npx\"\n", string(readFile(t, path)))
}

func TestTOMLRemoveDeletesOnlyOwnedTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	before := "# keep\nmodel = \"x\"\n\n" + testBlock + "\n[mcp_servers.other]\ncommand = \"npx\"\n"
	writeFile(t, path, []byte(before), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	removed, err := Remove(t.Context(), a, testName)
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Equal(t, "# keep\nmodel = \"x\"\n\n[mcp_servers.other]\ncommand = \"npx\"\n", string(readFile(t, path)))
}

// A server defined inline or with dotted keys is not rewritten. The error names
// the file and the server.
func TestTOMLRefusesServerWithoutOwnTables(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
	}{
		{"inline table", "[mcp_servers]\n" + testName + ` = { command = "/usr/bin/other", args = [] }` + "\n"},
		{"dotted keys", "[mcp_servers]\n" + testName + `.command = "/usr/bin/other"` + "\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			writeFile(t, path, []byte(tc.data), 0o600)
			a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

			_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, true)
			require.Error(t, err)
			assert.ErrorContains(t, err, path)
			assert.ErrorContains(t, err, testName)
			assert.Equal(t, tc.data, string(readFile(t, path)))

			_, err = Remove(t.Context(), a, testName)
			require.Error(t, err)
			assert.Equal(t, tc.data, string(readFile(t, path)))
		})
	}
}

// A root table written inline cannot take a [mcp_servers.<name>] table. The
// decoder would accept it, so the edit is refused before anything is written.
func TestTOMLRefusesInlineRootTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	data := "mcp_servers = {}\n"
	writeFile(t, path, []byte(data), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	assert.ErrorContains(t, err, "defined inline")
	assert.Equal(t, data, string(readFile(t, path)))
}

// The check after an edit compares everything but the server, so a change to
// other settings is refused even when the server itself is right.
func TestCheckEditRejectsOtherChanges(t *testing.T) {
	before := map[string]any{"model": "a", "mcp_servers": map[string]any{}}
	after := map[string]any{"model": "b", "mcp_servers": map[string]any{testName: testEntry(testCmd, testArgs("test"), nil)}}
	err := checkEdit(before, after, []string{"mcp_servers"}, testName, testEntry(testCmd, testArgs("test"), nil))
	assert.ErrorContains(t, err, "changed other settings")

	after["model"] = "a"
	assert.NoError(t, checkEdit(before, after, []string{"mcp_servers"}, testName, testEntry(testCmd, testArgs("test"), nil)))
	assert.ErrorContains(t, checkEdit(before, after, []string{"mcp_servers"}, testName, nil), "did not remove")
}

func TestTOMLKeepsBOMAndCRLF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, path, []byte("\xEF\xBB\xBF# c\r\nmodel = \"x\"\r\n"), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	got := string(readFile(t, path))
	assert.True(t, strings.HasPrefix(got, "\xEF\xBB\xBF# c\r\nmodel = \"x\"\r\n\r\n[mcp_servers."), got)
	assert.NotContains(t, strings.ReplaceAll(got, "\r\n", ""), "\n", "every newline is CRLF")
}

func TestTOMLEditOnlyTouchesServerBlockLines(t *testing.T) {
	lines := splitLines([]byte("a\nb\n"))
	assert.Len(t, lines, 2)
	assert.Equal(t, "b\n", string(lines[1]))
	_, ok := tomlTableKey("  [[mcp_servers.x]] # c")
	assert.True(t, ok)
	_, ok = tomlTableKey(`key = "[x]"`)
	assert.False(t, ok)
}
