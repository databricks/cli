package pipelines

import (
	"context"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipelines",
		Short: "Pipelines CLI",
		Long:  "Pipelines CLI (stub, to be filled in)",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	root.SetupRootCommand(ctx, cmd)

	cmd.AddCommand(initCommand())
	return cmd
}
