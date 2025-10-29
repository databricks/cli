// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package data_quality

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/dataquality"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data-quality",
		Short: `Manage the data quality of Unity Catalog objects (currently support schema and table).`,
		Long: `Manage the data quality of Unity Catalog objects (currently support schema
  and table)`,
		GroupID: "dataquality",
		Annotations: map[string]string{
			"package": "dataquality",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCancelRefresh())
	cmd.AddCommand(newCreateMonitor())
	cmd.AddCommand(newCreateRefresh())
	cmd.AddCommand(newDeleteMonitor())
	cmd.AddCommand(newDeleteRefresh())
	cmd.AddCommand(newGetMonitor())
	cmd.AddCommand(newGetRefresh())
	cmd.AddCommand(newListMonitor())
	cmd.AddCommand(newListRefresh())
	cmd.AddCommand(newUpdateMonitor())
	cmd.AddCommand(newUpdateRefresh())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start cancel-refresh command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cancelRefreshOverrides []func(
	*cobra.Command,
	*dataquality.CancelRefreshRequest,
)

func newCancelRefresh() *cobra.Command {
	cmd := &cobra.Command{}

	var cancelRefreshReq dataquality.CancelRefreshRequest

	cmd.Use = "cancel-refresh OBJECT_TYPE OBJECT_ID REFRESH_ID"
	cmd.Short = `Cancel a refresh.`
	cmd.Long = `Cancel a refresh.
  
  Cancels a data quality monitor refresh. Currently only supported for the
  table object_type.

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema or
      table.
    OBJECT_ID: The UUID of the request object. For example, schema id.
    REFRESH_ID: Unique id of the refresh operation.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		cancelRefreshReq.ObjectType = args[0]
		cancelRefreshReq.ObjectId = args[1]
		_, err = fmt.Sscan(args[2], &cancelRefreshReq.RefreshId)
		if err != nil {
			return fmt.Errorf("invalid REFRESH_ID: %s", args[2])
		}

		response, err := w.DataQuality.CancelRefresh(ctx, cancelRefreshReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range cancelRefreshOverrides {
		fn(cmd, &cancelRefreshReq)
	}

	return cmd
}

// start create-monitor command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createMonitorOverrides []func(
	*cobra.Command,
	*dataquality.CreateMonitorRequest,
)

func newCreateMonitor() *cobra.Command {
	cmd := &cobra.Command{}

	var createMonitorReq dataquality.CreateMonitorRequest
	createMonitorReq.Monitor = dataquality.Monitor{}
	var createMonitorJson flags.JsonFlag

	cmd.Flags().Var(&createMonitorJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: anomaly_detection_config
	// TODO: complex arg: data_profiling_config

	cmd.Use = "create-monitor OBJECT_TYPE OBJECT_ID"
	cmd.Short = `Create a monitor.`
	cmd.Long = `Create a monitor.
  
  Create a data quality monitor on a Unity Catalog object. The caller must
  provide either anomaly_detection_config for a schema monitor or
  data_profiling_config for a table monitor.
  
  For the table object_type, the caller must either: 1. be an owner of the
  table's parent catalog, have **USE_SCHEMA** on the table's parent schema, and
  have **SELECT** access on the table 2. have **USE_CATALOG** on the table's
  parent catalog, be an owner of the table's parent schema, and have **SELECT**
  access on the table. 3. have the following permissions: - **USE_CATALOG** on
  the table's parent catalog - **USE_SCHEMA** on the table's parent schema - be
  an owner of the table.
  
  Workspace assets, such as the dashboard, will be created in the workspace
  where this call was made.

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema or
      table.
    OBJECT_ID: The UUID of the request object. For example, schema id.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'object_type', 'object_id' in your JSON input")
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
			diags := createMonitorJson.Unmarshal(&createMonitorReq.Monitor)
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
			createMonitorReq.Monitor.ObjectType = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createMonitorReq.Monitor.ObjectId = args[1]
		}

		response, err := w.DataQuality.CreateMonitor(ctx, createMonitorReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createMonitorOverrides {
		fn(cmd, &createMonitorReq)
	}

	return cmd
}

// start create-refresh command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createRefreshOverrides []func(
	*cobra.Command,
	*dataquality.CreateRefreshRequest,
)

func newCreateRefresh() *cobra.Command {
	cmd := &cobra.Command{}

	var createRefreshReq dataquality.CreateRefreshRequest
	createRefreshReq.Refresh = dataquality.Refresh{}
	var createRefreshJson flags.JsonFlag

	cmd.Flags().Var(&createRefreshJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create-refresh OBJECT_TYPE OBJECT_ID"
	cmd.Short = `Create a refresh.`
	cmd.Long = `Create a refresh.
  
  Creates a refresh. Currently only supported for the table object_type.
  
  The caller must either: 1. be an owner of the table's parent catalog 2. have
  **USE_CATALOG** on the table's parent catalog and be an owner of the table's
  parent schema 3. have the following permissions: - **USE_CATALOG** on the
  table's parent catalog - **USE_SCHEMA** on the table's parent schema - be an
  owner of the table

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schemaor
      table.
    OBJECT_ID: The UUID of the request object. For example, table id.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createRefreshJson.Unmarshal(&createRefreshReq.Refresh)
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
		createRefreshReq.ObjectType = args[0]
		createRefreshReq.ObjectId = args[1]

		response, err := w.DataQuality.CreateRefresh(ctx, createRefreshReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createRefreshOverrides {
		fn(cmd, &createRefreshReq)
	}

	return cmd
}

// start delete-monitor command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteMonitorOverrides []func(
	*cobra.Command,
	*dataquality.DeleteMonitorRequest,
)

func newDeleteMonitor() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteMonitorReq dataquality.DeleteMonitorRequest

	cmd.Use = "delete-monitor OBJECT_TYPE OBJECT_ID"
	cmd.Short = `Delete a monitor.`
	cmd.Long = `Delete a monitor.
  
  Delete a data quality monitor on Unity Catalog object.
  
  For the table object_type, the caller must either: 1. be an owner of the
  table's parent catalog 2. have **USE_CATALOG** on the table's parent catalog
  and be an owner of the table's parent schema 3. have the following
  permissions: - **USE_CATALOG** on the table's parent catalog - **USE_SCHEMA**
  on the table's parent schema - be an owner of the table.
  
  Note that the metric tables and dashboard will not be deleted as part of this
  call; those assets must be manually cleaned up (if desired).

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema or
      table.
    OBJECT_ID: The UUID of the request object. For example, schema id.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteMonitorReq.ObjectType = args[0]
		deleteMonitorReq.ObjectId = args[1]

		err = w.DataQuality.DeleteMonitor(ctx, deleteMonitorReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteMonitorOverrides {
		fn(cmd, &deleteMonitorReq)
	}

	return cmd
}

// start delete-refresh command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteRefreshOverrides []func(
	*cobra.Command,
	*dataquality.DeleteRefreshRequest,
)

func newDeleteRefresh() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteRefreshReq dataquality.DeleteRefreshRequest

	cmd.Use = "delete-refresh OBJECT_TYPE OBJECT_ID REFRESH_ID"
	cmd.Short = `Delete a refresh.`
	cmd.Long = `Delete a refresh.
  
  (Unimplemented) Delete a refresh

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema or
      table.
    OBJECT_ID: The UUID of the request object. For example, schema id.
    REFRESH_ID: Unique id of the refresh operation.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteRefreshReq.ObjectType = args[0]
		deleteRefreshReq.ObjectId = args[1]
		_, err = fmt.Sscan(args[2], &deleteRefreshReq.RefreshId)
		if err != nil {
			return fmt.Errorf("invalid REFRESH_ID: %s", args[2])
		}

		err = w.DataQuality.DeleteRefresh(ctx, deleteRefreshReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteRefreshOverrides {
		fn(cmd, &deleteRefreshReq)
	}

	return cmd
}

// start get-monitor command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMonitorOverrides []func(
	*cobra.Command,
	*dataquality.GetMonitorRequest,
)

func newGetMonitor() *cobra.Command {
	cmd := &cobra.Command{}

	var getMonitorReq dataquality.GetMonitorRequest

	cmd.Use = "get-monitor OBJECT_TYPE OBJECT_ID"
	cmd.Short = `Read a monitor.`
	cmd.Long = `Read a monitor.
  
  Read a data quality monitor on Unity Catalog object.
  
  For the table object_type, the caller must either: 1. be an owner of the
  table's parent catalog 2. have **USE_CATALOG** on the table's parent catalog
  and be an owner of the table's parent schema. 3. have the following
  permissions: - **USE_CATALOG** on the table's parent catalog - **USE_SCHEMA**
  on the table's parent schema - **SELECT** privilege on the table.
  
  The returned information includes configuration values, as well as information
  on assets created by the monitor. Some information (e.g., dashboard) may be
  filtered out if the caller is in a different workspace than where the monitor
  was created.

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema or
      table.
    OBJECT_ID: The UUID of the request object. For example, schema id.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getMonitorReq.ObjectType = args[0]
		getMonitorReq.ObjectId = args[1]

		response, err := w.DataQuality.GetMonitor(ctx, getMonitorReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getMonitorOverrides {
		fn(cmd, &getMonitorReq)
	}

	return cmd
}

// start get-refresh command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getRefreshOverrides []func(
	*cobra.Command,
	*dataquality.GetRefreshRequest,
)

func newGetRefresh() *cobra.Command {
	cmd := &cobra.Command{}

	var getRefreshReq dataquality.GetRefreshRequest

	cmd.Use = "get-refresh OBJECT_TYPE OBJECT_ID REFRESH_ID"
	cmd.Short = `Get a refresh.`
	cmd.Long = `Get a refresh.
  
  Get data quality monitor refresh.
  
  For the table object_type, the caller must either: 1. be an owner of the
  table's parent catalog 2. have **USE_CATALOG** on the table's parent catalog
  and be an owner of the table's parent schema 3. have the following
  permissions: - **USE_CATALOG** on the table's parent catalog - **USE_SCHEMA**
  on the table's parent schema - **SELECT** privilege on the table.

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema or
      table.
    OBJECT_ID: The UUID of the request object. For example, schema id.
    REFRESH_ID: Unique id of the refresh operation.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getRefreshReq.ObjectType = args[0]
		getRefreshReq.ObjectId = args[1]
		_, err = fmt.Sscan(args[2], &getRefreshReq.RefreshId)
		if err != nil {
			return fmt.Errorf("invalid REFRESH_ID: %s", args[2])
		}

		response, err := w.DataQuality.GetRefresh(ctx, getRefreshReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getRefreshOverrides {
		fn(cmd, &getRefreshReq)
	}

	return cmd
}

// start list-monitor command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listMonitorOverrides []func(
	*cobra.Command,
	*dataquality.ListMonitorRequest,
)

func newListMonitor() *cobra.Command {
	cmd := &cobra.Command{}

	var listMonitorReq dataquality.ListMonitorRequest

	cmd.Flags().IntVar(&listMonitorReq.PageSize, "page-size", listMonitorReq.PageSize, ``)
	cmd.Flags().StringVar(&listMonitorReq.PageToken, "page-token", listMonitorReq.PageToken, ``)

	cmd.Use = "list-monitor"
	cmd.Short = `List monitors.`
	cmd.Long = `List monitors.
  
  (Unimplemented) List data quality monitors.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.DataQuality.ListMonitor(ctx, listMonitorReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listMonitorOverrides {
		fn(cmd, &listMonitorReq)
	}

	return cmd
}

// start list-refresh command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listRefreshOverrides []func(
	*cobra.Command,
	*dataquality.ListRefreshRequest,
)

func newListRefresh() *cobra.Command {
	cmd := &cobra.Command{}

	var listRefreshReq dataquality.ListRefreshRequest

	cmd.Flags().IntVar(&listRefreshReq.PageSize, "page-size", listRefreshReq.PageSize, ``)
	cmd.Flags().StringVar(&listRefreshReq.PageToken, "page-token", listRefreshReq.PageToken, ``)

	cmd.Use = "list-refresh OBJECT_TYPE OBJECT_ID"
	cmd.Short = `List refreshes.`
	cmd.Long = `List refreshes.
  
  List data quality monitor refreshes.
  
  For the table object_type, the caller must either: 1. be an owner of the
  table's parent catalog 2. have **USE_CATALOG** on the table's parent catalog
  and be an owner of the table's parent schema 3. have the following
  permissions: - **USE_CATALOG** on the table's parent catalog - **USE_SCHEMA**
  on the table's parent schema - **SELECT** privilege on the table.

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema or
      table.
    OBJECT_ID: The UUID of the request object. For example, schema id.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listRefreshReq.ObjectType = args[0]
		listRefreshReq.ObjectId = args[1]

		response := w.DataQuality.ListRefresh(ctx, listRefreshReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listRefreshOverrides {
		fn(cmd, &listRefreshReq)
	}

	return cmd
}

// start update-monitor command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateMonitorOverrides []func(
	*cobra.Command,
	*dataquality.UpdateMonitorRequest,
)

func newUpdateMonitor() *cobra.Command {
	cmd := &cobra.Command{}

	var updateMonitorReq dataquality.UpdateMonitorRequest
	updateMonitorReq.Monitor = dataquality.Monitor{}
	var updateMonitorJson flags.JsonFlag

	cmd.Flags().Var(&updateMonitorJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: anomaly_detection_config
	// TODO: complex arg: data_profiling_config

	cmd.Use = "update-monitor OBJECT_TYPE OBJECT_ID UPDATE_MASK OBJECT_TYPE OBJECT_ID"
	cmd.Short = `Update a monitor.`
	cmd.Long = `Update a monitor.
  
  Update a data quality monitor on Unity Catalog object.
  
  For the table object_type, The caller must either: 1. be an owner of the
  table's parent catalog 2. have **USE_CATALOG** on the table's parent catalog
  and be an owner of the table's parent schema 3. have the following
  permissions: - **USE_CATALOG** on the table's parent catalog - **USE_SCHEMA**
  on the table's parent schema - be an owner of the table.

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema or
      table.
    OBJECT_ID: The UUID of the request object. For example, schema id.
    UPDATE_MASK: The field mask to specify which fields to update as a comma-separated
      list. Example value:
      data_profiling_config.custom_metrics,data_profiling_config.schedule.quartz_cron_expression
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema or
      table.
    OBJECT_ID: The UUID of the request object. For example, schema id.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(3)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only OBJECT_TYPE, OBJECT_ID, UPDATE_MASK as positional arguments. Provide 'object_type', 'object_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(5)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateMonitorJson.Unmarshal(&updateMonitorReq.Monitor)
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
		updateMonitorReq.ObjectType = args[0]
		updateMonitorReq.ObjectId = args[1]
		updateMonitorReq.UpdateMask = args[2]
		if !cmd.Flags().Changed("json") {
			updateMonitorReq.Monitor.ObjectType = args[3]
		}
		if !cmd.Flags().Changed("json") {
			updateMonitorReq.Monitor.ObjectId = args[4]
		}

		response, err := w.DataQuality.UpdateMonitor(ctx, updateMonitorReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateMonitorOverrides {
		fn(cmd, &updateMonitorReq)
	}

	return cmd
}

// start update-refresh command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateRefreshOverrides []func(
	*cobra.Command,
	*dataquality.UpdateRefreshRequest,
)

func newUpdateRefresh() *cobra.Command {
	cmd := &cobra.Command{}

	var updateRefreshReq dataquality.UpdateRefreshRequest
	updateRefreshReq.Refresh = dataquality.Refresh{}
	var updateRefreshJson flags.JsonFlag

	cmd.Flags().Var(&updateRefreshJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update-refresh OBJECT_TYPE OBJECT_ID REFRESH_ID UPDATE_MASK OBJECT_TYPE OBJECT_ID"
	cmd.Short = `Update a refresh.`
	cmd.Long = `Update a refresh.
  
  (Unimplemented) Update a refresh

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema or
      table.
    OBJECT_ID: The UUID of the request object. For example, schema id.
    REFRESH_ID: Unique id of the refresh operation.
    UPDATE_MASK: The field mask to specify which fields to update.
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schemaor
      table.
    OBJECT_ID: The UUID of the request object. For example, table id.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(4)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only OBJECT_TYPE, OBJECT_ID, REFRESH_ID, UPDATE_MASK as positional arguments. Provide 'object_type', 'object_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(6)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateRefreshJson.Unmarshal(&updateRefreshReq.Refresh)
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
		updateRefreshReq.ObjectType = args[0]
		updateRefreshReq.ObjectId = args[1]
		_, err = fmt.Sscan(args[2], &updateRefreshReq.RefreshId)
		if err != nil {
			return fmt.Errorf("invalid REFRESH_ID: %s", args[2])
		}
		updateRefreshReq.UpdateMask = args[3]
		if !cmd.Flags().Changed("json") {
			updateRefreshReq.Refresh.ObjectType = args[4]
		}
		if !cmd.Flags().Changed("json") {
			updateRefreshReq.Refresh.ObjectId = args[5]
		}

		response, err := w.DataQuality.UpdateRefresh(ctx, updateRefreshReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateRefreshOverrides {
		fn(cmd, &updateRefreshReq)
	}

	return cmd
}

// end service DataQuality
