package mcp

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Show instructions for uninstalling the MCP server",
		Long:  `Show instructions for uninstalling the Databricks CLI MCP server from coding agents.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(cmd.Context())
		},
	}

	return cmd
}

func runUninstall(ctx context.Context) error {
	return errors.New("uninstall is not implemented\n\nTo uninstall the Databricks CLI MCP server, please ask your coding agent to remove it.\nFor Claude Code, use: claude mcp remove databricks-mcp\nFor Cursor, manually remove the entry from ~/.cursor/mcp.json")
}
