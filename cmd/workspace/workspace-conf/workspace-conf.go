// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_conf

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "workspace-conf",
	Short: `This API allows updating known workspace settings for advanced users.`,
	Long:  `This API allows updating known workspace settings for advanced users.`,
	Annotations: map[string]string{
		"package": "settings",
	},
}

// start get-status command

var getStatusReq settings.GetStatusRequest
var getStatusJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getStatusCmd)
	// TODO: short flags
	getStatusCmd.Flags().Var(&getStatusJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getStatusCmd = &cobra.Command{
	Use:   "get-status KEYS",
	Short: `Check configuration status.`,
	Long: `Check configuration status.
  
  Gets the configuration status for a workspace.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getStatusJson.Unmarshal(&getStatusReq)
			if err != nil {
				return err
			}
		}
		getStatusReq.Keys = args[0]

		response, err := w.WorkspaceConf.GetStatus(ctx, getStatusReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start set-status command

var setStatusReq settings.WorkspaceConf
var setStatusJson flags.JsonFlag

func init() {
	Cmd.AddCommand(setStatusCmd)
	// TODO: short flags
	setStatusCmd.Flags().Var(&setStatusJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var setStatusCmd = &cobra.Command{
	Use:   "set-status",
	Short: `Enable/disable features.`,
	Long: `Enable/disable features.
  
  Sets the configuration status for a workspace, including enabling or disabling
  it.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = setStatusJson.Unmarshal(&setStatusReq)
			if err != nil {
				return err
			}
		} else {
		}

		err = w.WorkspaceConf.SetStatus(ctx, setStatusReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service WorkspaceConf
