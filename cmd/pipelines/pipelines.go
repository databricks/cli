package pipelines

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipelines",
		Short: "Pipelines CLI",
		Long:  "Pipelines CLI (stub, to be filled in)",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	return cmd
}
