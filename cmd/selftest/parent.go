package selftest

import (
	"fmt"
	"os"

	"github.com/databricks/cli/libs/daemon"
	"github.com/spf13/cobra"
)

const OutputFile = "DATABRICKS_CLI_SELFTEST_CHILD_OUTPUT_FILE"

func newParentCommand() *cobra.Command {
	return &cobra.Command{
		Use: "parent",
		RunE: func(cmd *cobra.Command, args []string) error {
			d := daemon.Daemon{
				Env:         os.Environ(),
				Args:        []string{"selftest", "child"},
				LogFile:     os.Getenv(OutputFile),
				PidFilePath: "child.pid",
			}

			err := d.Start()
			if err != nil {
				return fmt.Errorf("failed to start child process: %w", err)
			}
			fmt.Println("[parent] started child")

			err = d.WriteInput([]byte("Hello from the other side\n"))
			if err != nil {
				return fmt.Errorf("failed to write to child process: %w", err)
			}
			fmt.Println("[parent] input sent to child: Hello from the other side")

			err = d.Release()
			if err != nil {
				return fmt.Errorf("failed to release child process: %w", err)
			}
			fmt.Println("[parent] exiting")
			return nil
		},
	}
}
