// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package lakehouse_monitors

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
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
		Use:   "lakehouse-monitors",
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
	}

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
	*catalog.CreateMonitor,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq catalog.CreateMonitor
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.BaselineTableName, "baseline-table-name", createReq.BaselineTableName, `Name of the baseline table from which drift metrics are computed from.`)
	// TODO: array: custom_metrics
	// TODO: complex arg: data_classification_config
	// TODO: complex arg: inference_log
	// TODO: array: notifications
	// TODO: complex arg: schedule
	cmd.Flags().BoolVar(&createReq.SkipBuiltinDashboard, "skip-builtin-dashboard", createReq.SkipBuiltinDashboard, `Whether to skip creating a default dashboard summarizing data quality metrics.`)
	// TODO: array: slicing_exprs
	// TODO: output-only field
	// TODO: complex arg: time_series
	cmd.Flags().StringVar(&createReq.WarehouseId, "warehouse-id", createReq.WarehouseId, `Optional argument to specify the warehouse for dashboard creation.`)

	cmd.Use = "create FULL_NAME ASSETS_DIR OUTPUT_SCHEMA_NAME"
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
    FULL_NAME: Full name of the table.
    ASSETS_DIR: The directory to store monitoring assets (e.g. dashboard, metric tables).
    OUTPUT_SCHEMA_NAME: Schema where output metric tables are created.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only FULL_NAME as positional arguments. Provide 'assets_dir', 'output_schema_name' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		}
		createReq.FullName = args[0]
		if !cmd.Flags().Changed("json") {
			createReq.AssetsDir = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createReq.OutputSchemaName = args[2]
		}

		response, err := w.LakehouseMonitors.Create(ctx, createReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newCreate())
	})
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*catalog.DeleteLakehouseMonitorRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteLakehouseMonitorRequest

	// TODO: short flags

	cmd.Use = "delete FULL_NAME"
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
    FULL_NAME: Full name of the table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		deleteReq.FullName = args[0]

		err = w.LakehouseMonitors.Delete(ctx, deleteReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDelete())
	})
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*catalog.GetLakehouseMonitorRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetLakehouseMonitorRequest

	// TODO: short flags

	cmd.Use = "get FULL_NAME"
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
    FULL_NAME: Full name of the table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getReq.FullName = args[0]

		response, err := w.LakehouseMonitors.Get(ctx, getReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGet())
	})
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

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.BaselineTableName, "baseline-table-name", updateReq.BaselineTableName, `Name of the baseline table from which drift metrics are computed from.`)
	// TODO: array: custom_metrics
	// TODO: complex arg: data_classification_config
	// TODO: complex arg: inference_log
	// TODO: array: notifications
	// TODO: complex arg: schedule
	// TODO: array: slicing_exprs
	// TODO: output-only field
	// TODO: complex arg: time_series

	cmd.Use = "update FULL_NAME ASSETS_DIR OUTPUT_SCHEMA_NAME"
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
    FULL_NAME: Full name of the table.
    ASSETS_DIR: The directory to store monitoring assets (e.g. dashboard, metric tables).
    OUTPUT_SCHEMA_NAME: Schema where output metric tables are created.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only FULL_NAME as positional arguments. Provide 'assets_dir', 'output_schema_name' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		}
		updateReq.FullName = args[0]
		if !cmd.Flags().Changed("json") {
			updateReq.AssetsDir = args[1]
		}
		if !cmd.Flags().Changed("json") {
			updateReq.OutputSchemaName = args[2]
		}

		response, err := w.LakehouseMonitors.Update(ctx, updateReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdate())
	})
}

// end service LakehouseMonitors
