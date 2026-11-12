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

// TestTOMLRefusesServerWithoutOwnTables checks that a server defined inline or
// with dotted keys is not rewritten. The error names the file and the server.
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
			assert.ErrorContains(t, err, filepath.ToSlash(path))
			assert.ErrorContains(t, err, testName)
			assert.Equal(t, tc.data, string(readFile(t, path)))

			_, err = Remove(t.Context(), a, testName)
			require.Error(t, err)
			assert.Equal(t, tc.data, string(readFile(t, path)))
		})
	}
}

// TestTOMLRefusesInlineRootTable checks that a root table written inline cannot
// take a [mcp_servers.<name>] table. The decoder would accept it, so the edit is
// refused before anything is written.
func TestTOMLRefusesInlineRootTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	data := "mcp_servers = {}\n"
	writeFile(t, path, []byte(data), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	assert.ErrorContains(t, err, "defined inline")
	assert.Equal(t, data, string(readFile(t, path)))
}

// TestCheckEditRejectsOtherChanges checks that the check after an edit compares
// everything but the server, so a change to other settings is refused even when
// the server itself is right.
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

// blockFor is the table block Upsert writes for testName with the given agent's args.
func blockFor(agent string) string {
	return "[mcp_servers." + testName + "]\n" +
		`args = ["aitools", "mcp", "serve", "--profile", "dogfood", "--service", "system.ai.genie_one_mcp", "--agent", "` + agent + `"]` + "\n" +
		`command = "` + testCmd + `"` + "\n"
}

// requireTidyTOML fails when the layout has a double blank line or a blank first
// line.
func requireTidyTOML(t *testing.T, got string) {
	t.Helper()
	require.NotContains(t, got, "\n\n\n", "no double blank line")
	require.False(t, strings.HasPrefix(got, "\n"), "no blank first line")
}

// TestTOMLEnvTablesLayout checks that env sub-tables are written, changed, dropped
// and removed with the server.
func TestTOMLEnvTablesLayout(t *testing.T) {
	other := "[mcp_servers.other]\ncommand = \"npx\"\n"
	envTable := "[mcp_servers." + testName + ".env]\nDATABRICKS_CONFIG_FILE = \"one\"\n"

	t.Run("add with env", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		writeFile(t, path, []byte(other), 0o600)
		a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

		_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), map[string]string{"DATABRICKS_CONFIG_FILE": "one"}, false)
		require.NoError(t, err)
		assert.Equal(t, StatusAdded, status)
		got := string(readFile(t, path))
		requireTidyTOML(t, got)
		entries, err := Read(t.Context(), a)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"DATABRICKS_CONFIG_FILE": "one"}, entries[testName]["env"])
		assert.Contains(t, entries, "other")
	})

	t.Run("change env", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		writeFile(t, path, []byte(blockFor("test")+"\n"+envTable+"\n"+other), 0o600)
		a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

		_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), map[string]string{"DATABRICKS_CONFIG_FILE": "two"}, false)
		require.NoError(t, err)
		assert.Equal(t, StatusUpdated, status)
		got := string(readFile(t, path))
		requireTidyTOML(t, got)
		assert.Contains(t, got, `DATABRICKS_CONFIG_FILE = "two"`)
		assert.NotContains(t, got, `"one"`)
		assert.True(t, strings.HasSuffix(got, "\n"+other), got)
	})

	t.Run("drop env", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		writeFile(t, path, []byte(blockFor("test")+"\n"+envTable+"\n"+other), 0o600)
		a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

		_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("other"), nil, false)
		require.NoError(t, err)
		assert.Equal(t, StatusUpdated, status)
		got := string(readFile(t, path))
		requireTidyTOML(t, got)
		assert.NotContains(t, got, "DATABRICKS_CONFIG_FILE")
		assert.Equal(t, blockFor("other")+"\n"+other, got)
	})

	t.Run("remove server with env", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		writeFile(t, path, []byte(blockFor("test")+"\n"+envTable+"\n"+other), 0o600)
		a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

		removed, err := Remove(t.Context(), a, testName)
		require.NoError(t, err)
		assert.True(t, removed)
		assert.Equal(t, other, string(readFile(t, path)))
	})

	t.Run("remove first server", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		writeFile(t, path, []byte(blockFor("test")+"\n"+other), 0o600)
		a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

		removed, err := Remove(t.Context(), a, testName)
		require.NoError(t, err)
		assert.True(t, removed)
		assert.Equal(t, other, string(readFile(t, path)))
	})
}

// TestTOMLQuotedDottedNameIsAnotherServer checks that a table name in quotes can
// contain dots, and a server named like its prefix is a different server. Neither
// is edited when the other one is.
func TestTOMLQuotedDottedNameIsAnotherServer(t *testing.T) {
	before := "[mcp_servers.\"genie.x\"]\ncommand = \"npx\"\n"
	path := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, path, []byte(before), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	removed, err := Remove(t.Context(), a, testName)
	require.NoError(t, err)
	assert.False(t, removed)
	assert.Equal(t, before, string(readFile(t, path)))

	_, status, err := Upsert(t.Context(), a, "genie", testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	assert.Equal(t, before+"\n"+strings.Replace(testBlock, testName, "genie", 1), string(readFile(t, path)))
}

// TestTOMLHeaderLikeLineInMultilineString checks that a line that looks like a
// table header inside a multi-line string is text, not a table, so it is not
// edited.
func TestTOMLHeaderLikeLineInMultilineString(t *testing.T) {
	notes := "[mcp_servers.notes]\ntext = \"\"\"\n[mcp_servers.genie]\ncommand = \"not a header\"\n\"\"\"\n"
	path := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, path, []byte(notes), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, status, err := Upsert(t.Context(), a, "genie", testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	assert.Equal(t, notes+"\n"+strings.Replace(testBlock, testName, "genie", 1), string(readFile(t, path)))

	// Removing our server leaves the string as it was.
	removed, err := Remove(t.Context(), a, "genie")
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Equal(t, notes, string(readFile(t, path)))
}

// TestTOMLRemoveKeepsOneSeparator checks that removing a table between two others
// keeps one blank line between them when the table had a blank line on either side
// of it.
func TestTOMLRemoveKeepsOneSeparator(t *testing.T) {
	other := "[mcp_servers.other]\ncommand = \"npx\"\n"
	for _, tc := range []struct {
		name   string
		before string
	}{
		{"blank before", "[mcp_servers.first]\ncommand = \"a\"\n\n" + blockFor("test") + other},
		{"blank after", "[mcp_servers.first]\ncommand = \"a\"\n" + blockFor("test") + "\n" + other},
		{"blank both sides", "[mcp_servers.first]\ncommand = \"a\"\n\n" + blockFor("test") + "\n" + other},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			writeFile(t, path, []byte(tc.before), 0o600)
			a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

			removed, err := Remove(t.Context(), a, testName)
			require.NoError(t, err)
			assert.True(t, removed)
			got := string(readFile(t, path))
			requireTidyTOML(t, got)
			assert.Equal(t, "[mcp_servers.first]\ncommand = \"a\"\n\n"+other, got)
		})
	}
}

// TestTOMLEditWithNonFiniteFloatElsewhere checks that inf and nan in a table the
// edit does not touch do not stop the edit, and that the file comes back as it was.
func TestTOMLEditWithNonFiniteFloatElsewhere(t *testing.T) {
	before := "[limits]\nmax = inf\nmin = -inf\nratio = nan\n\n[mcp_servers.other]\ncommand = \"npx\"\n"
	path := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, path, []byte(before), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, status, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	_, status, err = Upsert(t.Context(), a, testName, testCmd, testArgs("other"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	removed, err := Remove(t.Context(), a, testName)
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Equal(t, before, string(readFile(t, path)))
}

// TestTOMLAddKeepsTrailingBlankLineOnce checks that a file ending in a blank line
// gets one blank line before the new server, and keeps its blank line at the end.
func TestTOMLAddKeepsTrailingBlankLineOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, path, []byte("model = \"x\"\n\n"), 0o600)
	a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

	_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, "model = \"x\"\n\n"+testBlock+"\n", string(readFile(t, path)))
}

// TestTOMLAddThenRemoveRestoresBytes checks that adding a server and removing it
// again gives back the file's own bytes, whatever blank lines the file ends with.
func TestTOMLAddThenRemoveRestoresBytes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		before string
	}{
		{"empty", ""},
		{"no trailing blank", "model = \"x\"\n"},
		{"one trailing blank", "model = \"x\"\n\n"},
		{"two trailing blanks", "model = \"x\"\n\n\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			writeFile(t, path, []byte(tc.before), 0o600)
			a := testAgent(path, mcpconfig.FormatTOML, "mcp_servers")

			_, _, err := Upsert(t.Context(), a, testName, testCmd, testArgs("test"), nil, false)
			require.NoError(t, err)
			removed, err := Remove(t.Context(), a, testName)
			require.NoError(t, err)
			assert.True(t, removed)
			assert.Equal(t, tc.before, string(readFile(t, path)))
		})
	}
}
