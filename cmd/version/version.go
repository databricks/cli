package version

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/internal/build"
	"github.com/spf13/cobra"
)

var jsonOutput = false

var versionCmd = &cobra.Command{
	Use:  "version",
	Args: cobra.NoArgs,

	RunE: func(cmd *cobra.Command, args []string) error {
		info := build.GetInfo()
		if jsonOutput {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(info)
		}

		fmt.Fprintln(cmd.OutOrStdout(), info.Version)
		return nil
	},
}

func init() {
	versionCmd.Flags().BoolVar(&jsonOutput, "json", false, "output detailed version information as JSON")
	root.RootCmd.AddCommand(versionCmd)
}
