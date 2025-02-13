package selftest

import (
	"os"

	"github.com/spf13/cobra"
)

const (
	PrintStdinParentPid = "DATABRICKS_CLI_PRINT_STDIN_PARENT_PID"
)

// TODO CONTINUE: Write command that wait for each other via the PID.
// Ensure to check the process name as the PID otherwise can be reused pretty
// quick.

func newPrintStdin() *cobra.Command {
	return &cobra.Command{
		Use: "print-stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Getenv(PrintStdinParentPid) != "" {
				return nil
			}
		},
	}
}
