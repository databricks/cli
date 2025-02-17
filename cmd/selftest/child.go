package selftest

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/databricks/cli/libs/daemon"
	"github.com/databricks/cli/libs/process"
	"github.com/spf13/cobra"
)

// TODO: Manually test that indeed latency is not added.
func newChildCommand() *cobra.Command {
	return &cobra.Command{
		Use: "child",
		RunE: func(cmd *cobra.Command, args []string) error {
			parentPid, err := strconv.Atoi(os.Getenv(daemon.DatabricksCliParentPid))
			if err != nil {
				return fmt.Errorf("failed to parse parent PID: %w", err)
			}

			err = process.Wait(parentPid)
			if err != nil && !errors.As(err, &process.ErrProcessNotFound{}) {
				return fmt.Errorf("failed to wait for parent process: %w", err)
			}

			fmt.Println("\n====================")
			fmt.Println("\nAll output from this point on is from the child process")
			fmt.Println("Parent process has exited")

			in, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}

			fmt.Println("Received input from parent process:")
			fmt.Println(string(in))
			return nil
		},
	}
}
