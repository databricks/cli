package pipelines

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/bundle/statemgmt"
	bundleutils "github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

// newRunCommand is copied from cmd/bundle/run.go and adapted for pipelines use.
func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [flags] [KEY]",
		Short: "Run a pipeline",
		Long: `Run the pipeline identified by KEY.
The KEY is the unique identifier of the pipeline to run.`,
	}

	var runOptions run.Options

	// Only define pipeline flags, skip job flags
	pipelineGroup := cmdgroup.NewFlagGroup("Pipeline Run")
	runOptions.Pipeline.Define(pipelineGroup.FlagSet())
	pipelineGroup.FlagSet().MarkHidden("validate-only")

	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(pipelineGroup)

	var noWait bool
	var restart bool
	cmd.Flags().BoolVar(&noWait, "no-wait", false, "Don't wait for the run to complete.")
	cmd.Flags().BoolVar(&restart, "restart", false, "Restart the run if it is already running.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := bundleutils.ConfigureBundleWithVariables(cmd)
		if diags.HasError() {
			return RenderDiagnostics(cmd.OutOrStdout(), b, diags)
		}

		diags = diags.Extend(phases.Initialize(ctx, b))
		if diags.HasError() {
			return RenderDiagnostics(cmd.OutOrStdout(), b, diags)
		}

		key, args, err := ResolveRunArgument(ctx, b, args)
		if err != nil {
			return err
		}

		diags = diags.Extend(bundle.ApplySeq(ctx, b,
			terraform.Interpolate(),
			terraform.Write(),
			statemgmt.StatePull(),
			terraform.Load(terraform.ErrorOnEmptyState),
		))
		if diags.HasError() {
			return RenderDiagnostics(cmd.OutOrStdout(), b, diags)
		}

		runner, err := KeyToRunner(b, key)
		if err != nil {
			return err
		}

		runOptions.NoWait = noWait

		// Parse additional positional arguments.
		err = runner.ParseArgs(args, &runOptions)
		if err != nil {
			return err
		}

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
			completions := resources.Completions(b, run.IsRunnable)
			return maps.Keys(completions), cobra.ShellCompDirectiveNoFileComp
		} else {
			// If we know the resource to run, we can complete additional positional arguments.
			runner, err := KeyToRunner(b, args[0])
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return runner.CompleteArgs(args[1:], toComplete)
		}
	}

	return cmd
}
