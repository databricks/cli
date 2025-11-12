package mcp

import "github.com/spf13/cobra"

func NewMcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Model Context Protocol server for AI agents",
		Long: `Start and manage an MCP server that provides AI agents with tools to interact with Databricks.

The MCP server exposes the following capabilities:
- Databricks integration (query catalogs, schemas, tables, execute SQL)
- Project scaffolding (generate full-stack TypeScript applications)
- Workspace tools (file operations, bash, grep, glob)
- Sandboxed execution (isolated file/command execution)

The server communicates via stdio using the Model Context Protocol.`,
	}

	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newCheckCmd())

	return cmd
}
