// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspaceconf

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/workspaceconf"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "workspace-conf",
	Short: `This API allows updating known workspace settings for advanced users.`,
	Long:  `This API allows updating known workspace settings for advanced users.`,
}

// start get-status command

var getStatusReq workspaceconf.GetStatus

func init() {
	Cmd.AddCommand(getStatusCmd)
	// TODO: short flags

}

var getStatusCmd = &cobra.Command{
	Use:   "get-status KEYS",
	Short: `Check configuration status.`,
	Long: `Check configuration status.
  
  Gets the configuration status for a workspace.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getStatusReq.Keys = args[0]

		response, err := w.WorkspaceConf.GetStatus(ctx, getStatusReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start set-status command

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

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)

		err = w.WorkspaceConf.SetStatus(ctx, setStatusReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service WorkspaceConf

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

}
