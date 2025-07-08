package bundle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/exec"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

func promptRunArgument(ctx context.Context, b *bundle.Bundle) (string, error) {
	// Compute map of "Human readable name of resource" -> "resource key".
	inv := make(map[string]string)
	for k, ref := range resources.Completions(b, run.IsRunnable) {
		title := fmt.Sprintf("%s: %s", ref.Description.SingularTitle, ref.Resource.GetName())
		inv[title] = k
	}

	// Include scripts in the prompt options
	for k := range b.Config.Scripts {
		title := "Script: " + k
		inv[title] = k
	}

	key, err := cmdio.Select(ctx, inv, "Resource to run")
	if err != nil {
		return "", err
	}

	return key, nil
}

// resolveRunArgument resolves the resource key to run.
// It returns the remaining arguments to pass to the runner, if applicable.
func resolveRunArgument(ctx context.Context, b *bundle.Bundle, args []string) (string, []string, error) {
	// If no arguments are specified, prompt the user to select something to run.
	if len(args) == 0 && cmdio.IsPromptSupported(ctx) {
		key, err := promptRunArgument(ctx, b)
		if err != nil {
			return "", nil, err
		}
		return key, args, nil
	}

	if len(args) < 1 {
		return "", nil, errors.New("expected a KEY of the resource to run")
	}

	return args[0], args[1:], nil
}

func keyToRunner(b *bundle.Bundle, arg string) (run.Runner, error) {
	// Locate the resource to run.
	ref, err := resources.Lookup(b, arg, run.IsRunnable)
	if err != nil {
		return nil, err
	}

	// Convert the resource to a runnable resource.
	runner, err := run.ToRunner(b, ref)
	if err != nil {
		return nil, err
	}

	return runner, nil
}

func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [flags] [KEY]",
		Short: "Run a job or pipeline update",
		Long: `Run the job or pipeline identified by KEY.

The KEY is the unique identifier of the resource to run. In addition to
customizing the run using any of the available flags, you can also specify
keyword or positional arguments as shown in these examples:

   databricks bundle run my_job -- --key1 value1 --key2 value2

Or:

   databricks bundle run my_job -- value1 value2 value3

If the specified job uses job parameters or the job has a notebook task with
parameters, the first example applies and flag names are mapped to the
parameter names.

If the specified job does not use job parameters and the job has a Python file
task or a Python wheel task, the second example applies.

---------------------------------------------------------

You can also use the bundle run command to execute scripts / commands in the same
authentication context as the bundle.

Authentication to the input command will be provided by setting the appropriate
environment variables that Databricks tools use to authenticate.

Example usage:
1. databricks bundle run -- echo "hello, world"
2. databricks bundle run -- /bin/bash -c "echo hello"
3. databricks bundle run -- uv run pytest

---------------------------------------------------------
`,
	}

	var runOptions run.Options
	runOptions.Define(cmd)

	var noWait bool
	var restart bool
	cmd.Flags().BoolVar(&noWait, "no-wait", false, "Don't wait for the run to complete.")
	cmd.Flags().BoolVar(&restart, "restart", false, "Restart the run if it is already running.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := utils.ConfigureBundleWithVariables(cmd)
		if b == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		// If user runs the bundle run command as:
		// databricks bundle run -- <command> <args>
		// we execute the command inline.
		if cmd.ArgsLenAtDash() == 0 && len(args) > 0 {
			return executeInline(cmd, args, b)
		}

		phases.Initialize(ctx, b)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		key, args, err := resolveRunArgument(ctx, b, args)
		if err != nil {
			return err
		}

		if _, ok := b.Config.Scripts[key]; ok {
			if len(args) > 0 {
				return fmt.Errorf("additional arguments are not supported for scripts. Got: %v. We recommend using environment variables to pass runtime arguments to a script. For example: FOO=bar databricks bundle run my_script.", args)
			}

			content := b.Config.Scripts[key].Content
			return executeScript(content, cmd, b)
		}

		// Load resource IDs from terraform state.
		if !b.DirectDeployment {
			bundle.ApplySeqContext(ctx, b,
				terraform.Interpolate(),
				terraform.Write(),
			)
			if logdiag.HasError(ctx) {
				return root.ErrAlreadyPrinted
			}
		}

		bundle.ApplySeqContext(ctx, b,
			statemgmt.StatePull(),
			statemgmt.Load(statemgmt.ErrorOnEmptyState),
		)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		runner, err := keyToRunner(b, key)
		if err != nil {
			return err
		}

		// Parse additional positional arguments.
		err = runner.ParseArgs(args, &runOptions)
		if err != nil {
			return err
		}

		runOptions.NoWait = noWait
		var output output.RunOutput
		if restart {
			output, err = runner.Restart(ctx, &runOptions)
		} else {
			output, err = runner.Run(ctx, &runOptions)
		}
		if err != nil {
			return err
		}

		if output != nil {
			switch root.OutputType(cmd) {
			case flags.OutputText:
				resultString, err := output.String()
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write([]byte(resultString))
				if err != nil {
					return err
				}
			case flags.OutputJSON:
				b, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write(b)
				if err != nil {
					return err
				}
				_, _ = cmd.OutOrStdout().Write([]byte{'\n'})
			default:
				return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
			}
		}
		return nil
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		b := root.MustConfigureBundle(cmd)
		if logdiag.HasError(cmd.Context()) {
			return nil, cobra.ShellCompDirectiveError
		}

		// No completion in the context of a bundle.
		// Source and destination paths are taken from bundle configuration.
		if b == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		if len(args) == 0 {
			completions := resources.Completions(b, run.IsRunnable)
			return maps.Keys(completions), cobra.ShellCompDirectiveNoFileComp
		} else {
			// If we know the resource to run, we can complete additional positional arguments.
			runner, err := keyToRunner(b, args[0])
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return runner.CompleteArgs(args[1:], toComplete)
		}
	}

	return cmd
}

func scriptEnv(cmd *cobra.Command, b *bundle.Bundle) []string {
	out := auth.ProcessEnv(cmdctx.ConfigUsed(cmd.Context()))

	// If user has specified a target, pass it to the child command.
	//
	// This is only useful for when the Databricks CLI is the child command.
	if b.Config.Bundle.Target != "" {
		out = append(out, env.TargetVariable+"="+b.Config.Bundle.Target)
	}

	// If the bundle has a profile configured, explicitly pass it to the child command.
	//
	// This is only useful for when the Databricks CLI is the child command,
	// since if we do not explicitly pass the profile, the CLI will use the
	// auth configured in the bundle YAML configuration (if any).
	if b.Config.Workspace.Profile != "" {
		out = append(out, "DATABRICKS_CONFIG_PROFILE="+b.Config.Workspace.Profile)
	}

	return out
}

func executeScript(content string, cmd *cobra.Command, b *bundle.Bundle) error {
	return exec.ShellExecv(content, b.BundleRootPath, scriptEnv(cmd, b))
}

func executeInline(cmd *cobra.Command, args []string, b *bundle.Bundle) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	return exec.Execv(exec.ExecvOptions{
		Args: args,
		Env:  scriptEnv(cmd, b),
		Dir:  dir,
	})
}
