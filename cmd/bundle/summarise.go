package bundle

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/libs/cmdio"

	"github.com/databricks/cli/bundle/phases"
	"github.com/spf13/cobra"
)

func newSummariseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summarise",
		Short: "Describe the bundle resources and their deployment states",

		PreRunE: ConfigureBundleWithVariables,

		// This command is currently intended for the Databricks VSCode extension only
		Hidden: true,
	}

	var forcePull bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())

		err := bundle.Apply(cmd.Context(), b, phases.Initialize())
		if err != nil {
			return err
		}

		cacheDir, err := terraform.Dir(cmd.Context(), b)
		if err != nil {
			return err
		}
		_, err = os.Stat(filepath.Join(cacheDir, terraform.TerraformStateFileName))
		noCache := errors.Is(err, os.ErrNotExist)

		if forcePull || noCache {
			err = bundle.Apply(cmd.Context(), b, terraform.StatePull())
			if err != nil {
				return err
			}
		}

		err = bundle.Apply(cmd.Context(), b, terraform.Load())
		if err != nil {
			return err
		}

		buf, err := json.MarshalIndent(b.Config, "", "  ")
		if err != nil {
			return err
		}

		cmdio.LogString(cmd.Context(), string(buf))
		return nil
	}

	return cmd
}
