package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/render"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
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
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)
		if err := diags.Error(); err != nil {
			return diags.Error()
		}

		diags = phases.Initialize(ctx, b)
		if err := diags.Error(); err != nil {
			return err
		}

		cacheDir, err := terraform.Dir(ctx, b)
		if err != nil {
			return err
		}
		_, stateFileErr := os.Stat(filepath.Join(cacheDir, terraform.TerraformStateFileName))
		_, configFileErr := os.Stat(filepath.Join(cacheDir, terraform.TerraformConfigFileName))
		noCache := errors.Is(stateFileErr, os.ErrNotExist) || errors.Is(configFileErr, os.ErrNotExist)

		if forcePull || noCache {
			diags = bundle.ApplySeq(ctx, b,
				statemgmt.StatePull(),
				terraform.Interpolate(),
				terraform.Write(),
			)
			if err := diags.Error(); err != nil {
				return err
			}
		}

		diags = bundle.ApplySeq(ctx, b,
			terraform.Load(),
			mutator.InitializeURLs(),
		)

		// Include location information in the output if the flag is set.
		if includeLocations {
			diags = diags.Extend(bundle.Apply(ctx, b, mutator.PopulateLocations()))
		}

		if err := diags.Error(); err != nil {
			return err
		}

		switch root.OutputType(cmd) {
		case flags.OutputText:
			return render.RenderSummary(ctx, cmd.OutOrStdout(), b)
		case flags.OutputJSON:
			buf, err := json.MarshalIndent(b.Config.Value().AsAny(), "", "  ")
			if err != nil {
				return err
			}
			_, _ = cmd.OutOrStdout().Write(buf)
			_, _ = cmd.OutOrStdout().Write([]byte{'\n'})
		default:
			return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
		}

		return nil
	}

	return cmd
}
