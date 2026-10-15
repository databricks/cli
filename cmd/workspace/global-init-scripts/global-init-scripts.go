// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package global_init_scripts

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	globalinitscripts "github.com/databricks/cli/libs/sdktypes/globalinitscripts/v2"
	sdkglobalinitscripts "github.com/databricks/sdk-go/globalinitscripts/v2"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
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
		GroupID: "compute",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*globalinitscripts.CreateGlobalInitScriptRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq globalinitscripts.CreateGlobalInitScriptRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &createReq.Enabled, "enabled", `Specifies whether the script is enabled.`)
	flags.IntPtrVar(cmd.Flags(), &createReq.Position, "position", `The position of a global init script, where 0 represents the first script to run, 1 is the second script to run, in ascending order.`)

	cmd.Use = "create NAME SCRIPT"
	cmd.Short = `Create init script.`
	cmd.Long = `Create init script.

  Creates a new global init script in this workspace.

  Arguments:
    NAME: The name of the script
    SCRIPT: The Base64-encoded content of the script.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'script' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			_ = args[1] // TODO: complex arg: script
		}

		sdkReq, err := globalinitscripts.CreateGlobalInitScriptRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.GlobalInitScripts.CreateGlobalInitScript(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := globalinitscripts.CreateGlobalInitScriptResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*sdkglobalinitscripts.DeleteGlobalInitScriptRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkglobalinitscripts.DeleteGlobalInitScriptRequest

	cmd.Use = "delete SCRIPT_ID"
	cmd.Short = `Delete init script.`
	cmd.Long = `Delete init script.

  Deletes a global init script.

  Arguments:
    SCRIPT_ID: The ID of the global init script.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteReq.ScriptId = new(args[0])

		sdkReq := &deleteReq
		response, err := w.GlobalInitScripts.DeleteGlobalInitScript(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*sdkglobalinitscripts.GetGlobalInitScriptRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkglobalinitscripts.GetGlobalInitScriptRequest

	cmd.Use = "get SCRIPT_ID"
	cmd.Short = `Get an init script.`
	cmd.Long = `Get an init script.

  Gets all the details of a script, including its Base64-encoded contents.

  Arguments:
    SCRIPT_ID: The ID of the global init script.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getReq.ScriptId = new(args[0])

		sdkReq := &getReq
		response, err := w.GlobalInitScripts.GetGlobalInitScript(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := globalinitscripts.GlobalInitScriptDetailsFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "list"
	cmd.Short = `Get init scripts.`
	cmd.Long = `Get init scripts.

  Get a list of all global init scripts for this workspace. This returns all
  properties for each script but **not** the script contents. To retrieve the
  contents of a script, use the [get a global init
  script](:method:globalinitscripts/get) operation.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq := &sdkglobalinitscripts.ListGlobalInitScriptsRequest{}

		response, err := w.GlobalInitScripts.ListGlobalInitScripts(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

		out, err := globalinitscripts.ListGlobalInitScriptsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out.Scripts)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*globalinitscripts.UpdateGlobalInitScriptRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq globalinitscripts.UpdateGlobalInitScriptRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &updateReq.Enabled, "enabled", `Specifies whether the script is enabled.`)
	flags.IntPtrVar(cmd.Flags(), &updateReq.Position, "position", `The position of a script, where 0 represents the first script to run, 1 is the second script to run, in ascending order.`)

	cmd.Use = "update SCRIPT_ID NAME SCRIPT"
	cmd.Short = `Update init script.`
	cmd.Long = `Update init script.

  Updates a global init script, specifying only the fields to change. All fields
  are optional. Unspecified fields retain their current value.

  Arguments:
    SCRIPT_ID: The ID of the global init script.
    NAME: The name of the script
    SCRIPT: The Base64-encoded content of the script.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only SCRIPT_ID as positional arguments. Provide 'name', 'script' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateReq.ScriptId = new(args[0])
		if !cmd.Flags().Changed("json") {
			updateReq.Name = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			_ = args[2] // TODO: complex arg: script
		}

		sdkReq, err := globalinitscripts.UpdateGlobalInitScriptRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.GlobalInitScripts.UpdateGlobalInitScript(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service GlobalInitScripts
