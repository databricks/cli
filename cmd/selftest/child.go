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

// TODO: Look into the release function and ensure whether I need to call it.
const ()

// TODO CONTINUE: Write command that wait for each other via the PID.
// Ensure to check the process name as the PID otherwise can be reused pretty
// quick.
//
// Implement dummy child and parent commands, and write acceptance tests to account
// for all variations.
//
// Ensure that a robust timeout mechanism exists for the telemetry process. We
// do not want the daemons to hang indefinitely. Can this also be tested?
//
// TODO: One set of tests will be asserting that the tests have the right
// properties. A thread on my personal slack account will help with that.
// The other set of tests will assert on the functional behaviour, that the
// parent and child process are indeed indpenedent, and that the child process
// does not block the parent process.
//
// All this requires some PID handler which get the process information based on
// the PID and some "name", since PIDs can be reused, being a source of flakyness.
//
// TODO: Make sure to acknowledge the risk of failing when people try to delete
// the binary in windows.
//
// TODO: Ensure that child stdout / stderr are not sent to the parent process.

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
