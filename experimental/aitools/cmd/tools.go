package mcp

import (
	"github.com/databricks/cli/experimental/aitools/cmd/init_template"
	"github.com/spf13/cobra"
)

func newToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "tools",
		Short:  "CLI tools for AI agents",
		Long:   `CLI tools to be used by AI agents. These tools are optimized for AI coding agents like Claude Code and Cursor. The tools can change at any time. There are no stability guarantees for these tools.`,
		Hidden: false,
	}

	cmd.AddCommand(newQueryCmd())
	cmd.AddCommand(newDiscoverSchemaCmd())
	cmd.AddCommand(init_template.NewInitTemplateCommand())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newDeployCmd())
	cmd.AddCommand(newGetDefaultWarehouseCmd())

	return cmd
}
