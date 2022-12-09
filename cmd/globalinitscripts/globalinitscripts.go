package globalinitscripts

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/globalinitscripts"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "global-init-scripts",
	Short: `The Global Init Scripts API enables Workspace administrators to configure global initialization scripts for their workspace.`,
	Long: `The Global Init Scripts API enables Workspace administrators to configure
  global initialization scripts for their workspace. These scripts run on every
  node in every cluster in the workspace.
  
  **Important:** Existing clusters must be restarted to pick up any changes made
  to global init scripts. Global init scripts are run in order. If the init
  script returns with a bad exit code, the Apache Spark container fails to
  launch and init scripts with later position are skipped. If enough containers
  fail, the entire cluster fails with a GLOBAL_INIT_SCRIPT_FAILURE error code.`,
}

var createScriptReq globalinitscripts.GlobalInitScriptCreateRequest

func init() {
	Cmd.AddCommand(createScriptCmd)
	// TODO: short flags

	createScriptCmd.Flags().BoolVar(&createScriptReq.Enabled, "enabled", false, `Specifies whether the script is enabled.`)
	createScriptCmd.Flags().StringVar(&createScriptReq.Name, "name", "", `The name of the script.`)
	createScriptCmd.Flags().IntVar(&createScriptReq.Position, "position", 0, `The position of a global init script, where 0 represents the first script to run, 1 is the second script to run, in ascending order.`)
	createScriptCmd.Flags().StringVar(&createScriptReq.Script, "script", "", `The Base64-encoded content of the script.`)

}

var createScriptCmd = &cobra.Command{
	Use:   "create-script",
	Short: `Create init script.`,
	Long: `Create init script.
  
  Creates a new global init script in this workspace.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.GlobalInitScripts.CreateScript(ctx, createScriptReq)
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

var deleteScriptReq globalinitscripts.DeleteScript

func init() {
	Cmd.AddCommand(deleteScriptCmd)
	// TODO: short flags

	deleteScriptCmd.Flags().StringVar(&deleteScriptReq.ScriptId, "script-id", "", `The ID of the global init script.`)

}

var deleteScriptCmd = &cobra.Command{
	Use:   "delete-script",
	Short: `Delete init script.`,
	Long: `Delete init script.
  
  Deletes a global init script.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.GlobalInitScripts.DeleteScript(ctx, deleteScriptReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getScriptReq globalinitscripts.GetScript

func init() {
	Cmd.AddCommand(getScriptCmd)
	// TODO: short flags

	getScriptCmd.Flags().StringVar(&getScriptReq.ScriptId, "script-id", "", `The ID of the global init script.`)

}

var getScriptCmd = &cobra.Command{
	Use:   "get-script",
	Short: `Get an init script.`,
	Long: `Get an init script.
  
  Gets all the details of a script, including its Base64-encoded contents.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.GlobalInitScripts.GetScript(ctx, getScriptReq)
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

func init() {
	Cmd.AddCommand(listScriptsCmd)

}

var listScriptsCmd = &cobra.Command{
	Use:   "list-scripts",
	Short: `Get init scripts.`,
	Long: `Get init scripts.
  
  "Get a list of all global init scripts for this workspace. This returns all
  properties for each script but **not** the script contents. To retrieve the
  contents of a script, use the [get a global init
  script](#operation/get-script) operation.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.GlobalInitScripts.ListScriptsAll(ctx)
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

var updateScriptReq globalinitscripts.GlobalInitScriptUpdateRequest

func init() {
	Cmd.AddCommand(updateScriptCmd)
	// TODO: short flags

	updateScriptCmd.Flags().BoolVar(&updateScriptReq.Enabled, "enabled", false, `Specifies whether the script is enabled.`)
	updateScriptCmd.Flags().StringVar(&updateScriptReq.Name, "name", "", `The name of the script.`)
	updateScriptCmd.Flags().IntVar(&updateScriptReq.Position, "position", 0, `The position of a script, where 0 represents the first script to run, 1 is the second script to run, in ascending order.`)
	updateScriptCmd.Flags().StringVar(&updateScriptReq.Script, "script", "", `The Base64-encoded content of the script.`)
	updateScriptCmd.Flags().StringVar(&updateScriptReq.ScriptId, "script-id", "", `The ID of the global init script.`)

}

var updateScriptCmd = &cobra.Command{
	Use:   "update-script",
	Short: `Update init script.`,
	Long: `Update init script.
  
  Updates a global init script, specifying only the fields to change. All fields
  are optional. Unspecified fields retain their current value.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.GlobalInitScripts.UpdateScript(ctx, updateScriptReq)
		if err != nil {
			return err
		}

		return nil
	},
}

// end service GlobalInitScripts

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

}
