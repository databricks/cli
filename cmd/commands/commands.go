package commands

import (
	command_execution "github.com/databricks/bricks/cmd/commands/command-execution"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "commands",
}

func init() {

	Cmd.AddCommand(command_execution.Cmd)
}
