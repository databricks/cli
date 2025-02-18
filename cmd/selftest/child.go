package selftest

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/databricks/cli/libs/daemon"
	"github.com/spf13/cobra"
)

func newChildCommand() *cobra.Command {
	return &cobra.Command{
		Use: "child",
		RunE: func(cmd *cobra.Command, args []string) error {
			// wait_pid lives in acceptance/bin. We expect this command to only be called
			// from acceptance tests.
			//
			// Note: The golang stdlib only provides a way to wait on processes
			// that are children of the current process. While it's possible to
			// rely on os native syscalls to wait on arbitrary processes, it's hard
			// to get right and test. So I opted to just rely on the wait_pid
			// script here.
			waitCmd := exec.Command("bash", "-euo", "pipefail", "wait_pid", os.Getenv(daemon.DatabricksCliParentPid))
			b, err := waitCmd.Output()
			if err != nil {
				return fmt.Errorf("failed to wait for parent process: %w", err)
			}
			fmt.Print("[child]" + string(b))
			fmt.Println("[child] Parent process has exited")

			in, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}

			fmt.Println("[child] input from parent: " + string(in))
			return nil
		},
	}
}
