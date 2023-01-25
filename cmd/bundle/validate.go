package bundle

import (
	"encoding/json"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/phases"
	"github.com/databricks/bricks/cmd/root"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",

	PreRunE: root.MustConfigureBundle,
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
