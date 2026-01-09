package mcp

import (
	"github.com/databricks/cli/experimental/aitools/cmd/init_template"
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
	cmd.AddCommand(init_template.NewInitTemplateCommand())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newDeployCmd())

	return cmd
}
