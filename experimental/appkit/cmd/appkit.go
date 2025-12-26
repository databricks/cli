package appkit

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "appkit",
		Short:  "Manage Databricks AppKit applications",
		Hidden: true,
		Long: `Manage Databricks AppKit applications.

╔════════════════════════════════════════════════════════════════╗
║  ⚠️  EXPERIMENTAL: These commands may change in future versions ║
╚════════════════════════════════════════════════════════════════╝

AppKit provides a streamlined interface for creating, managing, and
monitoring full-stack Databricks applications built with TypeScript,
React, and Tailwind CSS.`,
	}

	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newListTemplatesCmd())
	cmd.AddCommand(newLogsCmd())
	cmd.AddCommand(newMetricsCmd())

	return cmd
}
