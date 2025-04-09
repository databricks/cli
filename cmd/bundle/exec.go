package bundle

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/exec"
	"github.com/spf13/cobra"
)

func newExecCommand() *cobra.Command {
	execCmd := &cobra.Command{
		Use:   "exec",
		Short: "Execute a command using the same authentication context as the bundle",
		Args:  cobra.MinimumNArgs(1),
		Long: `Execute a command using the same authentication context as the bundle

Note: The current working directory of the provided command will be set to the root
of the bundle.

Authentication to the input command will be provided by setting the appropriate
environment variables that Databricks tools use to authenticate.

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

			env := auth.ProcessEnv(cmdctx.ConfigUsed(cmd.Context()))

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

			// TODO: Test singal propogation in the acceptance testS?

			err := exec.Execv(exec.ExecvOptions{
				Args:   args,
				Env:    env,
				Dir:    b.BundleRootPath,
				Stderr: cmd.ErrOrStderr(),
				Stdout: cmd.OutOrStdout(),
			})
			if err != nil {
				return fmt.Errorf("running %q failed: %w", strings.Join(args, " "), err)
			}

			return nil
		},
	}

	return execCmd
}
