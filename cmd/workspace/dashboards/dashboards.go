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
  the Jobs API, e.g. :method:jobs/create.`,
		GroupID: "sql",
		Annotations: map[string]string{
			"package": "sql",
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
	*sql.DashboardPostContent,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq sql.DashboardPostContent
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create a dashboard object.`
	cmd.Long = `Create a dashboard object.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := w.Dashboards.Create(ctx, createReq)
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
	*sql.DeleteDashboardRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq sql.DeleteDashboardRequest

	// TODO: short flags

	cmd.Use = "delete DASHBOARD_ID"
	cmd.Short = `Remove a dashboard.`
	cmd.Long = `Remove a dashboard.
  
  Moves a dashboard to the trash. Trashed dashboards do not appear in list views
  or searches, and cannot be shared.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No DASHBOARD_ID argument specified. Loading names for Dashboards drop-down."
			names, err := w.Dashboards.DashboardNameToIdMap(ctx, sql.ListDashboardsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Dashboards drop-down. Please manually specify required arguments. Original error: %w", err)
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

		err = w.Dashboards.Delete(ctx, deleteReq)
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
	*sql.GetDashboardRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq sql.GetDashboardRequest

	// TODO: short flags

	cmd.Use = "get DASHBOARD_ID"
	cmd.Short = `Retrieve a definition.`
	cmd.Long = `Retrieve a definition.
  
  Returns a JSON representation of a dashboard object, including its
  visualization and query objects.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No DASHBOARD_ID argument specified. Loading names for Dashboards drop-down."
			names, err := w.Dashboards.DashboardNameToIdMap(ctx, sql.ListDashboardsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Dashboards drop-down. Please manually specify required arguments. Original error: %w", err)
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

		response, err := w.Dashboards.Get(ctx, getReq)
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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*sql.ListDashboardsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq sql.ListDashboardsRequest

	// TODO: short flags

	cmd.Flags().Var(&listReq.Order, "order", `Name of dashboard attribute to order by. Supported values: [created_at, name]`)
	cmd.Flags().IntVar(&listReq.Page, "page", listReq.Page, `Page number to retrieve.`)
	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Number of dashboards to return per page.`)
	cmd.Flags().StringVar(&listReq.Q, "q", listReq.Q, `Full text search term.`)

	cmd.Use = "list"
	cmd.Short = `Get dashboard objects.`
	cmd.Long = `Get dashboard objects.
  
  Fetch a paginated list of dashboard objects.
  
  ### **Warning: Calling this API concurrently 10 or more times could result in
  throttling, service degradation, or a temporary ban.**`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		response := w.Dashboards.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newList())
	})
}

// start restore command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var restoreOverrides []func(
	*cobra.Command,
	*sql.RestoreDashboardRequest,
)

func newRestore() *cobra.Command {
	cmd := &cobra.Command{}

	var restoreReq sql.RestoreDashboardRequest

	// TODO: short flags

	cmd.Use = "restore DASHBOARD_ID"
	cmd.Short = `Restore a dashboard.`
	cmd.Long = `Restore a dashboard.
  
  A restored dashboard appears in list views and searches and can be shared.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No DASHBOARD_ID argument specified. Loading names for Dashboards drop-down."
			names, err := w.Dashboards.DashboardNameToIdMap(ctx, sql.ListDashboardsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Dashboards drop-down. Please manually specify required arguments. Original error: %w", err)
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

		err = w.Dashboards.Restore(ctx, restoreReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newRestore())
	})
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*sql.DashboardEditContent,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq sql.DashboardEditContent
	var updateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The title of this dashboard that appears in list views and at the top of the dashboard page.`)
	cmd.Flags().Var(&updateReq.RunAsRole, "run-as-role", `Sets the **Run as** role for the object. Supported values: [owner, viewer]`)

	cmd.Use = "update DASHBOARD_ID"
	cmd.Short = `Change a dashboard definition.`
	cmd.Long = `Change a dashboard definition.
  
  Modify this dashboard definition. This operation only affects attributes of
  the dashboard object. It does not add, modify, or remove widgets.
  
  **Note**: You cannot undo this operation.`

	cmd.Annotations = make(map[string]string)

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
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No DASHBOARD_ID argument specified. Loading names for Dashboards drop-down."
			names, err := w.Dashboards.DashboardNameToIdMap(ctx, sql.ListDashboardsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Dashboards drop-down. Please manually specify required arguments. Original error: %w", err)
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
		updateReq.DashboardId = args[0]

		response, err := w.Dashboards.Update(ctx, updateReq)
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

// end service Dashboards
