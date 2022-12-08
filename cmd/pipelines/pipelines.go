package pipelines

import (
	"github.com/databricks/bricks/cmd/pipelines/pipelines"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "pipelines",
	Short: `The Delta Live Tables API allows you to create, edit, delete, start, and view details about pipelines.`,
	Long: `The Delta Live Tables API allows you to create, edit, delete, start, and view
  details about pipelines.`,
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(pipelines.Cmd)
}
