package aitools

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Show instructions for uninstalling the MCP server",
		Long:  `Show instructions for uninstalling the Databricks CLI MCP server from coding agents.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Currently we just provide instructions for users to uninstall manually via their agent.
			fmt.Print(`
To uninstall the Databricks CLI MCP server, please ask your coding agent to remove it.

For Claude Code, you can also use:
  claude mcp remove databricks-aitools

For Cursor, you can also manually remove the entry from:
  ~/.cursor/mcp.json
`)
		},
	}
}
