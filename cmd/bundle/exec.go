package bundle

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/command"
	"github.com/spf13/cobra"
)

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

			env := auth.ProcessEnv(command.ConfigUsed(cmd.Context()))

			// If user has specified a target, pass it to the child command.
			//
			// This is only useful for when the Databricks CLI is the child command.
			if target := root.GetTarget(cmd); target != "" {
				env = append(env, "DATABRICKS_CONFIG_TARGET="+target)
			}

			// If the bundle has a profile configured, explicitly pass it to the child command.
			//
			// This is only useful for when the Databricks CLI is the child command,
			// since if we do not explicitly pass the profile, the CLI will use the
			// auth configured in the bundle YAML configuration (if any).
			if b.Config.Workspace.Profile != "" {
				env = append(env, "DATABRICKS_CONFIG_PROFILE="+b.Config.Workspace.Profile)
			}

			childCmd.Env = env

			// Execute all scripts from the bundle root directory. This behavior can
			// be surprising in isolation, but we do it to keep the behavior consistent
			// for both these cases:
			// 1. One shot commands like `databricks bundle exec -- echo hello`
			// 2. (upcoming) Scripts that are defined in the scripts section of the DAB.
			//
			// TODO(shreyas): Add a DATABRICKS_BUNDLE_INITIAL_CWD environment variable
			// that users can read to figure out the original CWD. I'll do that when
			// adding support for the scripts section.
			childCmd.Dir = b.BundleRootPath

			stdout, err := childCmd.StdoutPipe()
			if err != nil {
				return fmt.Errorf("creating stdout pipe failed: %w", err)
			}

			stderr, err := childCmd.StderrPipe()
			if err != nil {
				return fmt.Errorf("creating stderr pipe failed: %w", err)
			}

			// Start the child command.
			err = childCmd.Start()
			if err != nil {
				return fmt.Errorf("starting %q failed: %w", strings.Join(args, " "), err)
			}

			var wg sync.WaitGroup
			wg.Add(2)

			var stdoutErr error
			go func() {
				defer wg.Done()

				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					_, err = fmt.Fprintln(cmd.OutOrStdout(), scanner.Text())
					if err != nil {
						stdoutErr = err
						break
					}
				}
			}()

			var stderrErr error
			go func() {
				defer wg.Done()

				scanner := bufio.NewScanner(stderr)
				for scanner.Scan() {
					_, err = fmt.Fprintln(cmd.ErrOrStderr(), scanner.Text())
					if err != nil {
						stderrErr = err
						break
					}
				}
			}()

			wg.Wait()

			if stdoutErr != nil {
				return fmt.Errorf("writing stdout failed: %w", stdoutErr)
			}

			if stderrErr != nil {
				return fmt.Errorf("writing stderr failed: %w", stderrErr)
			}

			err = childCmd.Wait()
			if exitErr, ok := err.(*exec.ExitError); ok {
				// We don't make the parent CLI process exit with the same exit code
				// as the child process because the exit codes for the CLI have not
				// been standardized yet.
				//
				// This keeps the door open for us to associate specific exit codes
				// with specific classes of errors in the future.
				return fmt.Errorf("running %q failed with exit code: %d", strings.Join(args, " "), exitErr.ExitCode())
			}
			if err != nil {
				return fmt.Errorf("running %q failed: %w", strings.Join(args, " "), err)
			}

			return nil
		},
	}

	return execCmd
}
