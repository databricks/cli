package agentconfig

import (
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
