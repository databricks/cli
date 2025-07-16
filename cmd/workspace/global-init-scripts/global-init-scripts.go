// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package global_init_scripts

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
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
		Annotations: map[string]string{
			"package": "compute",
		},
		RunE: root.ReportUnknownSubcommand,
	}

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
	*compute.GlobalInitScriptCreateRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq compute.GlobalInitScriptCreateRequest
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&createReq.Enabled, "enabled", createReq.Enabled, `Specifies whether the script is enabled.`)
	cmd.Flags().IntVar(&createReq.Position, "position", createReq.Position, `The position of a global init script, where 0 represents the first script to run, 1 is the second script to run, in ascending order.`)

	cmd.Use = "create NAME SCRIPT"
	cmd.Short = `Create init script.`
	cmd.Long = `Create init script.
  
  Creates a new global init script in this workspace.

  Arguments:
    NAME: The name of the script
    SCRIPT: The Base64-encoded content of the script.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'script' in your JSON input")
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

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createReq.Script = args[1]
		}

		response, err := w.GlobalInitScripts.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*compute.DeleteGlobalInitScriptRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq compute.DeleteGlobalInitScriptRequest

	cmd.Use = "delete SCRIPT_ID"
	cmd.Short = `Delete init script.`
	cmd.Long = `Delete init script.
  
  Deletes a global init script.

  Arguments:
    SCRIPT_ID: The ID of the global init script.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No SCRIPT_ID argument specified. Loading names for Global Init Scripts drop-down."
			names, err := w.GlobalInitScripts.GlobalInitScriptDetailsNameToScriptIdMap(ctx)
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Global Init Scripts drop-down. Please manually specify required arguments. Original error: %w", err)
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
	*compute.GetGlobalInitScriptRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq compute.GetGlobalInitScriptRequest

	cmd.Use = "get SCRIPT_ID"
	cmd.Short = `Get an init script.`
	cmd.Long = `Get an init script.
  
  Gets all the details of a script, including its Base64-encoded contents.

  Arguments:
    SCRIPT_ID: The ID of the global init script.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No SCRIPT_ID argument specified. Loading names for Global Init Scripts drop-down."
			names, err := w.GlobalInitScripts.GlobalInitScriptDetailsNameToScriptIdMap(ctx)
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Global Init Scripts drop-down. Please manually specify required arguments. Original error: %w", err)
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

	cmd.Use = "list"
	cmd.Short = `Get init scripts.`
	cmd.Long = `Get init scripts.
  
  Get a list of all global init scripts for this workspace. This returns all
  properties for each script but **not** the script contents. To retrieve the
  contents of a script, use the [get a global init
  script](:method:globalinitscripts/get) operation.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response := w.GlobalInitScripts.List(ctx)
		return cmdio.RenderIterator(ctx, response)
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
	*compute.GlobalInitScriptUpdateRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq compute.GlobalInitScriptUpdateRequest
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&updateReq.Enabled, "enabled", updateReq.Enabled, `Specifies whether the script is enabled.`)
	cmd.Flags().IntVar(&updateReq.Position, "position", updateReq.Position, `The position of a script, where 0 represents the first script to run, 1 is the second script to run, in ascending order.`)

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only SCRIPT_ID as positional arguments. Provide 'name', 'script' in your JSON input")
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

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateReq.ScriptId = args[0]
		if !cmd.Flags().Changed("json") {
			updateReq.Name = args[1]
		}
		if !cmd.Flags().Changed("json") {
			updateReq.Script = args[2]
		}

		err = w.GlobalInitScripts.Update(ctx, updateReq)
		if err != nil {
			return err
		}
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
