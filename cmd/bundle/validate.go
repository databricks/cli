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
		Use:   "validate",
		Short: "Validate configuration",
		Args:  root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)
		if err := diags.Error(); err != nil {
			return diags.Error()
		}

		diags = diags.Extend(bundle.Apply(ctx, b, phases.Initialize()))
		if err := diags.Error(); err != nil {
			return err
		}

		// Until we change up the output of this command to be a text representation,
		// we'll just output all diagnostics as debug logs.
		for _, diag := range diags {
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
