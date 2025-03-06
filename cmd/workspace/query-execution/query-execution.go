// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package query_execution

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "query-execution",
		Short:   `Query execution APIs for AI / BI Dashboards.`,
		Long:    `Query execution APIs for AI / BI Dashboards`,
		GroupID: "dashboards",
		Annotations: map[string]string{
			"package": "dashboards",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
	}

	// Add methods
	cmd.AddCommand(newCancelPublishedQueryExecution())
	cmd.AddCommand(newExecutePublishedDashboardQuery())
	cmd.AddCommand(newPollPublishedQueryStatus())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start cancel-published-query-execution command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cancelPublishedQueryExecutionOverrides []func(
	*cobra.Command,
	*dashboards.CancelPublishedQueryExecutionRequest,
)

func newCancelPublishedQueryExecution() *cobra.Command {
	cmd := &cobra.Command{}

	var cancelPublishedQueryExecutionReq dashboards.CancelPublishedQueryExecutionRequest

	// TODO: short flags

	// TODO: array: tokens

	cmd.Use = "cancel-published-query-execution DASHBOARD_NAME DASHBOARD_REVISION_ID"
	cmd.Short = `Cancel the results for the a query for a published, embedded dashboard.`
	cmd.Long = `Cancel the results for the a query for a published, embedded dashboard.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		cancelPublishedQueryExecutionReq.DashboardName = args[0]
		cancelPublishedQueryExecutionReq.DashboardRevisionId = args[1]

		response, err := w.QueryExecution.CancelPublishedQueryExecution(ctx, cancelPublishedQueryExecutionReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range cancelPublishedQueryExecutionOverrides {
		fn(cmd, &cancelPublishedQueryExecutionReq)
	}

	return cmd
}

// start execute-published-dashboard-query command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var executePublishedDashboardQueryOverrides []func(
	*cobra.Command,
	*dashboards.ExecutePublishedDashboardQueryRequest,
)

func newExecutePublishedDashboardQuery() *cobra.Command {
	cmd := &cobra.Command{}

	var executePublishedDashboardQueryReq dashboards.ExecutePublishedDashboardQueryRequest
	var executePublishedDashboardQueryJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&executePublishedDashboardQueryJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&executePublishedDashboardQueryReq.OverrideWarehouseId, "override-warehouse-id", executePublishedDashboardQueryReq.OverrideWarehouseId, `A dashboard schedule can override the warehouse used as compute for processing the published dashboard queries.`)

	cmd.Use = "execute-published-dashboard-query DASHBOARD_NAME DASHBOARD_REVISION_ID"
	cmd.Short = `Execute a query for a published dashboard.`
	cmd.Long = `Execute a query for a published dashboard.

  Arguments:
    DASHBOARD_NAME: Dashboard name and revision_id is required to retrieve
      PublishedDatasetDataModel which contains the list of datasets,
      warehouse_id, and embedded_credentials
    DASHBOARD_REVISION_ID: `

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'dashboard_name', 'dashboard_revision_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := executePublishedDashboardQueryJson.Unmarshal(&executePublishedDashboardQueryReq)
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
			executePublishedDashboardQueryReq.DashboardName = args[0]
		}
		if !cmd.Flags().Changed("json") {
			executePublishedDashboardQueryReq.DashboardRevisionId = args[1]
		}

		err = w.QueryExecution.ExecutePublishedDashboardQuery(ctx, executePublishedDashboardQueryReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range executePublishedDashboardQueryOverrides {
		fn(cmd, &executePublishedDashboardQueryReq)
	}

	return cmd
}

// start poll-published-query-status command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var pollPublishedQueryStatusOverrides []func(
	*cobra.Command,
	*dashboards.PollPublishedQueryStatusRequest,
)

func newPollPublishedQueryStatus() *cobra.Command {
	cmd := &cobra.Command{}

	var pollPublishedQueryStatusReq dashboards.PollPublishedQueryStatusRequest

	// TODO: short flags

	// TODO: array: tokens

	cmd.Use = "poll-published-query-status DASHBOARD_NAME DASHBOARD_REVISION_ID"
	cmd.Short = `Poll the results for the a query for a published, embedded dashboard.`
	cmd.Long = `Poll the results for the a query for a published, embedded dashboard.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		pollPublishedQueryStatusReq.DashboardName = args[0]
		pollPublishedQueryStatusReq.DashboardRevisionId = args[1]

		response, err := w.QueryExecution.PollPublishedQueryStatus(ctx, pollPublishedQueryStatusReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range pollPublishedQueryStatusOverrides {
		fn(cmd, &pollPublishedQueryStatusReq)
	}

	return cmd
}

// end service QueryExecution
