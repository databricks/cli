package agentconfig

import (
	"strings"

	"github.com/databricks/cli/libs/aitools/mcpconfig"
)

// IsUnmanaged reports whether entry connects to Databricks managed MCP servers
// without having been written by this CLI: the `ug mcp-proxy` bridge, or a
// direct /ai-gateway/mcp-services/ URL. Such entries are listed and never
// modified.
func IsUnmanaged(entry map[string]any) bool {
	if argv, ok := mcpconfig.Argv(entry); ok && len(argv) > 1 &&
		mcpconfig.IsBinary(argv[0], "ug", "ug.exe") && argv[1] == "mcp-proxy" {
		return true
	}
	url, _ := entry["url"].(string)
	return strings.Contains(url, "/ai-gateway/mcp-services/")
}
