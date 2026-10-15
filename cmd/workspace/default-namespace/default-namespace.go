// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package default_namespace

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
		Use:   "default-namespace",
		Short: `*Public Preview* The default namespace setting API allows users to configure the default namespace for a Databricks workspace.`,
		Long: `This command is in Public Preview and may change without notice.

The default namespace setting API allows users to configure the default
  namespace for a Databricks workspace.

  Through this API, users can retrieve, set, or modify the default namespace
  used when queries do not reference a fully qualified three-level name. For
  example, if you use the API to set 'retail_prod' as the default catalog, then
  a query 'SELECT * FROM myTable' would reference the object
  'retail_prod.default.myTable' (the schema 'default' is always assumed).

  This setting requires a restart of clusters and SQL warehouses to take effect.
  Additionally, the default namespace only applies when using Unity
  Catalog-enabled compute.`,
		RunE: root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
	cmd.Short = `*Public Preview* Delete the default namespace setting.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete the default namespace setting.

  Deletes the default namespace setting for the workspace. A fresh etag needs to
  be provided in DELETE requests (as a query parameter). The etag can be
  retrieved by making a GET request before the DELETE request. If the
  setting is updated/deleted concurrently, DELETE fails with 409 and the
  request must be retried by using the fresh etag in the 409 response.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
		rawPath := "/api/2.0/settings/types/default_namespace_ws/names/default"

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
	cmd.Short = `*Public Preview* Get the default namespace setting.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get the default namespace setting.

  Gets the default namespace setting.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
		rawPath := "/api/2.0/settings/types/default_namespace_ws/names/default"

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
	cmd.Short = `*Public Preview* Update the default namespace setting.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update the default namespace setting.

  Updates the default namespace setting for the workspace. A fresh etag needs to
  be provided in PATCH requests (as part of the setting field). The etag can
  be retrieved by making a GET request before the PATCH request. Note that
  if the setting does not exist, GET returns a NOT_FOUND error and the etag is
  present in the error response, which should be set in the PATCH request. If
  the setting is updated concurrently, PATCH fails with 409 and the request
  must be retried by using the fresh etag in the 409 response.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
		rawPath := "/api/2.0/settings/types/default_namespace_ws/names/default"

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

// end service DefaultNamespace
