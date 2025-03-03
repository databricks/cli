package bundle

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/spf13/cobra"
)

// TODO: Confirm that quoted strings are parsed as a single argument.
// TODO: test that -- works with flags as well.
// TODO CONTINUE: Making the bundle exec function work.
// TODO CONTINUE: Adding the scripts section to DABs.
// TODO: Ensure that these multi word strings work with the exec command. Example: echo "Hello, world!"
//       Or if it does not work, be sure why. Probably because string parsing is a part of the bash shell.

func newExecCommand() *cobra.Command {
	execCmd := &cobra.Command{
		Use:   "exec",
		Short: "Execute a command using the same authentication context as the bundle",
		Args:  cobra.MinimumNArgs(1),
		Long: `Examples:
1. databricks bundle exec -- echo "Hello, world!"
2. databricks bundle exec -- /bin/bash -c "echo 'Hello, world!'"
3. databricks bundle exec -- uv run pytest"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.ArgsLenAtDash() != 0 {
				return fmt.Errorf("Please add a '--' separator. Usage: 'databricks bundle exec -- %s'", strings.Join(args, " "))
			}

			// Load the bundle configuration to get the authentication credentials
			// set in the context.
			// TODO: What happens when no bundle is configured?
			_, diags := root.MustConfigureBundle(cmd)
			if diags.HasError() {
				return diags.Error()
			}

			childCmd := exec.Command(args[1], args[2:]...)
			childCmd.Env = auth.ProcessEnv(root.ConfigUsed(cmd.Context()))

			// Create pipes for stdout and stderr
			stdout, err := childCmd.StdoutPipe()
			if err != nil {
				return fmt.Errorf("Error creating stdout pipe: %w", err)
			}

			stderr, err := childCmd.StderrPipe()
			if err != nil {
				return fmt.Errorf("Error creating stderr pipe: %w", err)
			}

			// Start the command
			if err := childCmd.Start(); err != nil {
				return fmt.Errorf("Error starting command: %s\n", err)
			}

			// Stream both stdout and stderr to the user.
			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					fmt.Println(scanner.Text())
				}
			}()

			go func() {
				defer wg.Done()
				scanner := bufio.NewScanner(stderr)
				for scanner.Scan() {
					fmt.Println(scanner.Text())
				}
			}()

			// Wait for the command to finish.
			// TODO: Pretty exit codes?
			// TODO: Make CLI return the same exit codes?
			err = childCmd.Wait()
			if exitErr, ok := err.(*exec.ExitError); ok {
				return fmt.Errorf("Command exited with code: %d", exitErr.ExitCode())
			}

			if err := childCmd.Wait(); err != nil {
				return fmt.Errorf("Error waiting for command: %w", err)
			}

			// Wait for the goroutines to finish printing to stdout and stderr.
			wg.Wait()

			return nil
		},
	}

	// TODO: Is this needed to make -- work with flags?
	// execCmd.Flags().SetInterspersed(false)

	// TODO: func (c *Command) ArgsLenAtDash() int solves my problems here.

	return execCmd
}
