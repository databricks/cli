package dashboards

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/dbsql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "dashboards",
	Short: `In general, there is little need to modify dashboards using the API.`,
}

var createDashboardReq dbsql.CreateDashboardRequest

func init() {
	Cmd.AddCommand(createDashboardCmd)
	// TODO: short flags

	createDashboardCmd.Flags().BoolVar(&createDashboardReq.DashboardFiltersEnabled, "dashboard-filters-enabled", false, `In the web application, query filters that share a name are coupled to a single selection box if this value is true.`)
	createDashboardCmd.Flags().BoolVar(&createDashboardReq.IsDraft, "is-draft", false, `Draft dashboards only appear in list views for their owners.`)
	createDashboardCmd.Flags().BoolVar(&createDashboardReq.IsTrashed, "is-trashed", false, `Indicates whether the dashboard is trashed.`)
	createDashboardCmd.Flags().StringVar(&createDashboardReq.Name, "name", "", `The title of this dashboard that appears in list views and at the top of the dashboard page.`)
	// TODO: complex arg: tags
	// TODO: complex arg: widgets

}

var createDashboardCmd = &cobra.Command{
	Use:   "create-dashboard",
	Short: `Create a dashboard object.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Dashboards.CreateDashboard(ctx, createDashboardReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var deleteDashboardReq dbsql.DeleteDashboardRequest

func init() {
	Cmd.AddCommand(deleteDashboardCmd)
	// TODO: short flags

	deleteDashboardCmd.Flags().StringVar(&deleteDashboardReq.DashboardId, "dashboard-id", "", ``)

}

var deleteDashboardCmd = &cobra.Command{
	Use:   "delete-dashboard",
	Short: `Remove a dashboard.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	getDashboardCmd.Flags().StringVar(&getDashboardReq.DashboardId, "dashboard-id", "", ``)

}

var getDashboardCmd = &cobra.Command{
	Use:   "get-dashboard",
	Short: `Retrieve a definition.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Dashboards.GetDashboard(ctx, getDashboardReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var listDashboardsReq dbsql.ListDashboardsRequest

func init() {
	Cmd.AddCommand(listDashboardsCmd)
	// TODO: short flags

	// TODO: complex arg: order
	listDashboardsCmd.Flags().IntVar(&listDashboardsReq.Page, "page", 0, `Page number to retrieve.`)
	listDashboardsCmd.Flags().IntVar(&listDashboardsReq.PageSize, "page-size", 0, `Number of dashboards to return per page.`)
	listDashboardsCmd.Flags().StringVar(&listDashboardsReq.Q, "q", "", `Full text search term.`)

}

var listDashboardsCmd = &cobra.Command{
	Use:   "list-dashboards",
	Short: `Get dashboard objects.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Dashboards.ListDashboardsAll(ctx, listDashboardsReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var restoreDashboardReq dbsql.RestoreDashboardRequest

func init() {
	Cmd.AddCommand(restoreDashboardCmd)
	// TODO: short flags

	restoreDashboardCmd.Flags().StringVar(&restoreDashboardReq.DashboardId, "dashboard-id", "", ``)

}

var restoreDashboardCmd = &cobra.Command{
	Use:   "restore-dashboard",
	Short: `Restore a dashboard.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Dashboards.RestoreDashboard(ctx, restoreDashboardReq)
		if err != nil {
			return err
		}

		return nil
	},
}
