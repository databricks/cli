// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package alerts_legacy

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
		Use:   "alerts-legacy",
		Short: `*Public Preview* The alerts API can be used to perform CRUD operations on alerts.`,
		Long: `This command is in Public Preview and may change without notice.

The alerts API can be used to perform CRUD operations on alerts. An alert is a
  Databricks SQL object that periodically runs a query, evaluates a condition of
  its result, and notifies one or more users and/or notification destinations if
  the condition was met. Alerts can be scheduled using the sql_task type of
  the Jobs API, e.g. :method:jobs/create.

  **Warning**: This API is deprecated. Please see the latest version of the
  Databricks SQL API. [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`,
		GroupID: "sql",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
	*map[string]any,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq map[string]any
	createReq = map[string]any{}
	var createNameRaw string
	var createParentRaw string
	var createQueryIdRaw string
	var createRearmRaw int

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createParentRaw, "parent", createParentRaw, `The identifier of the workspace folder containing the object.`)
	cmd.Flags().IntVar(&createRearmRaw, "rearm", createRearmRaw, `Number of seconds after being triggered before the alert rearms itself and can be triggered again.`)

	cmd.Use = "create"
	cmd.Short = `*Public Preview* Create an alert.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create an alert.

  Creates an alert. An alert is a Databricks SQL object that periodically runs a
  query, evaluates a condition of its result, and notifies users or notification
  destinations if the condition was met.

  **Warning**: This API is deprecated. Please use :method:alerts/create instead.
  [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
		if cmd.Flags().Changed("name") || createNameRaw != "" {
			createReq["name"] = createNameRaw
		}
		if cmd.Flags().Changed("parent") || createParentRaw != "" {
			createReq["parent"] = createParentRaw
		}
		if cmd.Flags().Changed("query-id") || createQueryIdRaw != "" {
			createReq["query_id"] = createQueryIdRaw
		}
		if cmd.Flags().Changed("rearm") || createRearmRaw != 0 {
			createReq["rearm"] = createRearmRaw
		}
		rawPath := "/api/2.0/preview/sql/alerts"

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
	var deleteAlertIdRaw string

	cmd.Use = "delete ALERT_ID"
	cmd.Short = `*Public Preview* Delete an alert.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete an alert.

  Deletes an alert. Deleted alerts are no longer accessible and cannot be
  restored. **Note**: Unlike queries and dashboards, alerts cannot be moved to
  the trash.

  **Warning**: This API is deprecated. Please use :method:alerts/delete instead.
  [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteAlertIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/alerts/%s", fmt.Sprint(deleteAlertIdRaw))

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
	var getAlertIdRaw string

	cmd.Use = "get ALERT_ID"
	cmd.Short = `*Public Preview* Get an alert.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get an alert.

  Gets an alert.

  **Warning**: This API is deprecated. Please use :method:alerts/get instead.
  [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getAlertIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/alerts/%s", fmt.Sprint(getAlertIdRaw))

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
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list"
	cmd.Short = `*Public Preview* Get alerts.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get alerts.

  Gets a list of alerts.

  **Warning**: This API is deprecated. Please use :method:alerts/list instead.
  [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w
		listReq := map[string]any{}
		rawQuery := map[string]any{}
		rawPath := "/api/2.0/preview/sql/alerts"

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
		return cmdio.Render(ctx, response)
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
	*map[string]any,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq map[string]any
	updateReq = map[string]any{}
	var updateAlertIdRaw string
	var updateNameRaw string
	var updateQueryIdRaw string
	var updateRearmRaw int

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&updateRearmRaw, "rearm", updateRearmRaw, `Number of seconds after being triggered before the alert rearms itself and can be triggered again.`)

	cmd.Use = "update ALERT_ID"
	cmd.Short = `*Public Preview* Update an alert.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update an alert.

  Updates an alert.

  **Warning**: This API is deprecated. Please use :method:alerts/update instead.
  [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}
		updateAlertIdRaw = args[0]

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("name") || updateNameRaw != "" {
			updateReq["name"] = updateNameRaw
		}
		if cmd.Flags().Changed("query-id") || updateQueryIdRaw != "" {
			updateReq["query_id"] = updateQueryIdRaw
		}
		if cmd.Flags().Changed("rearm") || updateRearmRaw != 0 {
			updateReq["rearm"] = updateRearmRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/alerts/%s", fmt.Sprint(updateAlertIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "PUT", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, updateReq, nil)
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

// end service AlertsLegacy
