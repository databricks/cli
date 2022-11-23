package debug

import (
	"fmt"

	"github.com/databricks/bricks/bundle"
	parent "github.com/databricks/bricks/cmd/bundle"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use: "whoami",

	PreRunE: parent.ConfigureBundle,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := bundle.Get(ctx).WorkspaceClient()
		user, err := w.CurrentUser.Me(ctx)
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), user.UserName)
		return nil
	},
}

func init() {
	debugCmd.AddCommand(whoamiCmd)
}
