package mcp

import (
	"context"
	"errors"
)

func runUninstall(ctx context.Context) error {
	return errors.New("uninstall is not implemented\n\nTo uninstall the Databricks CLI MCP server, please ask your coding agent to remove it.\nFor Claude Code, use: claude mcp remove databricks-cli\nFor Cursor, manually remove the entry from ~/.cursor/mcp.json")
}
