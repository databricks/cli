package agentconfig

import (
	"strings"

	"github.com/databricks/cli/libs/aitools/mcpconfig"
)

// IsUGProxy reports whether entry runs the `ug mcp-proxy` bridge: a ug binary whose first argument is mcp-proxy.
func IsUGProxy(entry map[string]any) bool {
	argv, ok := mcpconfig.Argv(entry)
	return ok && len(argv) > 1 && mcpconfig.IsBinary(argv[0], "ug", "ug.exe") && argv[1] == "mcp-proxy"
}

// IsUnmanaged reports whether entry connects to Databricks managed MCP servers
// without having been written by this CLI: the `ug mcp-proxy` bridge, or a
// direct /ai-gateway/mcp-services/ URL. Such entries are listed by `list`, and
// Upsert replaces one only with force.
func IsUnmanaged(entry map[string]any) bool {
	if IsUGProxy(entry) {
		return true
	}
	// Remote entries name their URL differently per agent: url (Cursor, Claude
	// Code), httpUrl (Gemini streamable HTTP), serverUrl (Antigravity).
	for _, key := range []string{"url", "httpUrl", "serverUrl"} {
		if url, _ := entry[key].(string); strings.Contains(url, "/ai-gateway/mcp-services/") {
			return true
		}
	}
	return false
}
