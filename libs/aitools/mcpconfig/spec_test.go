package mcpconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgv(t *testing.T) {
	for _, tc := range []struct {
		name  string
		entry map[string]any
		want  []string
		ok    bool
	}{
		{
			name:  "command and args",
			entry: map[string]any{"command": "/bin/x", "args": []any{"a", "b"}},
			want:  []string{"/bin/x", "a", "b"},
			ok:    true,
		},
		{
			name:  "command array",
			entry: map[string]any{"command": []any{"/bin/x", "a"}},
			want:  []string{"/bin/x", "a"},
			ok:    true,
		},
		{
			name:  "empty command array",
			entry: map[string]any{"command": []any{}},
			ok:    false,
		},
		{
			name:  "url entry has no command",
			entry: map[string]any{"url": "https://example.test/ai-gateway/mcp-services/x"},
			ok:    false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			argv, ok := Argv(tc.entry)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, argv)
		})
	}
}

func TestIsDatabricksEntry(t *testing.T) {
	serve := []any{"aitools", "mcp", "serve", "--profile", "p"}
	for _, tc := range []struct {
		name  string
		entry map[string]any
		want  bool
	}{
		{"unix path", map[string]any{"command": "/opt/homebrew/bin/databricks", "args": serve}, true},
		{"bare name", map[string]any{"command": "databricks", "args": serve}, true},
		{"windows path", map[string]any{"command": `C:\Program Files\databricks\databricks.exe`, "args": serve}, true},
		{"binary name in any case", map[string]any{"command": `C:\Tools\Databricks.EXE`, "args": serve}, true},
		{"opencode array", map[string]any{"command": []any{"/usr/bin/databricks", "aitools", "mcp", "serve"}}, true},
		{"other binary", map[string]any{"command": "/usr/bin/other", "args": serve}, false},
		{"other subcommand", map[string]any{"command": "databricks", "args": []any{"aitools", "mcp", "list"}}, false},
		{"suffix is not a basename", map[string]any{"command": "/x/mydatabricks", "args": serve}, false},
		{"no command", map[string]any{"url": "https://example.test"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsDatabricksEntry(tc.entry))
		})
	}
}

func TestValidServerName(t *testing.T) {
	assert.True(t, ValidServerName("genie_one-mcp1"))
	for _, name := range []string{"", "a.b", "a b", "a/b", "é", "genie\n"} {
		assert.False(t, ValidServerName(name), name)
	}
}
