package bundle

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "run tests for the project",
		Long:  `This is longer description of the command`,

		// We're not ready to expose this command until we specify its semantics.
		Hidden: true,

		PreRunE: root.MustConfigureBundle,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("TODO")
		// results := project.RunPythonOnDev(cmd.Context(), `return 1`)
		// if results.Failed() {
		// 	return results.Err()
		// }
		// fmt.Fprintf(cmd.OutOrStdout(), "Success: %s", results.Text())
		// return nil
	}

	return cmd
}
