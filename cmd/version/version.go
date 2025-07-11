// Copied to cmd/pipelines/version.go and adapted for pipelines use.
// Consider if changes made here should be made to the pipelines counterpart as well.
package version

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Args:  root.NoArgs,
		Short: "Retrieve information about the current version of this CLI",
		Annotations: map[string]string{
			"template": "Databricks CLI v{{.Version}}\n",
		},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return cmdio.Render(cmd.Context(), build.GetInfo())
	}

	return cmd
}
