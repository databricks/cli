package workspace_conf

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/workspaceconf"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "workspace-conf",
	Short: `This API allows updating known workspace settings for advanced users.`,
}

var getStatusReq workspaceconf.GetStatus

func init() {
	Cmd.AddCommand(getStatusCmd)
	// TODO: short flags

	getStatusCmd.Flags().StringVar(&getStatusReq.Keys, "keys", "", ``)

}

var getStatusCmd = &cobra.Command{
	Use:   "get-status",
	Short: `Check configuration status.`,
	Long: `Check configuration status.
  
  Gets the configuration status for a workspace.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.WorkspaceConf.GetStatus(ctx, getStatusReq)
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

var setStatusReq workspaceconf.WorkspaceConf

func init() {
	Cmd.AddCommand(setStatusCmd)
	// TODO: short flags

}

var setStatusCmd = &cobra.Command{
	Use:   "set-status",
	Short: `Enable/disable features.`,
	Long: `Enable/disable features.
  
  Sets the configuration status for a workspace, including enabling or disabling
  it.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.WorkspaceConf.SetStatus(ctx, setStatusReq)
		if err != nil {
			return err
		}

		return nil
	},
}
