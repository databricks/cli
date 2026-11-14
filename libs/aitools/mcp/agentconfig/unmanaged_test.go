package agentconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsUnmanaged(t *testing.T) {
	for _, tc := range []struct {
		name  string
		entry map[string]any
		want  bool
	}{
		{
			name:  "ug mcp-proxy bridge",
			entry: map[string]any{"command": "/usr/local/bin/ug", "args": []any{"mcp-proxy", "--url", "x"}},
			want:  true,
		},
		{
			name:  "ug.exe bridge in command array",
			entry: map[string]any{"command": []any{`C:\tools\ug.exe`, "mcp-proxy"}},
			want:  true,
		},
		{
			name:  "ug without mcp-proxy",
			entry: map[string]any{"command": "/usr/local/bin/ug", "args": []any{"claude"}},
			want:  false,
		},
		{
			name:  "direct mcp services url",
			entry: map[string]any{"type": "http", "url": "https://example.test/ai-gateway/mcp-services/system.ai.dbsql"},
			want:  true,
		},
		{
			name:  "gemini httpUrl",
			entry: map[string]any{"httpUrl": "https://example.test/ai-gateway/mcp-services/system.ai.dbsql"},
			want:  true,
		},
		{
			name:  "antigravity serverUrl",
			entry: map[string]any{"serverUrl": "https://example.test/ai-gateway/mcp-services/system.ai.dbsql"},
			want:  true,
		},
		{
			name:  "unrelated url",
			entry: map[string]any{"type": "http", "url": "https://example.test/mcp"},
			want:  false,
		},
		{
			name:  "our own entry is not unmanaged",
			entry: map[string]any{"command": "/usr/bin/databricks", "args": []any{"aitools", "mcp", "serve"}},
			want:  false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsUnmanaged(tc.entry))
		})
	}
}
