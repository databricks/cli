package aircmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// New returns the root command for the experimental AI runtime CLI.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "air",
		Short: "Run and manage AI runtime training workloads",
		Long: `Run and manage AI runtime training workloads on Databricks serverless GPU compute.

This command set is the Go port of the standalone Python "air" CLI. It is
experimental and may change in future versions.`,
	}

	cmd.AddCommand(newRunCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newLogsCommand())
	cmd.AddCommand(newCancelCommand())
	cmd.AddCommand(newRegisterImageCommand())
	cmd.AddCommand(newConvertToDabsCommand())
	wrapRunErrorsWithDebugTip(cmd)

	return cmd
}

func wrapRunErrorsWithDebugTip(cmd *cobra.Command) {
	if cmd.RunE != nil {
		runE := cmd.RunE
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return withDebugErrorTip(cmd, runE(cmd, args))
		}
	}
	for _, child := range cmd.Commands() {
		wrapRunErrorsWithDebugTip(child)
	}
}

func withDebugErrorTip(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	debugFlag := cmd.Root().PersistentFlags().Lookup("debug")
	if debugFlag != nil && debugFlag.Value.String() == "true" {
		return err
	}

	commandPrefix := cmd.Root().CommandPath()
	command := commandPrefix + " --debug" + strings.TrimPrefix(cmd.CommandPath(), commandPrefix)
	return fmt.Errorf(
		"%w\n\nTip: use the --debug flag to see more details and a trace of this error:\n  %s …",
		err,
		command,
	)
}
