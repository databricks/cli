// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package quality_monitors

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quality-monitors",
		Short: `A monitor computes and monitors data or model quality metrics for a table over time.`,
		Long: `A monitor computes and monitors data or model quality metrics for a table over
  time. It generates metrics tables and a dashboard that you can use to monitor
  table health and set alerts.
  
  Most write operations require the user to be the owner of the table (or its
  parent schema or parent catalog). Viewing the dashboard, computed metrics, or
  monitor configuration only requires the user to have **SELECT** privileges on
  the table (along with **USE_SCHEMA** and **USE_CATALOG**).`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCancelRefresh())
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetRefresh())
	cmd.AddCommand(newListRefreshes())
	cmd.AddCommand(newRegenerateDashboard())
	cmd.AddCommand(newRunRefresh())
	cmd.AddCommand(newUpdate())

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
	*catalog.CancelRefreshRequest,
)

func newCancelRefresh() *cobra.Command {
	cmd := &cobra.Command{}

	var cancelRefreshReq catalog.CancelRefreshRequest

	cmd.Use = "cancel-refresh TABLE_NAME REFRESH_ID"
	cmd.Short = `Cancel refresh.`
	cmd.Long = `Cancel refresh.
  
  Cancel an active monitor refresh for the given refresh ID.
  
  The caller must either: 1. be an owner of the table's parent catalog 2. have
  **USE_CATALOG** on the table's parent catalog and be an owner of the table's
  parent schema 3. have the following permissions: - **USE_CATALOG** on the
  table's parent catalog - **USE_SCHEMA** on the table's parent schema - be an
  owner of the table
  
  Additionally, the call must be made from the workspace where the monitor was
  created.

  Arguments:
    TABLE_NAME: Full name of the table.
    REFRESH_ID: ID of the refresh.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		cancelRefreshReq.TableName = args[0]
		cancelRefreshReq.RefreshId = args[1]

		err = w.QualityMonitors.CancelRefresh(ctx, cancelRefreshReq)
		if err != nil {
			return err
		}
		return nil
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

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*catalog.CreateMonitor,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq catalog.CreateMonitor
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.BaselineTableName, "baseline-table-name", createReq.BaselineTableName, `Name of the baseline table from which drift metrics are computed from.`)
	// TODO: array: custom_metrics
	// TODO: complex arg: data_classification_config
	// TODO: complex arg: inference_log
	// TODO: complex arg: notifications
	// TODO: complex arg: schedule
	cmd.Flags().BoolVar(&createReq.SkipBuiltinDashboard, "skip-builtin-dashboard", createReq.SkipBuiltinDashboard, `Whether to skip creating a default dashboard summarizing data quality metrics.`)
	// TODO: array: slicing_exprs
	// TODO: complex arg: snapshot
	// TODO: complex arg: time_series
	cmd.Flags().StringVar(&createReq.WarehouseId, "warehouse-id", createReq.WarehouseId, `Optional argument to specify the warehouse for dashboard creation.`)

	cmd.Use = "create TABLE_NAME ASSETS_DIR OUTPUT_SCHEMA_NAME"
	cmd.Short = `Create a table monitor.`
	cmd.Long = `Create a table monitor.
  
  Creates a new monitor for the specified table.
  
  The caller must either: 1. be an owner of the table's parent catalog, have
  **USE_SCHEMA** on the table's parent schema, and have **SELECT** access on the
  table 2. have **USE_CATALOG** on the table's parent catalog, be an owner of
  the table's parent schema, and have **SELECT** access on the table. 3. have
  the following permissions: - **USE_CATALOG** on the table's parent catalog -
  **USE_SCHEMA** on the table's parent schema - be an owner of the table.
  
  Workspace assets, such as the dashboard, will be created in the workspace
  where this call was made.

  Arguments:
    TABLE_NAME: Full name of the table.
    ASSETS_DIR: The directory to store monitoring assets (e.g. dashboard, metric tables).
    OUTPUT_SCHEMA_NAME: Schema where output metric tables are created.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only TABLE_NAME as positional arguments. Provide 'assets_dir', 'output_schema_name' in your JSON input")
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
		createReq.TableName = args[0]
		if !cmd.Flags().Changed("json") {
			createReq.AssetsDir = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createReq.OutputSchemaName = args[2]
		}

		response, err := w.QualityMonitors.Create(ctx, createReq)
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
	*catalog.DeleteQualityMonitorRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteQualityMonitorRequest

	cmd.Use = "delete TABLE_NAME"
	cmd.Short = `Delete a table monitor.`
	cmd.Long = `Delete a table monitor.
  
  Deletes a monitor for the specified table.
  
  The caller must either: 1. be an owner of the table's parent catalog 2. have
  **USE_CATALOG** on the table's parent catalog and be an owner of the table's
  parent schema 3. have the following permissions: - **USE_CATALOG** on the
  table's parent catalog - **USE_SCHEMA** on the table's parent schema - be an
  owner of the table.
  
  Additionally, the call must be made from the workspace where the monitor was
  created.
  
  Note that the metric tables and dashboard will not be deleted as part of this
  call; those assets must be manually cleaned up (if desired).

  Arguments:
    TABLE_NAME: Full name of the table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.TableName = args[0]

		err = w.QualityMonitors.Delete(ctx, deleteReq)
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
	*catalog.GetQualityMonitorRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetQualityMonitorRequest

	cmd.Use = "get TABLE_NAME"
	cmd.Short = `Get a table monitor.`
	cmd.Long = `Get a table monitor.
  
  Gets a monitor for the specified table.
  
  The caller must either: 1. be an owner of the table's parent catalog 2. have
  **USE_CATALOG** on the table's parent catalog and be an owner of the table's
  parent schema. 3. have the following permissions: - **USE_CATALOG** on the
  table's parent catalog - **USE_SCHEMA** on the table's parent schema -
  **SELECT** privilege on the table.
  
  The returned information includes configuration values, as well as information
  on assets created by the monitor. Some information (e.g., dashboard) may be
  filtered out if the caller is in a different workspace than where the monitor
  was created.

  Arguments:
    TABLE_NAME: Full name of the table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.TableName = args[0]

		response, err := w.QualityMonitors.Get(ctx, getReq)
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

// start get-refresh command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getRefreshOverrides []func(
	*cobra.Command,
	*catalog.GetRefreshRequest,
)

func newGetRefresh() *cobra.Command {
	cmd := &cobra.Command{}

	var getRefreshReq catalog.GetRefreshRequest

	cmd.Use = "get-refresh TABLE_NAME REFRESH_ID"
	cmd.Short = `Get refresh.`
	cmd.Long = `Get refresh.
  
  Gets info about a specific monitor refresh using the given refresh ID.
  
  The caller must either: 1. be an owner of the table's parent catalog 2. have
  **USE_CATALOG** on the table's parent catalog and be an owner of the table's
  parent schema 3. have the following permissions: - **USE_CATALOG** on the
  table's parent catalog - **USE_SCHEMA** on the table's parent schema -
  **SELECT** privilege on the table.
  
  Additionally, the call must be made from the workspace where the monitor was
  created.

  Arguments:
    TABLE_NAME: Full name of the table.
    REFRESH_ID: ID of the refresh.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getRefreshReq.TableName = args[0]
		getRefreshReq.RefreshId = args[1]

		response, err := w.QualityMonitors.GetRefresh(ctx, getRefreshReq)
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

// start list-refreshes command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listRefreshesOverrides []func(
	*cobra.Command,
	*catalog.ListRefreshesRequest,
)

func newListRefreshes() *cobra.Command {
	cmd := &cobra.Command{}

	var listRefreshesReq catalog.ListRefreshesRequest

	cmd.Use = "list-refreshes TABLE_NAME"
	cmd.Short = `List refreshes.`
	cmd.Long = `List refreshes.
  
  Gets an array containing the history of the most recent refreshes (up to 25)
  for this table.
  
  The caller must either: 1. be an owner of the table's parent catalog 2. have
  **USE_CATALOG** on the table's parent catalog and be an owner of the table's
  parent schema 3. have the following permissions: - **USE_CATALOG** on the
  table's parent catalog - **USE_SCHEMA** on the table's parent schema -
  **SELECT** privilege on the table.
  
  Additionally, the call must be made from the workspace where the monitor was
  created.

  Arguments:
    TABLE_NAME: Full name of the table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listRefreshesReq.TableName = args[0]

		response, err := w.QualityMonitors.ListRefreshes(ctx, listRefreshesReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listRefreshesOverrides {
		fn(cmd, &listRefreshesReq)
	}

	return cmd
}

// start regenerate-dashboard command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var regenerateDashboardOverrides []func(
	*cobra.Command,
	*catalog.RegenerateDashboardRequest,
)

func newRegenerateDashboard() *cobra.Command {
	cmd := &cobra.Command{}

	var regenerateDashboardReq catalog.RegenerateDashboardRequest
	var regenerateDashboardJson flags.JsonFlag

	cmd.Flags().Var(&regenerateDashboardJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&regenerateDashboardReq.WarehouseId, "warehouse-id", regenerateDashboardReq.WarehouseId, `Optional argument to specify the warehouse for dashboard regeneration.`)

	cmd.Use = "regenerate-dashboard TABLE_NAME"
	cmd.Short = `Regenerate a monitoring dashboard.`
	cmd.Long = `Regenerate a monitoring dashboard.
  
  Regenerates the monitoring dashboard for the specified table.
  
  The caller must either: 1. be an owner of the table's parent catalog 2. have
  **USE_CATALOG** on the table's parent catalog and be an owner of the table's
  parent schema 3. have the following permissions: - **USE_CATALOG** on the
  table's parent catalog - **USE_SCHEMA** on the table's parent schema - be an
  owner of the table
  
  The call must be made from the workspace where the monitor was created. The
  dashboard will be regenerated in the assets directory that was specified when
  the monitor was created.

  Arguments:
    TABLE_NAME: Full name of the table.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := regenerateDashboardJson.Unmarshal(&regenerateDashboardReq)
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
		regenerateDashboardReq.TableName = args[0]

		response, err := w.QualityMonitors.RegenerateDashboard(ctx, regenerateDashboardReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range regenerateDashboardOverrides {
		fn(cmd, &regenerateDashboardReq)
	}

	return cmd
}

// start run-refresh command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var runRefreshOverrides []func(
	*cobra.Command,
	*catalog.RunRefreshRequest,
)

func newRunRefresh() *cobra.Command {
	cmd := &cobra.Command{}

	var runRefreshReq catalog.RunRefreshRequest

	cmd.Use = "run-refresh TABLE_NAME"
	cmd.Short = `Queue a metric refresh for a monitor.`
	cmd.Long = `Queue a metric refresh for a monitor.
  
  Queues a metric refresh on the monitor for the specified table. The refresh
  will execute in the background.
  
  The caller must either: 1. be an owner of the table's parent catalog 2. have
  **USE_CATALOG** on the table's parent catalog and be an owner of the table's
  parent schema 3. have the following permissions: - **USE_CATALOG** on the
  table's parent catalog - **USE_SCHEMA** on the table's parent schema - be an
  owner of the table
  
  Additionally, the call must be made from the workspace where the monitor was
  created.

  Arguments:
    TABLE_NAME: Full name of the table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		runRefreshReq.TableName = args[0]

		response, err := w.QualityMonitors.RunRefresh(ctx, runRefreshReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range runRefreshOverrides {
		fn(cmd, &runRefreshReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*catalog.UpdateMonitor,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdateMonitor
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.BaselineTableName, "baseline-table-name", updateReq.BaselineTableName, `Name of the baseline table from which drift metrics are computed from.`)
	// TODO: array: custom_metrics
	cmd.Flags().StringVar(&updateReq.DashboardId, "dashboard-id", updateReq.DashboardId, `Id of dashboard that visualizes the computed metrics.`)
	// TODO: complex arg: data_classification_config
	// TODO: complex arg: inference_log
	// TODO: complex arg: notifications
	// TODO: complex arg: schedule
	// TODO: array: slicing_exprs
	// TODO: complex arg: snapshot
	// TODO: complex arg: time_series

	cmd.Use = "update TABLE_NAME OUTPUT_SCHEMA_NAME"
	cmd.Short = `Update a table monitor.`
	cmd.Long = `Update a table monitor.
  
  Updates a monitor for the specified table.
  
  The caller must either: 1. be an owner of the table's parent catalog 2. have
  **USE_CATALOG** on the table's parent catalog and be an owner of the table's
  parent schema 3. have the following permissions: - **USE_CATALOG** on the
  table's parent catalog - **USE_SCHEMA** on the table's parent schema - be an
  owner of the table.
  
  Additionally, the call must be made from the workspace where the monitor was
  created, and the caller must be the original creator of the monitor.
  
  Certain configuration fields, such as output asset identifiers, cannot be
  updated.

  Arguments:
    TABLE_NAME: Full name of the table.
    OUTPUT_SCHEMA_NAME: Schema where output metric tables are created.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only TABLE_NAME as positional arguments. Provide 'output_schema_name' in your JSON input")
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
		updateReq.TableName = args[0]
		if !cmd.Flags().Changed("json") {
			updateReq.OutputSchemaName = args[1]
		}

		response, err := w.QualityMonitors.Update(ctx, updateReq)
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

// end service QualityMonitors
