package current_user

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "current-user",
	Short: `This API allows retrieving information about currently authenticated user or service principal.`,
	Long: `This API allows retrieving information about currently authenticated user or
  service principal.`,
}

func init() {
	Cmd.AddCommand(meCmd)

}

var meCmd = &cobra.Command{
	Use:   "me",
	Short: `Get current user info.`,
	Long: `Get current user info.
  
  Get details about the current method caller's identity.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.CurrentUser.Me(ctx)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

// end service CurrentUser
