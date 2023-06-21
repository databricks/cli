package version

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:  "version",
	Args: cobra.NoArgs,

	Annotations: map[string]string{
		"template": "Databricks CLI v{{.Version}}\n",
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdio.Render(cmd.Context(), build.GetInfo())
	},
}

func init() {
	root.RootCmd.AddCommand(versionCmd)
}
