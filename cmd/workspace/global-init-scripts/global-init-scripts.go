// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package global_init_scripts

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/compute"
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

var createReq compute.GlobalInitScriptCreateRequest

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().BoolVar(&createReq.Enabled, "enabled", createReq.Enabled, `Specifies whether the script is enabled.`)
	createCmd.Flags().IntVar(&createReq.Position, "position", createReq.Position, `The position of a global init script, where 0 represents the first script to run, 1 is the second script to run, in ascending order.`)

}

var createCmd = &cobra.Command{
	Use:   "create NAME SCRIPT",
	Short: `Create init script.`,
	Long: `Create init script.
  
  Creates a new global init script in this workspace.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		createReq.Name = args[0]
		createReq.Script = args[1]

		response, err := w.GlobalInitScripts.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq compute.DeleteGlobalInitScriptRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete [SCRIPT_ID]",
	Short: `Delete init script.`,
	Long: `Delete init script.
  
  Deletes a global init script.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.GlobalInitScripts.GlobalInitScriptDetailsNameToScriptIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "The ID of the global init script")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the id of the global init script")
		}
		deleteReq.ScriptId = args[0]

		err = w.GlobalInitScripts.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq compute.GetGlobalInitScriptRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get [SCRIPT_ID]",
	Short: `Get an init script.`,
	Long: `Get an init script.
  
  Gets all the details of a script, including its Base64-encoded contents.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.GlobalInitScripts.GlobalInitScriptDetailsNameToScriptIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "The ID of the global init script")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the id of the global init script")
		}
		getReq.ScriptId = args[0]

		response, err := w.GlobalInitScripts.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
  
  Get a list of all global init scripts for this workspace. This returns all
  properties for each script but **not** the script contents. To retrieve the
  contents of a script, use the [get a global init
  script](#operation/get-script) operation.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.GlobalInitScripts.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq compute.GlobalInitScriptUpdateRequest

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().BoolVar(&updateReq.Enabled, "enabled", updateReq.Enabled, `Specifies whether the script is enabled.`)
	updateCmd.Flags().IntVar(&updateReq.Position, "position", updateReq.Position, `The position of a script, where 0 represents the first script to run, 1 is the second script to run, in ascending order.`)

}

var updateCmd = &cobra.Command{
	Use:   "update NAME SCRIPT SCRIPT_ID",
	Short: `Update init script.`,
	Long: `Update init script.
  
  Updates a global init script, specifying only the fields to change. All fields
  are optional. Unspecified fields retain their current value.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(3),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		updateReq.Name = args[0]
		updateReq.Script = args[1]
		updateReq.ScriptId = args[2]

		err = w.GlobalInitScripts.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service GlobalInitScripts
