package bundle

import (
	"encoding/json"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",

	PreRunE: ConfigureBundleWithVariables,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())

		err := bundle.Apply(cmd.Context(), b, []bundle.Mutator{
			phases.Initialize(),
		})
		if err != nil {
			return err
		}

		buf, err := json.MarshalIndent(b.Config, "", "  ")
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(buf)
		return nil
	},
}

func init() {
	AddCommand(validateCmd)
}
