package app

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Manage Databricks applications",
		Long: `Manage Databricks applications.

Provides a streamlined interface for creating, managing, and monitoring
full-stack Databricks applications built with TypeScript, React, and
Tailwind CSS.`,
	}

	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newImportCmd())
	cmd.AddCommand(newDeployCmd())
	cmd.AddCommand(newDevRemoteCmd())

	return cmd
}
