package bundle

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/cmd/root"
	"github.com/spf13/cobra"
)

func ConfigureBundleWithVariables(cmd *cobra.Command, args []string) error {
	// Load bundle config and apply environment
	err := root.MustConfigureBundle(cmd, args)
	if err != nil {
		return nil
	}

	// Initial variable values command line args
	b := bundle.Get(cmd.Context())
	return b.Config.InitializeVariables(variables)
}
