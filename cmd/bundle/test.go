package bundle

import (
	"errors"

	"github.com/spf13/cobra"
)

func newTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "run tests for the project",
		Long:  `This is longer description of the command`,

		// We're not ready to expose this command until we specify its semantics.
		Hidden: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return errors.New("TODO")
		// results := project.RunPythonOnDev(cmd.Context(), `return 1`)
		// if results.Failed() {
		// 	return results.Err()
		// }
		// fmt.Fprintf(cmd.OutOrStdout(), "Success: %s", results.Text())
		// return nil
	}

	return cmd
}
