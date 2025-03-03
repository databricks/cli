package bundle

import (
	"fmt"
	"os/exec"
	"strings"

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
1. databricks bundle exec -- echo "hello, world"
2. databricks bundle exec -- /bin/bash -c "echo hello"
3. databricks bundle exec -- uv run pytest`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.ArgsLenAtDash() != 0 {
				return fmt.Errorf("Please add a '--' separator. Usage: 'databricks bundle exec -- %s'", strings.Join(args, " "))
			}

			// Load the bundle configuration to get the authentication credentials.
			b, diags := root.MustConfigureBundle(cmd)
			if diags.HasError() {
				return diags.Error()
			}

			childCmd := exec.Command(args[0], args[1:]...)

			env := auth.ProcessEnv(root.ConfigUsed(cmd.Context()))

			// If user has specified a target, pass it to the child command. DABs
			// defines a "default" target which is a placeholder for when no target is defined.
			// If that's the case, i.e. no targets are defined, then do not pass the target.
			//
			// This is only useful for when the Databricks CLI is the child command.
			if b.Config.Bundle.Target != mutator.DefaultTargetPlaceholder {
				env = append(env, "DATABRICKS_BUNDLE_TARGET="+b.Config.Bundle.Target)
			}

			// If the bundle has a profile configured, explicitly pass it to the child command.
			//
			// This is only useful for when the Databricks CLI is the child command,
			// since if we do not explicitly pass the profile, the CLI will use the
			// profile configured in the bundle YAML configuration (if any).We don't propagate the exit code as is because exit codes
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

			// Stream the stdout and stderr of the child process directly.
			childCmd.Stdout = cmd.OutOrStdout()
			childCmd.Stderr = cmd.ErrOrStderr()

			// Start the command
			if err := childCmd.Start(); err != nil {
				return fmt.Errorf("Error starting command: %s\n", err)
			}

			// Wait for the command to finish.
			err := childCmd.Wait()
			if exitErr, ok := err.(*exec.ExitError); ok {
				// We don't make the parent CLI process exit with the same exit code
				// as the child process because the exit codes for the CLI have not
				// been standardized yet.
				//
				// This keeps the door open for us to associate specific exit codes
				// with specific classes of errors in the future.
				return &exitCodeErr{
					exitCode: exitErr.ExitCode(),
					args:     args,
				}
			}
			if err != nil {
				return fmt.Errorf("Error waiting for command: %w", err)
			}

			return nil
		},
	}

	return execCmd
}
