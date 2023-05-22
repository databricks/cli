package version

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/spf13/cobra"
)

var detail = false

var versionCmd = &cobra.Command{
	Use:  "version",
	Args: cobra.NoArgs,

	RunE: func(cmd *cobra.Command, args []string) error {
		info := build.GetInfo()
		if detail {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(info)
		}

		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Databricks CLI v%s\n", info.Version)
		return err
	},
}

func init() {
	versionCmd.Flags().BoolVar(&detail, "detail", false, "output detailed version information as JSON")
	root.RootCmd.AddCommand(versionCmd)
}
