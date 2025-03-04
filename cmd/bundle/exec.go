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

type runErr struct {
	err  error
	args []string
}

func (e *runErr) Error() string {
	return fmt.Sprintf("Running %q failed: %s", strings.Join(e.args, " "), e.err)
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
			// profile configured in the bundle YAML configuration (if any).
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
				reader := bufio.NewReader(stdout)
				line, err := reader.ReadString('\n')
				for err == nil {
					_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", strings.TrimSpace(line))
					if err != nil {
						stdoutErr = err
						break
					}
					line, err = reader.ReadString('\n')
				}

				wg.Done()
			}()

			var stderrErr error
			go func() {
				reader := bufio.NewReader(stderr)
				// TODO CONTINUE: The formatting is messed u[] because of the new line business
				// here.
				// Fix that.
				line, err := reader.ReadString('\n')
				for err == nil {
					_, err = fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", strings.TrimSpace(line))
					if err != nil {
						stderrErr = err
						break
					}
					line, err = reader.ReadString('\n')
				}

				wg.Done()
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
				return &exitCodeErr{
					exitCode: exitErr.ExitCode(),
					args:     args,
				}
			}
			if err != nil {
				return fmt.Errorf("running %q failed: %w", strings.Join(args, " "), err)
			}

			return nil
		},
	}

	return execCmd
}
