package global_init_scripts

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/globalinitscripts"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "global-init-scripts",
	Short: `The Global Init Scripts API enables Workspace administrators to configure global initialization scripts for their workspace.`, // TODO: fix FirstSentence logic and append dot to summary
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
	Short: `Create init script Creates a new global init script in this workspace.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	Short: `Delete init script Deletes a global init script.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	Short: `Get an init script Gets all the details of a script, including its Base64-encoded contents.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	Short: `Get init scripts "Get a list of all global init scripts for this workspace.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	Short: `Update init script Updates a global init script, specifying only the fields to change.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.GlobalInitScripts.UpdateScript(ctx, updateScriptReq)
		if err != nil {
			return err
		}

		return nil
	},
}
