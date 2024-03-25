package bundle

import (
	"encoding/json"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func newValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "validate",
		Short:   "Validate configuration",
		Args:    root.NoArgs,
		PreRunE: utils.ConfigureBundleWithVariables,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())

		diags := bundle.Apply(cmd.Context(), b, phases.Initialize())
		if err := diags.Error(); err != nil {
			return err
		}

		// Until we change up the output of this command to be a text representation,
		// we'll just output all diagnostics as debug logs.
		for _, diag := range b.Config.Diagnostics() {
			log.Debugf(cmd.Context(), "[%s]: %s", diag.Location, diag.Summary)
		}

		buf, err := json.MarshalIndent(b.Config, "", "  ")
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(buf)
		return nil
	}

	return cmd
}
