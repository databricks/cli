package bundle

import (
	"errors"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newLaunchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "launch",
		Short: "Launches a notebook on development cluster",
		Long:  `Reads a file and executes it on dev cluster`,
		Args:  root.ExactArgs(1),

		// We're not ready to expose this command until we specify its semantics.
		Hidden: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return errors.New("TODO")
		// contents, err := os.ReadFile(args[0])
		// if err != nil {
		// 	return err
		// }
		// results := project.RunPythonOnDev(cmd.Context(), string(contents))
		// if results.Failed() {
		// 	return results.Err()
		// }
		// fmt.Fprintf(cmd.OutOrStdout(), "Success: %s", results.Text())
		// return nil
	}

	return cmd
}
