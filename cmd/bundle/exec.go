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

// TODO: test that -- works with flags as well.

// TODO: Can bundle auth be resolved twice? What about:
// databricks bundle exec -t foo -- databricks jobs list -t bar?
// OR
// databricks bundle exec -- databricks jobs list -t bar?
// OR
// databricks bundle exec -- databricks jobs list?
// OR
// databricks bundle exec -t foo -- databricks jobs list?
//
// For the first two, undefined behavior is fine. For the latter two we need to ensure
// that the target from exec is respected.
//
// Add tests for all four of these cases.
// --> Do I need similar tests for --profile as well?
// --> Also add test for what happens with a default target?

// TODO: Add acceptance test that flags are indeed not parsed by the exec command and
// instead are parsed by the child command.

// # TODO: Table test casing the target permutations

func newExecCommand() *cobra.Command {
	execCmd := &cobra.Command{
		Use:   "exec",
		Short: "Execute a command using the same authentication context as the bundle",
		Args:  cobra.MinimumNArgs(1),
		// TODO: format once we have all the documentation here.
		Long: `
Note: This command executes scripts

Examples:
1. databricks bundle exec -- echo hello
2. databricks bundle exec -- /bin/bash -c "echo hello""
3. databricks bundle exec -- uv run pytest"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.ArgsLenAtDash() != 0 {
				return fmt.Errorf("Please add a '--' separator. Usage: 'databricks bundle exec -- %s'", strings.Join(args, " "))
			}

			// Load the bundle configuration to get the authentication credentials
			// set in the context.
			b, diags := root.MustConfigureBundle(cmd)
			if diags.HasError() {
				return diags.Error()
			}

			childCmd := exec.Command(args[0], args[1:]...)

			env := auth.ProcessEnv(root.ConfigUsed(cmd.Context()))
			// TODO: Test that this works correctly for all permutations.
			// TODO: Do the same for profile flag.
			// TODO: TODO: What happens here if a default target is resolved? When
			// no targets are defined?
			env = append(env, "DATABRICKS_BUNDLE_TARGET="+b.Config.Bundle.Target)
			childCmd.Env = env

			// Execute all scripts from the bundle root directory. This behavior can
			// be surprising in isolation, but we do it to keep the behavior consistent
			// for both cases:
			// 1. One shot commands like `databricks bundle exec -- echo hello`
			// 2. Scripts that are defined in the scripts section of the DAB.
			//
			// TODO(shreyas): Add a DATABRICKS_BUNDLE_INITIAL_CWD environment variable
			// that users can read to figure out the original CWD. I'll do that when
			// adding support for the scripts section.
			childCmd.Dir = b.BundleRootPath

			// Create pipes for stdout and stderr.
			// TODO: Test streaming of this? Is there a way?
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
			// TODO: Make CLI return the same exit codes? It has to, that's a requirement.
			err = childCmd.Wait()
			if exitErr, ok := err.(*exec.ExitError); ok {
				return fmt.Errorf("Command exited with code: %d", exitErr.ExitCode())
			}
			if err != nil {
				return fmt.Errorf("Error waiting for command: %w", err)
			}

			// Wait for the goroutines to finish printing to stdout and stderr.
			wg.Wait()

			return nil
		},
	}

	// TODO: Is this needed to make -- work with flags? What does this option do?
	// execCmd.Flags().SetInterspersed(false)

	return execCmd
}
