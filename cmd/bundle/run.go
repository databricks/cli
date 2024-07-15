package bundle

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [flags] KEY",
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
`,
	}

	var runOptions run.Options
	runOptions.Define(cmd)

	var noWait bool
	var restart bool
	var logResults bool
	cmd.Flags().BoolVar(&noWait, "no-wait", false, "Don't wait for the run to complete.")
	cmd.Flags().BoolVar(&restart, "restart", false, "Restart the run if it is already running.")
	cmd.Flags().BoolVar(&logResults, "log-results", false, "Log notebook cell results.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)
		if err := diags.Error(); err != nil {
			return diags.Error()
		}

		diags = bundle.Apply(ctx, b, bundle.Seq(
			phases.Initialize(),
			terraform.Interpolate(),
			terraform.Write(),
			terraform.StatePull(),
			terraform.Load(terraform.ErrorOnEmptyState),
		))
		if err := diags.Error(); err != nil {
			return err
		}

		// If no arguments are specified, prompt the user to select something to run.
		if len(args) == 0 && cmdio.IsPromptSupported(ctx) {
			// Invert completions from KEY -> NAME, to NAME -> KEY.
			inv := make(map[string]string)
			for k, v := range run.ResourceCompletionMap(b) {
				inv[v] = k
			}
			id, err := cmdio.Select(ctx, inv, "Resource to run")
			if err != nil {
				return err
			}
			args = append(args, id)
		}

		if len(args) < 1 {
			return fmt.Errorf("expected a KEY of the resource to run")
		}

		runner, err := run.Find(b, args[0])
		if err != nil {
			return err
		}

		// Parse additional positional arguments.
		err = runner.ParseArgs(args[1:], &runOptions)
		if err != nil {
			return err
		}

		runOptions.NoWait = noWait
		runOptions.LogResults = logResults
		if restart {
			s := cmdio.Spinner(ctx)
			s <- "Cancelling all runs"
			err := runner.Cancel(ctx)
			close(s)
			if err != nil {
				return err
			}
		}
		output, err := runner.Run(ctx, &runOptions)
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
				cmd.OutOrStdout().Write([]byte(resultString))
			case flags.OutputJSON:
				b, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return err
				}
				cmd.OutOrStdout().Write(b)
			default:
				return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
			}
		}
		return nil
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		b, diags := root.MustConfigureBundle(cmd)
		if err := diags.Error(); err != nil {
			cobra.CompErrorln(err.Error())
			return nil, cobra.ShellCompDirectiveError
		}

		// No completion in the context of a bundle.
		// Source and destination paths are taken from bundle configuration.
		if b == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		if len(args) == 0 {
			return run.ResourceCompletions(b), cobra.ShellCompDirectiveNoFileComp
		} else {
			// If we know the resource to run, we can complete additional positional arguments.
			runner, err := run.Find(b, args[0])
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return runner.CompleteArgs(args[1:], toComplete)
		}
	}

	return cmd
}
