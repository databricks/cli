package bundle

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func ConfigureBundleWithVariables(cmd *cobra.Command, args []string) error {
	// Load bundle config and apply environment
	err := root.MustConfigureBundle(cmd, args)
	if err != nil {
		return err
	}

	// Initialize variables by assigning them values passed as command line flags
	b := bundle.Get(cmd.Context())
	return b.Config.InitializeVariables(variables)
}

func AddVariableFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringSliceVar(&variables, "var", []string{}, `set values for variables defined in bundle config. Example: --var="foo=bar"`)
}
