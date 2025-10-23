package bundle

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/render"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

func newSummaryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Summarize resources deployed by this bundle",
		Long: `Summarize resources deployed by this bundle with their workspace URLs.
Useful after deployment to see what was created and where to find it.`,
		Args: root.NoArgs,
	}

	var forcePull bool
	var includeLocations bool
	var shouldShowFullConfig bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")
	cmd.Flags().BoolVar(&includeLocations, "include-locations", false, "Include location information in the output")
	cmd.Flags().MarkHidden("include-locations")
	cmd.Flags().BoolVar(&shouldShowFullConfig, "show-full-config", false, "Load and output the full bundle config")
	cmd.Flags().MarkHidden("show-full-config")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var err error
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)
		logdiag.SetSeverity(ctx, diag.Warning)

		if shouldShowFullConfig {
			err = showFullConfig(ctx, cmd)
			if err != nil {
				return err
			}
		} else {
			err = showSummary(ctx, cmd, forcePull, includeLocations)
			if err != nil {
				return err
			}
		}

		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		return nil
	}

	return cmd
}

func showFullConfig(ctx context.Context, cmd *cobra.Command) error {
	// call `MustLoad` directly instead of `MustConfigureBundle` because the latter does
	// validation that we're not interested in here
	b := bundle.MustLoad(ctx)
	if b == nil || logdiag.HasError(ctx) {
		return nil
	}

	mutator.DefaultMutators(ctx, b)
	if logdiag.HasError(ctx) {
		return nil
	}

	err := renderJsonOutput(cmd, b)
	if err != nil {
		return err
	}

	return nil
}

func showSummary(ctx context.Context, cmd *cobra.Command, forcePull, includeLocations bool) error {
	var err error
	b := prepareBundleForSummary(cmd, forcePull, includeLocations)

	if b != nil {
		if root.OutputType(cmd) == flags.OutputText {
			err = render.RenderSummary(ctx, cmd.OutOrStdout(), b)
			if err != nil {
				return err
			}
		}
		if root.OutputType(cmd) == flags.OutputJSON {
			err = renderJsonOutput(cmd, b)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func prepareBundleForSummary(cmd *cobra.Command, forcePull, includeLocations bool) *bundle.Bundle {
	b := utils.ConfigureBundleWithVariables(cmd)
	ctx := cmd.Context()
	if b == nil || logdiag.HasError(ctx) {
		return nil
	}
	ctx = cmd.Context()

	phases.Initialize(ctx, b)
	if logdiag.HasError(ctx) {
		return nil
	}

	//cacheDir, err := terraform.Dir(ctx, b)
	//if err != nil {
	//	logdiag.LogError(ctx, err)
	//	return nil
	//}
	//_, stateFileErr := os.Stat(filepath.Join(cacheDir, b.StateFilename()))
	//_, configFileErr := os.Stat(filepath.Join(cacheDir, terraform.TerraformConfigFileName))
	//noCache := errors.Is(stateFileErr, os.ErrNotExist) || errors.Is(configFileErr, os.ErrNotExist)
	noCache := true

	if forcePull || noCache {
		ctx = statemgmt.PullResourcesState(ctx, b)

		if logdiag.HasError(ctx) {
			return nil
		}
	}

	bundle.ApplySeqContext(ctx, b,
		statemgmt.Load(),
		mutator.InitializeURLs(),
	)

	// Include location information in the output if the flag is set.
	if includeLocations {
		bundle.ApplyContext(ctx, b, mutator.PopulateLocations())
	}

	if logdiag.HasError(ctx) {
		return nil
	}

	return b
}
