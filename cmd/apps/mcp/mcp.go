package mcp

import "github.com/spf13/cobra"

func NewMcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Model Context Protocol server for AI agents",
		Long:  "Start and manage an MCP server that provides Databricks tools to AI agents via the Model Context Protocol.",
	}

	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newCheckCmd())

	return cmd
}
