package commands

import (
	command_execution "github.com/databricks/bricks/cmd/commands/command-execution"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "commands",
	Short: `This API allows execution of Python, Scala, SQL, or R commands on running Databricks Clusters.`,
	Long: `This API allows execution of Python, Scala, SQL, or R commands on running
  Databricks Clusters.`,
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(command_execution.Cmd)
}
