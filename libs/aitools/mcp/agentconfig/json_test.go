package agentconfig

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tailscale/hujson"
)

// semanticJSONC decodes a file with comments and trailing commas.
func semanticJSONC(t *testing.T, data string) map[string]any {
	t.Helper()
	std, err := hujson.Standardize([]byte(data))
	require.NoError(t, err)
	return semantic(t, mcpconfig.FormatJSON, std)
}

// jsonEdit runs fn on a document parsed from data and returns the result bytes.
func jsonEdit(t *testing.T, data string, fn func(d document)) string {
	t.Helper()
	doc, err := parseJSON([]byte(data))
	require.NoError(t, err)
	fn(doc)
	out, err := doc.bytes()
	require.NoError(t, err)
	return string(out)
}

func TestJSONKeepsCommentsTrailingCommasAndValues(t *testing.T) {
	data := `{
  // Agent settings
  "theme": "dark", /* keep */
  "big": 12345678901234567890, "text": "café",
  "mcpServers": {
    "other": {"command": "npx"},
  },
}
`
	entry := testEntry(testCmd, testArgs("test"), nil)
	got := jsonEdit(t, data, func(d document) {
		require.NoError(t, d.set([]string{"mcpServers"}, testName, entry))
	})
	assert.Contains(t, got, "// Agent settings\n")
	assert.Contains(t, got, `"theme": "dark", /* keep */`)
	assert.Contains(t, got, `"big": 12345678901234567890, "text": "café",`)
	assert.Contains(t, got, "\"other\": {\"command\": \"npx\"},\n    \""+testName+"\": {\n")
	assert.Contains(t, got, "\n  },\n}\n")

	// Removing the entry leaves the other members and the comments in place.
	back := jsonEdit(t, got, func(d document) {
		require.NoError(t, d.remove([]string{"mcpServers"}, testName))
	})
	assert.Contains(t, back, "// Agent settings\n")
	assert.Equal(t, semanticJSONC(t, data), semanticJSONC(t, back))
}

func TestJSONCreatesMissingLevels(t *testing.T) {
	got := jsonEdit(t, `{"theme": "dark"}`, func(d document) {
		require.NoError(t, d.set([]string{"mcp", "servers"}, "x", map[string]any{"type": "local"}))
	})
	assert.Equal(t, map[string]any{
		"theme": "dark",
		"mcp":   map[string]any{"servers": map[string]any{"x": map[string]any{"type": "local"}}},
	}, semantic(t, mcpconfig.FormatJSON, []byte(got)))
}

func TestJSONBOMIsKept(t *testing.T) {
	got := jsonEdit(t, "\xEF\xBB\xBF{\"a\": 1}\n", func(d document) {
		require.NoError(t, d.set([]string{"mcpServers"}, testName, testEntry(testCmd, nil, nil)))
	})
	assert.Equal(t, "\xEF\xBB\xBF", got[:3])
	assert.Contains(t, got, `"a": 1`)
	assert.Contains(t, got, testName)
}

// A new file gets two-space indentation and a trailing newline, and <, > and &
// are written as they are.
func TestJSONNewFileLayout(t *testing.T) {
	entry := map[string]any{"command": "/a<b>&c", "args": []string{"x"}}
	got := jsonEdit(t, "", func(d document) {
		require.NoError(t, d.set([]string{"mcpServers"}, "x", entry))
	})
	assert.Equal(t, `{
  "mcpServers": {
    "x": {
      "args": [
        "x"
      ],
      "command": "/a<b>&c"
    }
  }
}
`, got)
}

func TestJSONRejectsNonObjectTopLevel(t *testing.T) {
	_, err := parseJSON([]byte(`"text"`))
	require.ErrorIs(t, err, ErrConfigInvalid)
}

func TestJSONEntriesSkipNonObjects(t *testing.T) {
	doc, err := parseJSON([]byte(`{"mcpServers": {"a": {"command": "x"}, "b": 3}}`))
	require.NoError(t, err)
	entries, err := doc.entries([]string{"mcpServers"})
	require.NoError(t, err)
	assert.Equal(t, map[string]map[string]any{"a": {"command": "x"}}, entries)
}

// requireStrictJSON fails unless data is strict JSON, which agents such as
// Claude Code read.
func requireStrictJSON(t *testing.T, data string) {
	t.Helper()
	var v any
	require.NoError(t, json.Unmarshal([]byte(data), &v), data)
}

// A comment after the last member is on that member's line. Adding a member
// after it keeps the comment there.
func TestJSONAddKeepsCommentOnPreviousMember(t *testing.T) {
	data := "{\n  \"mcpServers\": {\n    \"other\": {\"command\": \"npx\"} // other server\n  }\n}\n"
	got := jsonEdit(t, data, func(d document) {
		require.NoError(t, d.set([]string{"mcpServers"}, "x", map[string]any{"command": "x"}))
	})
	assert.Equal(t, "{\n  \"mcpServers\": {\n    \"other\": {\"command\": \"npx\"}, // other server\n    \"x\": {\n      \"command\": \"x\"\n    }\n  }\n}\n", got)
}

// An empty table gets its member on its own line, and its closing brace on a
// line of its own.
func TestJSONAddToEmptyTable(t *testing.T) {
	got := jsonEdit(t, "{\n  \"mcpServers\": {}\n}\n", func(d document) {
		require.NoError(t, d.set([]string{"mcpServers"}, "x", map[string]any{"command": "x"}))
	})
	assert.Equal(t, "{\n  \"mcpServers\": {\n    \"x\": {\n      \"command\": \"x\"\n    }\n  }\n}\n", got)
}

// A file with only comments and an empty object keeps its comments. Only the
// layout of the object changes.
func TestJSONAddKeepsCommentsOfEmptyObject(t *testing.T) {
	for _, tc := range []struct {
		data string
		want string
	}{
		{"// x\n{}", "// x\n{\n  \"mcpServers\": {\n    \"x\": {\n      \"command\": \"x\"\n    }\n  }\n}"},
		{"{ /* c */ }", "{\n  \"mcpServers\": {\n    \"x\": {\n      \"command\": \"x\"\n    }\n  } /* c */\n}"},
	} {
		t.Run(tc.data, func(t *testing.T) {
			got := jsonEdit(t, tc.data, func(d document) {
				require.NoError(t, d.set([]string{"mcpServers"}, "x", map[string]any{"command": "x"}))
			})
			assert.Equal(t, tc.want, got)
		})
	}
}

// An empty object file gets one trailing newline, as the file had.
func TestJSONAddToEmptyObjectFile(t *testing.T) {
	want := "{\n  \"mcpServers\": {\n    \"x\": {\n      \"command\": \"x\"\n    }\n  }\n}\n"
	for _, data := range []string{"{}\n", "", " \n"} {
		got := jsonEdit(t, data, func(d document) {
			require.NoError(t, d.set([]string{"mcpServers"}, "x", map[string]any{"command": "x"}))
		})
		assert.Equal(t, want, got, "input %q", data)
	}
}

// Removing the last member keeps a strict file strict. The comma that stood
// after the previous member's space is not left behind.
func TestJSONRemoveLastKeepsStrictJSON(t *testing.T) {
	for _, data := range []string{
		"{\n  \"mcpServers\": {\n    \"a\": {} ,\n    \"b\": {\"command\": \"x\"}\n  }\n}\n",
		"{\"mcpServers\": {\"a\": {} , \"b\": {\"command\": \"x\"}}}",
		"{\"mcpServers\": {\"a\": {}  , \"b\": {\"command\": \"x\"}}}",
	} {
		got := jsonEdit(t, data, func(d document) {
			require.NoError(t, d.remove([]string{"mcpServers"}, "b"))
		})
		requireStrictJSON(t, got)
		assert.Equal(t, map[string]any{"mcpServers": map[string]any{"a": map[string]any{}}}, semantic(t, mcpconfig.FormatJSON, []byte(got)))
	}
}

// A file with a trailing comma keeps the trailing comma when its last member is removed.
func TestJSONRemoveLastKeepsTrailingComma(t *testing.T) {
	got := jsonEdit(t, "{\"mcpServers\": {\"a\": {}, \"b\": {},}}", func(d document) {
		require.NoError(t, d.remove([]string{"mcpServers"}, "b"))
	})
	assert.Equal(t, "{\"mcpServers\": {\"a\": {},}}", got)
}

// A CRLF file gets CRLF in inserted text, so the file has one line ending.
func TestJSONKeepsCRLF(t *testing.T) {
	data := "{\r\n  \"mcpServers\": {\r\n    \"other\": {\"command\": \"npx\"}\r\n  }\r\n}\r\n"
	got := jsonEdit(t, data, func(d document) {
		require.NoError(t, d.set([]string{"mcpServers"}, "x", map[string]any{"command": "x", "args": []string{"a"}}))
	})
	assert.Equal(t, strings.Count(got, "\n"), strings.Count(got, "\r\n"), got)
	assert.Contains(t, got, "\"other\": {\"command\": \"npx\"},\r\n    \"x\": {\r\n")

	got = jsonEdit(t, got, func(d document) {
		require.NoError(t, d.remove([]string{"mcpServers"}, "x"))
	})
	assert.Equal(t, data, got)
}

// Adding a member to a strict file keeps it strict, whatever its layout.
func TestJSONAddKeepsStrictJSON(t *testing.T) {
	for _, data := range []string{
		"{}",
		"{\n}\n",
		`{"mcpServers": {}}`,
		"{\n  \"mcpServers\": {\n    \"a\": {} , \"b\": {\"command\": \"x\"}\n  }\n}\n",
		"{\"theme\": \"dark\", \"mcpServers\": {\"a\": {\"command\": \"x\"}}}",
	} {
		got := jsonEdit(t, data, func(d document) {
			require.NoError(t, d.set([]string{"mcpServers"}, "x", map[string]any{"command": "y"}))
		})
		requireStrictJSON(t, got)
		assert.Equal(t, "y", semantic(t, mcpconfig.FormatJSON, []byte(got))["mcpServers"].(map[string]any)["x"].(map[string]any)["command"], data)
	}
}
