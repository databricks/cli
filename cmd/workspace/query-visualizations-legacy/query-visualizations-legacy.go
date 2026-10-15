// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package query_visualizations_legacy

import (
	"errors"
	"fmt"

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
		Use:   "query-visualizations-legacy",
		Short: `This is an evolving API that facilitates the addition and removal of visualizations from existing queries within the Databricks Workspace.`,
		Long: `This is an evolving API that facilitates the addition and removal of
  visualizations from existing queries within the Databricks Workspace. Data
  structures may change over time.

  **Warning**: This API is deprecated. Please see the latest version of the
  Databricks SQL API. [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`,
		GroupID: "sql",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
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
	*map[string]any,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq map[string]any
	createReq = map[string]any{}
	var createDescriptionRaw string
	var createNameRaw string
	var createQueryIdRaw string
	var createTypeRaw string

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createDescriptionRaw, "description", createDescriptionRaw, `A short description of this visualization.`)
	cmd.Flags().StringVar(&createNameRaw, "name", createNameRaw, `The name of the visualization that appears on dashboards and the query screen.`)

	cmd.Use = "create"
	cmd.Short = `Add visualization to a query.`
	cmd.Long = `Add visualization to a query.

  Creates visualization in the query.

  **Warning**: This API is deprecated. Please use
  :method:queryvisualizations/create instead. [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

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
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("description") || createDescriptionRaw != "" {
			createReq["description"] = createDescriptionRaw
		}
		if cmd.Flags().Changed("name") || createNameRaw != "" {
			createReq["name"] = createNameRaw
		}
		if cmd.Flags().Changed("query-id") || createQueryIdRaw != "" {
			createReq["query_id"] = createQueryIdRaw
		}
		if cmd.Flags().Changed("type") || createTypeRaw != "" {
			createReq["type"] = createTypeRaw
		}
		rawPath := "/api/2.0/preview/sql/visualizations"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, createReq, &response)
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
	*map[string]any,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq map[string]any
	deleteReq = map[string]any{}
	var deleteIdRaw string

	cmd.Use = "delete ID"
	cmd.Short = `Remove visualization.`
	cmd.Long = `Remove visualization.

  Removes a visualization from the query.

  **Warning**: This API is deprecated. Please use
  :method:queryvisualizations/delete instead. [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html

  Arguments:
    ID: Widget ID returned by :method:queryvisualizations/create`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/visualizations/%s", fmt.Sprint(deleteIdRaw))

		maps.Copy(rawQuery, deleteReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, nil)
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
	var updateCreatedAtRaw string
	var updateDescriptionRaw string
	var updateIdRaw string
	var updateNameRaw string
	var updateTypeRaw string
	var updateUpdatedAtRaw string

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateCreatedAtRaw, "created-at", updateCreatedAtRaw, ``)
	cmd.Flags().StringVar(&updateDescriptionRaw, "description", updateDescriptionRaw, `A short description of this visualization.`)
	cmd.Flags().StringVar(&updateIdRaw, "id", updateIdRaw, `The UUID for this visualization.`)
	cmd.Flags().StringVar(&updateNameRaw, "name", updateNameRaw, `The name of the visualization that appears on dashboards and the query screen.`)
	// TODO: any: options
	// TODO: complex arg: query
	cmd.Flags().StringVar(&updateTypeRaw, "type", updateTypeRaw, `The type of visualization: chart, table, pivot table, and so on.`)
	cmd.Flags().StringVar(&updateUpdatedAtRaw, "updated-at", updateUpdatedAtRaw, ``)

	cmd.Use = "update"
	cmd.Short = `Edit existing visualization.`
	cmd.Long = `Edit existing visualization.

  Updates visualization in the query.

  **Warning**: This API is deprecated. Please use
  :method:queryvisualizations/update instead. [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
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

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("created-at") || updateCreatedAtRaw != "" {
			updateReq["created_at"] = updateCreatedAtRaw
		}
		if cmd.Flags().Changed("description") || updateDescriptionRaw != "" {
			updateReq["description"] = updateDescriptionRaw
		}
		if cmd.Flags().Changed("id") || updateIdRaw != "" {
			updateReq["id"] = updateIdRaw
		}
		if cmd.Flags().Changed("name") || updateNameRaw != "" {
			updateReq["name"] = updateNameRaw
		}
		if cmd.Flags().Changed("type") || updateTypeRaw != "" {
			updateReq["type"] = updateTypeRaw
		}
		if cmd.Flags().Changed("updated-at") || updateUpdatedAtRaw != "" {
			updateReq["updated_at"] = updateUpdatedAtRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/visualizations/%s", fmt.Sprint(updateIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, updateReq, &response)
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

// end service QueryVisualizationsLegacy
