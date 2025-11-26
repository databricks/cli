package mcp

import (
	"github.com/spf13/cobra"
)

func newToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "tools",
		Short:  "MCP tools for AI agents",
		Hidden: true,
	}

	cmd.AddCommand(newQueryCmd())
	cmd.AddCommand(newDiscoverSchemaCmd())
	cmd.AddCommand(newInitTemplateCmd())

	return cmd
}
