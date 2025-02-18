package selftest

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/databricks/cli/libs/daemon"
	"github.com/spf13/cobra"
)

// TODO: Manually test that indeed latency is not added.
func newChildCommand() *cobra.Command {
	return &cobra.Command{
		Use: "child",
		RunE: func(cmd *cobra.Command, args []string) error {
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
