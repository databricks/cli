// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package disable_legacy_dbfs

import (
	"errors"

	"maps"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable-legacy-dbfs",
		Short: `Disabling legacy DBFS has the following implications: 1.`,
		Long: `Disabling legacy DBFS has the following implications:

  1. Access to DBFS root and DBFS mounts is disallowed (as well as the creation
  of new mounts). 2. Disables Databricks Runtime versions prior to 13.3LTS.

  When the setting is off, all DBFS functionality is enabled and no restrictions
  are imposed on Databricks Runtime versions. This setting can take up to 20
  minutes to take effect and requires a manual restart of all-purpose compute
  clusters and SQL warehouses.`,
		RunE: root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq map[string]any
	deleteReq = map[string]any{}
	var deleteEtagRaw string

	cmd.Flags().StringVar(&deleteEtagRaw, "etag", deleteEtagRaw, `etag used for versioning.`)

	cmd.Use = "delete"
	cmd.Short = `Delete the disable legacy DBFS setting.`
	cmd.Long = `Delete the disable legacy DBFS setting.

  Deletes the disable legacy DBFS setting for a workspace, reverting back to the
  default.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("etag") || deleteEtagRaw != "" {
			rawQuery["etag"] = deleteEtagRaw
		}
		rawPath := "/api/2.0/settings/types/disable_legacy_dbfs/names/default"

		maps.Copy(rawQuery, deleteReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "DELETE", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*map[string]any,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq map[string]any
	getReq = map[string]any{}
	var getEtagRaw string

	cmd.Flags().StringVar(&getEtagRaw, "etag", getEtagRaw, `etag used for versioning.`)

	cmd.Use = "get"
	cmd.Short = `Get the disable legacy DBFS setting.`
	cmd.Long = `Get the disable legacy DBFS setting.

  Gets the disable legacy DBFS setting.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("etag") || getEtagRaw != "" {
			rawQuery["etag"] = getEtagRaw
		}
		rawPath := "/api/2.0/settings/types/disable_legacy_dbfs/names/default"

		maps.Copy(rawQuery, getReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
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

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq map[string]any
	updateReq = map[string]any{}
	var updateAllowMissingRaw bool
	var updateFieldMaskRaw string

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update"
	cmd.Short = `Update the disable legacy DBFS setting.`
	cmd.Long = `Update the disable legacy DBFS setting.

  Updates the disable legacy DBFS setting for the workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("allow-missing") || updateAllowMissingRaw != false {
			updateReq["allow_missing"] = updateAllowMissingRaw
		}
		if cmd.Flags().Changed("field-mask") || updateFieldMaskRaw != "" {
			updateReq["field_mask"] = updateFieldMaskRaw
		}
		rawPath := "/api/2.0/settings/types/disable_legacy_dbfs/names/default"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, updateReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

// end service DisableLegacyDbfs
