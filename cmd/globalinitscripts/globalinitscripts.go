// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

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

// start create command

var createReq globalinitscripts.GlobalInitScriptCreateRequest

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().BoolVar(&createReq.Enabled, "enabled", createReq.Enabled, `Specifies whether the script is enabled.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `The name of the script.`)
	createCmd.Flags().IntVar(&createReq.Position, "position", createReq.Position, `The position of a global init script, where 0 represents the first script to run, 1 is the second script to run, in ascending order.`)
	createCmd.Flags().StringVar(&createReq.Script, "script", createReq.Script, `The Base64-encoded content of the script.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create init script.`,
	Long: `Create init script.
  
  Creates a new global init script in this workspace.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.GlobalInitScripts.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq globalinitscripts.Delete

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.ScriptId, "script-id", deleteReq.ScriptId, `The ID of the global init script.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete init script.`,
	Long: `Delete init script.
  
  Deletes a global init script.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.GlobalInitScripts.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq globalinitscripts.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.ScriptId, "script-id", getReq.ScriptId, `The ID of the global init script.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get an init script.`,
	Long: `Get an init script.
  
  Gets all the details of a script, including its Base64-encoded contents.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.GlobalInitScripts.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get init scripts.`,
	Long: `Get init scripts.
  
  "Get a list of all global init scripts for this workspace. This returns all
  properties for each script but **not** the script contents. To retrieve the
  contents of a script, use the [get a global init
  script](#operation/get-script) operation.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.GlobalInitScripts.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq globalinitscripts.GlobalInitScriptUpdateRequest

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().BoolVar(&updateReq.Enabled, "enabled", updateReq.Enabled, `Specifies whether the script is enabled.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The name of the script.`)
	updateCmd.Flags().IntVar(&updateReq.Position, "position", updateReq.Position, `The position of a script, where 0 represents the first script to run, 1 is the second script to run, in ascending order.`)
	updateCmd.Flags().StringVar(&updateReq.Script, "script", updateReq.Script, `The Base64-encoded content of the script.`)
	updateCmd.Flags().StringVar(&updateReq.ScriptId, "script-id", updateReq.ScriptId, `The ID of the global init script.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update init script.`,
	Long: `Update init script.
  
  Updates a global init script, specifying only the fields to change. All fields
  are optional. Unspecified fields retain their current value.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.GlobalInitScripts.Update(ctx, updateReq)
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
