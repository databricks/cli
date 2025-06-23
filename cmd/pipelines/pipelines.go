package pipelines

import (
	bundlecmd "github.com/databricks/cli/cmd/bundle"
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

	bundlecmd.InitVariableFlag(cmd)
	cmd.AddCommand(Deploy())

	return cmd
}
