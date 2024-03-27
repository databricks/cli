package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

func newSummaryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Describe the bundle resources and their deployment states",
		Args:  root.NoArgs,

		// This command is currently intended for the Databricks VSCode extension only
		Hidden: true,
	}

	var forcePull bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)
		if err := diags.Error(); err != nil {
			return diags.Error()
		}

		diags = bundle.Apply(ctx, b, phases.Initialize())
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
			diags = bundle.Apply(ctx, b, bundle.Seq(
				terraform.StatePull(),
				terraform.Interpolate(),
				terraform.Write(),
			))
			if err := diags.Error(); err != nil {
				return err
			}
		}

		diags = bundle.Apply(ctx, b, terraform.Load())
		if err := diags.Error(); err != nil {
			return err
		}

		switch root.OutputType(cmd) {
		case flags.OutputText:
			return fmt.Errorf("%w, only json output is supported", errors.ErrUnsupported)
		case flags.OutputJSON:
			buf, err := json.MarshalIndent(b.Config, "", "  ")
			if err != nil {
				return err
			}
			cmd.OutOrStdout().Write(buf)
		default:
			return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
		}

		return nil
	}

	return cmd
}
