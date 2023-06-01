// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package dashboards

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "dashboards",
	Short: `In general, there is little need to modify dashboards using the API.`,
	Long: `In general, there is little need to modify dashboards using the API. However,
  it can be useful to use dashboard objects to look-up a collection of related
  query IDs. The API can also be used to duplicate multiple dashboards at once
  since you can get a dashboard definition with a GET request and then POST it
  to create a new one. Dashboards can be scheduled using the sql_task type of
  the Jobs API, e.g. :method:jobs/create.`,
}

// start create command

var createReq sql.CreateDashboardRequest
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().BoolVar(&createReq.IsFavorite, "is-favorite", createReq.IsFavorite, `Indicates whether this query object should appear in the current user's favorites list.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `The title of this dashboard that appears in list views and at the top of the dashboard page.`)
	createCmd.Flags().StringVar(&createReq.Parent, "parent", createReq.Parent, `The identifier of the workspace folder containing the dashboard.`)
	// TODO: array: tags

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a dashboard object.`,
	Long:  `Create a dashboard object.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.Dashboards.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq sql.DeleteDashboardRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete DASHBOARD_ID",
	Short: `Remove a dashboard.`,
	Long: `Remove a dashboard.
  
  Moves a dashboard to the trash. Trashed dashboards do not appear in list views
  or searches, and cannot be shared.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "Loading prompts for missing command argument. You can cancel the process and provide an argument yourself instead."
				names, err := w.Dashboards.DashboardNameToIdMap(ctx, sql.ListDashboardsRequest{})
				close(promptSpinner)
				if err != nil {
					return err
				}
				id, err := cmdio.Select(ctx, names, "")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have ")
			}
			deleteReq.DashboardId = args[0]
		}

		err = w.Dashboards.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq sql.GetDashboardRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get DASHBOARD_ID",
	Short: `Retrieve a definition.`,
	Long: `Retrieve a definition.
  
  Returns a JSON representation of a dashboard object, including its
  visualization and query objects.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "Loading prompts for missing command argument. You can cancel the process and provide an argument yourself instead."
				names, err := w.Dashboards.DashboardNameToIdMap(ctx, sql.ListDashboardsRequest{})
				close(promptSpinner)
				if err != nil {
					return err
				}
				id, err := cmdio.Select(ctx, names, "")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have ")
			}
			getReq.DashboardId = args[0]
		}

		response, err := w.Dashboards.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

var listReq sql.ListDashboardsRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	listCmd.Flags().Var(&listReq.Order, "order", `Name of dashboard attribute to order by.`)
	listCmd.Flags().IntVar(&listReq.Page, "page", listReq.Page, `Page number to retrieve.`)
	listCmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Number of dashboards to return per page.`)
	listCmd.Flags().StringVar(&listReq.Q, "q", listReq.Q, `Full text search term.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get dashboard objects.`,
	Long: `Get dashboard objects.
  
  Fetch a paginated list of dashboard objects.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.Dashboards.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start restore command

var restoreReq sql.RestoreDashboardRequest
var restoreJson flags.JsonFlag

func init() {
	Cmd.AddCommand(restoreCmd)
	// TODO: short flags
	restoreCmd.Flags().Var(&restoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var restoreCmd = &cobra.Command{
	Use:   "restore DASHBOARD_ID",
	Short: `Restore a dashboard.`,
	Long: `Restore a dashboard.
  
  A restored dashboard appears in list views and searches and can be shared.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = restoreJson.Unmarshal(&restoreReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "Loading prompts for missing command argument. You can cancel the process and provide an argument yourself instead."
				names, err := w.Dashboards.DashboardNameToIdMap(ctx, sql.ListDashboardsRequest{})
				close(promptSpinner)
				if err != nil {
					return err
				}
				id, err := cmdio.Select(ctx, names, "")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have ")
			}
			restoreReq.DashboardId = args[0]
		}

		err = w.Dashboards.Restore(ctx, restoreReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Dashboards
