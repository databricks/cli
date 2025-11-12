package apps

import (
	"github.com/databricks/cli/cmd/apps/mcp"
	"github.com/spf13/cobra"
)

func NewAppsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apps",
		Short:   "Databricks apps development tools",
		Long:    "Tools for developing and managing Databricks applications, including MCP servers for AI agents.",
		GroupID: "development",
	}

	cmd.AddCommand(mcp.NewMcpCmd())

	return cmd
}
