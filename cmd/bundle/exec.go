package bundle

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/spf13/cobra"
)

type exitCodeErr struct {
	exitCode int
	args     []string
}

func (e *exitCodeErr) Error() string {
	return fmt.Sprintf("Running %q failed with exit code: %d", strings.Join(e.args, " "), e.exitCode)
}

func newExecCommand() *cobra.Command {
	execCmd := &cobra.Command{
		Use:   "exec",
		Short: "Execute a command using the same authentication context as the bundle",
		Args:  cobra.MinimumNArgs(1),
		Long: `Execute a command using the same authentication context as the bundle

The current working directory of the provided command will be set to the root
of the bundle.

Example usage:
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

			// If user has specified a target, pass it to the child command. If the
			// target is the default target, we don't need to pass it explicitly since
			// the CLI will use the default target by default.
			// This is only useful for when the Databricks CLI is the child command.
			if b.Config.Bundle.Target != mutator.DefaultTargetName {
				env = append(env, "DATABRICKS_BUNDLE_TARGET="+b.Config.Bundle.Target)
			}

			// If the bundle has a profile, explicitly pass it to the child command.
			// This is unnecessary for tools that follow the unified authentication spec.
			// However, because the CLI can read the profile from the bundle itself, we
			// need to pass it explicitly.
			// This is only useful for when the Databricks CLI is the child command.
			if b.Config.Workspace.Profile != "" {
				env = append(env, "DATABRICKS_CONFIG_PROFILE="+b.Config.Workspace.Profile)
			}

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

			// Create pipes to stream the stdout and stderr output from the child
			// process.
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

			// Stream both stdout and stderr to the user. We do this so that the user
			// does not have to wait for the command to finish before seeing the output.
			var wg sync.WaitGroup
			var stdoutErr, stderrErr error
			wg.Add(2)

			go func() {
				defer wg.Done()
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					_, err = cmd.OutOrStdout().Write([]byte(scanner.Text() + "\n"))
					if err != nil {
						stdoutErr = fmt.Errorf("Error writing to stdout: %w", err)
						return
					}
				}
			}()

			go func() {
				defer wg.Done()
				scanner := bufio.NewScanner(stderr)
				for scanner.Scan() {
					_, err := cmd.ErrOrStderr().Write([]byte(scanner.Text() + "\n"))
					if err != nil {
						stderrErr = fmt.Errorf("Error writing to stderr: %w", err)
						return
					}
				}
			}()

			// Wait for the command to finish.
			err = childCmd.Wait()
			if exitErr, ok := err.(*exec.ExitError); ok {
				// We don't propagate the exit code as is because exit codes for
				// the CLI have not been standardized yet. At some point in the
				// future we might want to associate specific exit codes with
				// specific classes of errors.
				return &exitCodeErr{
					exitCode: exitErr.ExitCode(),
					args:     args,
				}
			}
			if err != nil {
				return fmt.Errorf("Error waiting for command: %w", err)
			}

			// Wait for the goroutines to finish printing to stdout and stderr.
			wg.Wait()

			if stdoutErr != nil {
				return stdoutErr
			}

			if stderrErr != nil {
				return stderrErr
			}

			return nil
		},
	}

	return execCmd
}
