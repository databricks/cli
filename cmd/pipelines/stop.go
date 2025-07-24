package pipelines

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

// resolveStopArgument resolves the pipeline key to stop
// If there is only one pipeline in the project, KEY is optional and the pipeline will be auto-selected.
// Otherwise, the user will be prompted to select a pipeline.
func resolveStopArgument(ctx context.Context, b *bundle.Bundle, args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}

	if key := autoSelectSinglePipeline(b); key != "" {
		return key, nil
	}

	if cmdio.IsPromptSupported(ctx) {
		return promptRunArgument(ctx, b)
	}

	return "", errors.New("expected a KEY of the pipeline to stop")
}

func stopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [KEY]",
		Short: "Stop a pipeline",
		Long: `Stop the pipeline if it's running, identified by KEY.
KEY is the unique name of the pipeline to stop, as defined in its YAML file.
If there is only one pipeline in the project, KEY is optional and the pipeline will be auto-selected.`,
		Args: root.MaximumNArgs(1),
	}

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

		key, err := resolveStopArgument(ctx, b, args)
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

		cmdio.LogString(ctx, fmt.Sprintf("Stopping %s...", key))
		err = runner.Cancel(ctx)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, key+" stopped successfully.")
		return nil
	}

	// TODO: This autocomplete functionality was copied from cmd/bundle/run.go and is not working properly.
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
			// If we know the resource to stop, we can complete additional positional arguments.
			runner, err := keyToRunner(b, args[0])
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return runner.CompleteArgs(args[1:], toComplete)
		}
	}

	return cmd
}
