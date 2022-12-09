package dashboards

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/dbsql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "dashboards",
	Short: `In general, there is little need to modify dashboards using the API.`,
	Long: `In general, there is little need to modify dashboards using the API. However,
  it can be useful to use dashboard objects to look-up a collection of related
  query IDs. The API can also be used to duplicate multiple dashboards at once
  since you can get a dashboard definition with a GET request and then POST it
  to create a new one.`,
}

var createDashboardReq dbsql.CreateDashboardRequest

func init() {
	Cmd.AddCommand(createDashboardCmd)
	// TODO: short flags

	createDashboardCmd.Flags().BoolVar(&createDashboardReq.DashboardFiltersEnabled, "dashboard-filters-enabled", createDashboardReq.DashboardFiltersEnabled, `In the web application, query filters that share a name are coupled to a single selection box if this value is true.`)
	createDashboardCmd.Flags().BoolVar(&createDashboardReq.IsDraft, "is-draft", createDashboardReq.IsDraft, `Draft dashboards only appear in list views for their owners.`)
	createDashboardCmd.Flags().BoolVar(&createDashboardReq.IsTrashed, "is-trashed", createDashboardReq.IsTrashed, `Indicates whether the dashboard is trashed.`)
	createDashboardCmd.Flags().StringVar(&createDashboardReq.Name, "name", createDashboardReq.Name, `The title of this dashboard that appears in list views and at the top of the dashboard page.`)
	// TODO: array: tags
	// TODO: array: widgets

}

var createDashboardCmd = &cobra.Command{
	Use:   "create-dashboard",
	Short: `Create a dashboard object.`,
	Long:  `Create a dashboard object.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Dashboards.CreateDashboard(ctx, createDashboardReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var deleteDashboardReq dbsql.DeleteDashboardRequest

func init() {
	Cmd.AddCommand(deleteDashboardCmd)
	// TODO: short flags

	deleteDashboardCmd.Flags().StringVar(&deleteDashboardReq.DashboardId, "dashboard-id", deleteDashboardReq.DashboardId, ``)

}

var deleteDashboardCmd = &cobra.Command{
	Use:   "delete-dashboard",
	Short: `Remove a dashboard.`,
	Long: `Remove a dashboard.
  
  Moves a dashboard to the trash. Trashed dashboards do not appear in list views
  or searches, and cannot be shared.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Dashboards.DeleteDashboard(ctx, deleteDashboardReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var getDashboardReq dbsql.GetDashboardRequest

func init() {
	Cmd.AddCommand(getDashboardCmd)
	// TODO: short flags

	getDashboardCmd.Flags().StringVar(&getDashboardReq.DashboardId, "dashboard-id", getDashboardReq.DashboardId, ``)

}

var getDashboardCmd = &cobra.Command{
	Use:   "get-dashboard",
	Short: `Retrieve a definition.`,
	Long: `Retrieve a definition.
  
  Returns a JSON representation of a dashboard object, including its
  visualization and query objects.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Dashboards.GetDashboard(ctx, getDashboardReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var listDashboardsReq dbsql.ListDashboardsRequest

func init() {
	Cmd.AddCommand(listDashboardsCmd)
	// TODO: short flags

	listDashboardsCmd.Flags().Var(&listDashboardsReq.Order, "order", `Name of dashboard attribute to order by.`)
	listDashboardsCmd.Flags().IntVar(&listDashboardsReq.Page, "page", listDashboardsReq.Page, `Page number to retrieve.`)
	listDashboardsCmd.Flags().IntVar(&listDashboardsReq.PageSize, "page-size", listDashboardsReq.PageSize, `Number of dashboards to return per page.`)
	listDashboardsCmd.Flags().StringVar(&listDashboardsReq.Q, "q", listDashboardsReq.Q, `Full text search term.`)

}

var listDashboardsCmd = &cobra.Command{
	Use:   "list-dashboards",
	Short: `Get dashboard objects.`,
	Long: `Get dashboard objects.
  
  Fetch a paginated list of dashboard objects.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Dashboards.ListDashboardsAll(ctx, listDashboardsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var restoreDashboardReq dbsql.RestoreDashboardRequest

func init() {
	Cmd.AddCommand(restoreDashboardCmd)
	// TODO: short flags

	restoreDashboardCmd.Flags().StringVar(&restoreDashboardReq.DashboardId, "dashboard-id", restoreDashboardReq.DashboardId, ``)

}

var restoreDashboardCmd = &cobra.Command{
	Use:   "restore-dashboard",
	Short: `Restore a dashboard.`,
	Long: `Restore a dashboard.
  
  A restored dashboard appears in list views and searches and can be shared.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Dashboards.RestoreDashboard(ctx, restoreDashboardReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Dashboards
