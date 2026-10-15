// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package dashboards

import (
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
		Use:   "dashboards",
		Short: `In general, there is little need to modify dashboards using the API.`,
		Long: `In general, there is little need to modify dashboards using the API. However,
  it can be useful to use dashboard objects to look-up a collection of related
  query IDs. The API can also be used to duplicate multiple dashboards at once
  since you can get a dashboard definition with a GET request and then POST it
  to create a new one. Dashboards can be scheduled using the sql_task type of
  the Jobs API, e.g. :method:jobs/create.

  **Warning**: This API is deprecated. Please use the AI/BI Dashboards API
  instead. [Learn more]

  [Learn more]: https://docs.databricks.com/en/dashboards/`,
		GroupID: "sql",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newRestore())
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
	var deleteDashboardIdRaw string

	cmd.Use = "delete DASHBOARD_ID"
	cmd.Short = `Remove a dashboard.`
	cmd.Long = `Remove a dashboard.

  Moves a dashboard to the trash. Trashed dashboards do not appear in list views
  or searches, and cannot be shared.

  **Warning**: This API is deprecated. Please use the AI/BI Dashboards API
  instead. [Learn more]

  [Learn more]: https://docs.databricks.com/en/dashboards/`

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

		deleteDashboardIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/dashboards/%s", fmt.Sprint(deleteDashboardIdRaw))

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
	var getDashboardIdRaw string

	cmd.Use = "get DASHBOARD_ID"
	cmd.Short = `Retrieve a definition.`
	cmd.Long = `Retrieve a definition.

  Returns a JSON representation of a dashboard object, including its
  visualization and query objects.

  **Warning**: This API is deprecated. Please use the AI/BI Dashboards API
  instead. [Learn more]

  [Learn more]: https://docs.databricks.com/en/dashboards/`

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

		getDashboardIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/dashboards/%s", fmt.Sprint(getDashboardIdRaw))

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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq map[string]any
	listReq = map[string]any{}
	var listOrderRaw string
	var listPageRaw int
	var listPageSizeRaw int
	var listQRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.EnumVar(cmd.Flags(), &listOrderRaw, "order", `Name of dashboard attribute to order by. Supported values: [created_at, name]`, "ListOrder", "created_at", "name")
	cmd.Flags().StringVar(&listQRaw, "q", listQRaw, `Full text search term.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().IntVar(&listPageRaw, "page", listPageRaw, `Page number to retrieve.`)
	cmd.Flags().Lookup("page").Hidden = true
	cmd.Flags().IntVar(&listPageSizeRaw, "page-size", listPageSizeRaw, `Number of results per API page.`)
	cmd.Flags().Lookup("page-size").Hidden = true

	cmd.Use = "list"
	cmd.Short = `Get dashboard objects.`
	cmd.Long = `Get dashboard objects.

  Fetch a paginated list of dashboard objects.

  **Warning**: Calling this API concurrently 10 or more times could result in
  throttling, service degradation, or a temporary ban.

  **Warning**: This API is deprecated. Please use the AI/BI Dashboards API
  instead. [Learn more]

  [Learn more]: https://docs.databricks.com/en/dashboards/`

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
		if cmd.Flags().Changed("order") || listOrderRaw != "" {
			rawQuery["order"] = listOrderRaw
		}
		if cmd.Flags().Changed("page") || listPageRaw != 0 {
			rawQuery["page"] = listPageRaw
		}
		if cmd.Flags().Changed("page-size") || listPageSizeRaw != 0 {
			rawQuery["page_size"] = listPageSizeRaw
		}
		if cmd.Flags().Changed("q") || listQRaw != "" {
			rawQuery["q"] = listQRaw
		}
		rawPath := "/api/2.0/preview/sql/dashboards"

		maps.Copy(rawQuery, listReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// start restore command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var restoreOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newRestore() *cobra.Command {
	cmd := &cobra.Command{}
	var restoreReq map[string]any
	restoreReq = map[string]any{}
	var restoreDashboardIdRaw string

	cmd.Use = "restore DASHBOARD_ID"
	cmd.Short = `Restore a dashboard.`
	cmd.Long = `Restore a dashboard.

  A restored dashboard appears in list views and searches and can be shared.

  **Warning**: This API is deprecated. Please use the AI/BI Dashboards API
  instead. [Learn more]

  [Learn more]: https://docs.databricks.com/en/dashboards/`

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

		restoreDashboardIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/dashboards/trash/%s", fmt.Sprint(restoreDashboardIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, restoreReq, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range restoreOverrides {
		fn(cmd, &restoreReq)
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
	var updateDashboardIdRaw string
	var updateNameRaw string
	var updateRunAsRoleRaw string

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateNameRaw, "name", updateNameRaw, `The title of this dashboard that appears in list views and at the top of the dashboard page.`)
	flags.EnumVar(cmd.Flags(), &updateRunAsRoleRaw, "run-as-role", `Sets the **Run as** role for the object. Supported values: [owner, viewer]`, "RunAsRole", "owner", "viewer")
	// TODO: array: tags

	cmd.Use = "update DASHBOARD_ID"
	cmd.Short = `Change a dashboard definition.`
	cmd.Long = `Change a dashboard definition.

  Modify this dashboard definition. This operation only affects attributes of
  the dashboard object. It does not add, modify, or remove widgets.

  **Note**: You cannot undo this operation.

  **Warning**: This API is deprecated. Please use the AI/BI Dashboards API
  instead. [Learn more]

  [Learn more]: https://docs.databricks.com/en/dashboards/`

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
		updateDashboardIdRaw = args[0]

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("name") || updateNameRaw != "" {
			updateReq["name"] = updateNameRaw
		}
		if cmd.Flags().Changed("run-as-role") || updateRunAsRoleRaw != "" {
			updateReq["run_as_role"] = updateRunAsRoleRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/dashboards/%s", fmt.Sprint(updateDashboardIdRaw))

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

// end service Dashboards
