package dev

import (
	"github.com/databricks/cli/experimental/dev/cmd/app"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Development tools for Databricks applications",
		Long: `Development tools for Databricks applications.

Provides commands for creating, developing, and deploying full-stack
Databricks applications.`,
	}

	cmd.AddCommand(app.New())

	return cmd
}
