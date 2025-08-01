// Copied from cmd/bundle/run.go and adapted for pipelines use.
// Consider if changes made here should be made to the bundle counterpart as well.
package pipelines

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	bundleresources "github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	bundlerunoutput "github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/utils"
	pipelinetemplates "github.com/databricks/cli/cmd/pipelines/templates"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

func runCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [flags] [KEY]",
		Short: "Run a pipeline",
		Long: `Run the pipeline identified by KEY.
KEY is the unique name of the pipeline to run, as defined in its YAML file.
If there is only one pipeline in the project, KEY is optional and the pipeline will be auto-selected.
Refreshes all tables in the pipeline unless otherwise specified.`,
	}

	var refresh []string
	var fullRefreshAll bool
	var fullRefresh []string

	pipelineGroup := cmdgroup.NewFlagGroup("Pipeline Run")
	pipelineGroup.FlagSet().StringSliceVar(&refresh, "refresh", nil, "List of tables to run.")
	pipelineGroup.FlagSet().BoolVar(&fullRefreshAll, "full-refresh-all", false, "Perform a full graph reset and recompute.")
	pipelineGroup.FlagSet().StringSliceVar(&fullRefresh, "full-refresh", nil, "List of tables to reset and recompute.")

	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(pipelineGroup)

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

		phases.Initialize(ctx, b)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		key, _, err := resolveRunArgument(ctx, b, args)
		if err != nil {
			return err
		}

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

		runOptions := run.Options{
			Pipeline: run.PipelineOptions{
				Refresh:        refresh,
				FullRefreshAll: fullRefreshAll,
				FullRefresh:    fullRefresh,
			},
			NoWait: noWait,
		}

		var runOutput bundlerunoutput.RunOutput
		if restart {
			runOutput, err = runner.Restart(ctx, &runOptions)
		} else {
			runOutput, err = runner.Run(ctx, &runOptions)
		}
		if err != nil {
			return err
		}

		if runOutput != nil {
			switch root.OutputType(cmd) {
			case flags.OutputText:
				resultString, err := runOutput.String()
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write([]byte(resultString))
				if err != nil {
					return err
				}
			case flags.OutputJSON:
				b, err := json.MarshalIndent(runOutput, "", "  ")
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
		ref, err := bundleresources.Lookup(b, key, run.IsRunnable)
		if err != nil {
			return err
		}
		if ref.Description.SingularName == "pipeline" && runOutput != nil {
			if pipelineOutput, ok := runOutput.(*bundlerunoutput.PipelineOutput); ok && pipelineOutput.UpdateId != "" {
				err = pipelinetemplates.FetchAndDisplayPipelineUpdate(ctx, b, ref, pipelineOutput.UpdateId)
				if err != nil {
					return err
				}
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
			completions := bundleresources.Completions(b, run.IsRunnable)
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
