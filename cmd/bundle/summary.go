package bundle

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/spf13/cobra"
)

func newSummaryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Summarize resources deployed by this bundle",
		Args:  root.NoArgs,
	}

	var forcePull bool
	var includeLocations bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")
	cmd.Flags().BoolVar(&includeLocations, "include-locations", false, "Include location information in the output")
	cmd.Flags().MarkHidden("include-locations")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b, diags := prepareBundleForSummary(cmd, forcePull, includeLocations)
		return renderBundle(cmd, b, diags, true)
	}

	return cmd
}

func prepareBundleForSummary(cmd *cobra.Command, forcePull, includeLocations bool) (*bundle.Bundle, diag.Diagnostics) {
	ctx := cmd.Context()
	b, diags := utils.ConfigureBundleWithVariables(cmd)
	if err := diags.Error(); err != nil {
		return nil, diags
	}

	diags = diags.Extend(phases.Initialize(ctx, b))
	if err := diags.Error(); err != nil {
		return nil, diags
	}

	cacheDir, err := terraform.Dir(ctx, b)
	if err != nil {
		return nil, diags
	}
	_, stateFileErr := os.Stat(filepath.Join(cacheDir, b.StateFilename()))
	_, configFileErr := os.Stat(filepath.Join(cacheDir, terraform.TerraformConfigFileName))
	noCache := errors.Is(stateFileErr, os.ErrNotExist) || errors.Is(configFileErr, os.ErrNotExist)

	if forcePull || noCache {
		diags = diags.Extend(bundle.Apply(ctx, b, statemgmt.StatePull()))
		if err := diags.Error(); err != nil {
			return nil, diags
		}

		if !b.DirectDeployment {
			diags = diags.Extend(bundle.ApplySeq(ctx, b,
				terraform.Interpolate(),
				terraform.Write(),
			))
			if err := diags.Error(); err != nil {
				return nil, diags
			}
		}
	}

	diags = diags.Extend(bundle.ApplySeq(ctx, b,
		statemgmt.Load(),
		mutator.InitializeURLs(),
	))

	// Include location information in the output if the flag is set.
	if includeLocations {
		diags = diags.Extend(bundle.Apply(ctx, b, mutator.PopulateLocations()))
	}

	if err := diags.Error(); err != nil {
		return nil, diags
	}

	return b, diags
}
