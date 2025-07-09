// Copied from cmd/version/version.go and adapted for pipelines use.
// Consider if changes made here should be made to the bundle counterpart as well.
package pipelines

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func versionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Args:  root.NoArgs,
		Short: "Retrieve information about the current version of the Pipelines CLI",
		Annotations: map[string]string{
			"template": "Pipelines CLI v{{.Version}} (based on Databricks CLI v{{.Version}})\n",
		},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return cmdio.Render(cmd.Context(), build.GetInfo())
	}

	return cmd
}
