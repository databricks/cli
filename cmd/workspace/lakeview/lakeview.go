// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package lakeview

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
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
		Use:   "lakeview",
		Short: `These APIs provide specific management operations for Lakeview dashboards.`,
		Long: `These APIs provide specific management operations for Lakeview dashboards.
  Generic resource management can be done with Workspace API (import, export,
  get-status, list, delete).`,
		GroupID: "dashboards",
		Annotations: map[string]string{
			"package": "dashboards",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newCreateSchedule())
	cmd.AddCommand(newCreateSubscription())
	cmd.AddCommand(newDeleteSchedule())
	cmd.AddCommand(newDeleteSubscription())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetPublished())
	cmd.AddCommand(newGetSchedule())
	cmd.AddCommand(newGetSubscription())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListSchedules())
	cmd.AddCommand(newListSubscriptions())
	cmd.AddCommand(newMigrate())
	cmd.AddCommand(newPublish())
	cmd.AddCommand(newTrash())
	cmd.AddCommand(newUnpublish())
	cmd.AddCommand(newUpdate())
	cmd.AddCommand(newUpdateSchedule())

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
	*dashboards.CreateDashboardRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq dashboards.CreateDashboardRequest
	createReq.Dashboard = dashboards.Dashboard{}
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Dashboard.DisplayName, "display-name", createReq.Dashboard.DisplayName, `The display name of the dashboard.`)
	cmd.Flags().StringVar(&createReq.Dashboard.SerializedDashboard, "serialized-dashboard", createReq.Dashboard.SerializedDashboard, `The contents of the dashboard in serialized string form.`)
	cmd.Flags().StringVar(&createReq.Dashboard.WarehouseId, "warehouse-id", createReq.Dashboard.WarehouseId, `The warehouse ID used to run the dashboard.`)

	cmd.Use = "create"
	cmd.Short = `Create dashboard.`
	cmd.Long = `Create dashboard.
  
  Create a draft dashboard.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq.Dashboard)
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

		response, err := w.Lakeview.Create(ctx, createReq)
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

// start create-schedule command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createScheduleOverrides []func(
	*cobra.Command,
	*dashboards.CreateScheduleRequest,
)

func newCreateSchedule() *cobra.Command {
	cmd := &cobra.Command{}

	var createScheduleReq dashboards.CreateScheduleRequest
	createScheduleReq.Schedule = dashboards.Schedule{}
	var createScheduleJson flags.JsonFlag

	cmd.Flags().Var(&createScheduleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createScheduleReq.Schedule.DisplayName, "display-name", createScheduleReq.Schedule.DisplayName, `The display name for schedule.`)
	cmd.Flags().Var(&createScheduleReq.Schedule.PauseStatus, "pause-status", `The status indicates whether this schedule is paused or not. Supported values: [PAUSED, UNPAUSED]`)
	cmd.Flags().StringVar(&createScheduleReq.Schedule.WarehouseId, "warehouse-id", createScheduleReq.Schedule.WarehouseId, `The warehouse id to run the dashboard with for the schedule.`)

	cmd.Use = "create-schedule DASHBOARD_ID"
	cmd.Short = `Create dashboard schedule.`
	cmd.Long = `Create dashboard schedule.

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to which the schedule belongs.`

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
			diags := createScheduleJson.Unmarshal(&createScheduleReq.Schedule)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		createScheduleReq.DashboardId = args[0]

		response, err := w.Lakeview.CreateSchedule(ctx, createScheduleReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createScheduleOverrides {
		fn(cmd, &createScheduleReq)
	}

	return cmd
}

// start create-subscription command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSubscriptionOverrides []func(
	*cobra.Command,
	*dashboards.CreateSubscriptionRequest,
)

func newCreateSubscription() *cobra.Command {
	cmd := &cobra.Command{}

	var createSubscriptionReq dashboards.CreateSubscriptionRequest
	createSubscriptionReq.Subscription = dashboards.Subscription{}
	var createSubscriptionJson flags.JsonFlag

	cmd.Flags().Var(&createSubscriptionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create-subscription DASHBOARD_ID SCHEDULE_ID"
	cmd.Short = `Create schedule subscription.`
	cmd.Long = `Create schedule subscription.

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to which the subscription belongs.
    SCHEDULE_ID: UUID identifying the schedule to which the subscription belongs.`

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
			diags := createSubscriptionJson.Unmarshal(&createSubscriptionReq.Subscription)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		createSubscriptionReq.DashboardId = args[0]
		createSubscriptionReq.ScheduleId = args[1]

		response, err := w.Lakeview.CreateSubscription(ctx, createSubscriptionReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createSubscriptionOverrides {
		fn(cmd, &createSubscriptionReq)
	}

	return cmd
}

// start delete-schedule command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteScheduleOverrides []func(
	*cobra.Command,
	*dashboards.DeleteScheduleRequest,
)

func newDeleteSchedule() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteScheduleReq dashboards.DeleteScheduleRequest

	cmd.Flags().StringVar(&deleteScheduleReq.Etag, "etag", deleteScheduleReq.Etag, `The etag for the schedule.`)

	cmd.Use = "delete-schedule DASHBOARD_ID SCHEDULE_ID"
	cmd.Short = `Delete dashboard schedule.`
	cmd.Long = `Delete dashboard schedule.

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to which the schedule belongs.
    SCHEDULE_ID: UUID identifying the schedule.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteScheduleReq.DashboardId = args[0]
		deleteScheduleReq.ScheduleId = args[1]

		err = w.Lakeview.DeleteSchedule(ctx, deleteScheduleReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteScheduleOverrides {
		fn(cmd, &deleteScheduleReq)
	}

	return cmd
}

// start delete-subscription command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSubscriptionOverrides []func(
	*cobra.Command,
	*dashboards.DeleteSubscriptionRequest,
)

func newDeleteSubscription() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteSubscriptionReq dashboards.DeleteSubscriptionRequest

	cmd.Flags().StringVar(&deleteSubscriptionReq.Etag, "etag", deleteSubscriptionReq.Etag, `The etag for the subscription.`)

	cmd.Use = "delete-subscription DASHBOARD_ID SCHEDULE_ID SUBSCRIPTION_ID"
	cmd.Short = `Delete schedule subscription.`
	cmd.Long = `Delete schedule subscription.

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard which the subscription belongs.
    SCHEDULE_ID: UUID identifying the schedule which the subscription belongs.
    SUBSCRIPTION_ID: UUID identifying the subscription.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteSubscriptionReq.DashboardId = args[0]
		deleteSubscriptionReq.ScheduleId = args[1]
		deleteSubscriptionReq.SubscriptionId = args[2]

		err = w.Lakeview.DeleteSubscription(ctx, deleteSubscriptionReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSubscriptionOverrides {
		fn(cmd, &deleteSubscriptionReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*dashboards.GetDashboardRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq dashboards.GetDashboardRequest

	cmd.Use = "get DASHBOARD_ID"
	cmd.Short = `Get dashboard.`
	cmd.Long = `Get dashboard.
  
  Get a draft dashboard.

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.DashboardId = args[0]

		response, err := w.Lakeview.Get(ctx, getReq)
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

// start get-published command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPublishedOverrides []func(
	*cobra.Command,
	*dashboards.GetPublishedDashboardRequest,
)

func newGetPublished() *cobra.Command {
	cmd := &cobra.Command{}

	var getPublishedReq dashboards.GetPublishedDashboardRequest

	cmd.Use = "get-published DASHBOARD_ID"
	cmd.Short = `Get published dashboard.`
	cmd.Long = `Get published dashboard.
  
  Get the current published dashboard.

  Arguments:
    DASHBOARD_ID: UUID identifying the published dashboard.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPublishedReq.DashboardId = args[0]

		response, err := w.Lakeview.GetPublished(ctx, getPublishedReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPublishedOverrides {
		fn(cmd, &getPublishedReq)
	}

	return cmd
}

// start get-schedule command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getScheduleOverrides []func(
	*cobra.Command,
	*dashboards.GetScheduleRequest,
)

func newGetSchedule() *cobra.Command {
	cmd := &cobra.Command{}

	var getScheduleReq dashboards.GetScheduleRequest

	cmd.Use = "get-schedule DASHBOARD_ID SCHEDULE_ID"
	cmd.Short = `Get dashboard schedule.`
	cmd.Long = `Get dashboard schedule.

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to which the schedule belongs.
    SCHEDULE_ID: UUID identifying the schedule.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getScheduleReq.DashboardId = args[0]
		getScheduleReq.ScheduleId = args[1]

		response, err := w.Lakeview.GetSchedule(ctx, getScheduleReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getScheduleOverrides {
		fn(cmd, &getScheduleReq)
	}

	return cmd
}

// start get-subscription command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSubscriptionOverrides []func(
	*cobra.Command,
	*dashboards.GetSubscriptionRequest,
)

func newGetSubscription() *cobra.Command {
	cmd := &cobra.Command{}

	var getSubscriptionReq dashboards.GetSubscriptionRequest

	cmd.Use = "get-subscription DASHBOARD_ID SCHEDULE_ID SUBSCRIPTION_ID"
	cmd.Short = `Get schedule subscription.`
	cmd.Long = `Get schedule subscription.

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard which the subscription belongs.
    SCHEDULE_ID: UUID identifying the schedule which the subscription belongs.
    SUBSCRIPTION_ID: UUID identifying the subscription.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getSubscriptionReq.DashboardId = args[0]
		getSubscriptionReq.ScheduleId = args[1]
		getSubscriptionReq.SubscriptionId = args[2]

		response, err := w.Lakeview.GetSubscription(ctx, getSubscriptionReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSubscriptionOverrides {
		fn(cmd, &getSubscriptionReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*dashboards.ListDashboardsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq dashboards.ListDashboardsRequest

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `The number of dashboards to return per page.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `A page token, received from a previous ListDashboards call.`)
	cmd.Flags().BoolVar(&listReq.ShowTrashed, "show-trashed", listReq.ShowTrashed, `The flag to include dashboards located in the trash.`)
	cmd.Flags().Var(&listReq.View, "view", `DASHBOARD_VIEW_BASIConly includes summary metadata from the dashboard. Supported values: [DASHBOARD_VIEW_BASIC]`)

	cmd.Use = "list"
	cmd.Short = `List dashboards.`
	cmd.Long = `List dashboards.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Lakeview.List(ctx, listReq)
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

// start list-schedules command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSchedulesOverrides []func(
	*cobra.Command,
	*dashboards.ListSchedulesRequest,
)

func newListSchedules() *cobra.Command {
	cmd := &cobra.Command{}

	var listSchedulesReq dashboards.ListSchedulesRequest

	cmd.Flags().IntVar(&listSchedulesReq.PageSize, "page-size", listSchedulesReq.PageSize, `The number of schedules to return per page.`)
	cmd.Flags().StringVar(&listSchedulesReq.PageToken, "page-token", listSchedulesReq.PageToken, `A page token, received from a previous ListSchedules call.`)

	cmd.Use = "list-schedules DASHBOARD_ID"
	cmd.Short = `List dashboard schedules.`
	cmd.Long = `List dashboard schedules.

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to which the schedules belongs.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listSchedulesReq.DashboardId = args[0]

		response := w.Lakeview.ListSchedules(ctx, listSchedulesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSchedulesOverrides {
		fn(cmd, &listSchedulesReq)
	}

	return cmd
}

// start list-subscriptions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSubscriptionsOverrides []func(
	*cobra.Command,
	*dashboards.ListSubscriptionsRequest,
)

func newListSubscriptions() *cobra.Command {
	cmd := &cobra.Command{}

	var listSubscriptionsReq dashboards.ListSubscriptionsRequest

	cmd.Flags().IntVar(&listSubscriptionsReq.PageSize, "page-size", listSubscriptionsReq.PageSize, `The number of subscriptions to return per page.`)
	cmd.Flags().StringVar(&listSubscriptionsReq.PageToken, "page-token", listSubscriptionsReq.PageToken, `A page token, received from a previous ListSubscriptions call.`)

	cmd.Use = "list-subscriptions DASHBOARD_ID SCHEDULE_ID"
	cmd.Short = `List schedule subscriptions.`
	cmd.Long = `List schedule subscriptions.

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard which the subscriptions belongs.
    SCHEDULE_ID: UUID identifying the schedule which the subscriptions belongs.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listSubscriptionsReq.DashboardId = args[0]
		listSubscriptionsReq.ScheduleId = args[1]

		response := w.Lakeview.ListSubscriptions(ctx, listSubscriptionsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSubscriptionsOverrides {
		fn(cmd, &listSubscriptionsReq)
	}

	return cmd
}

// start migrate command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var migrateOverrides []func(
	*cobra.Command,
	*dashboards.MigrateDashboardRequest,
)

func newMigrate() *cobra.Command {
	cmd := &cobra.Command{}

	var migrateReq dashboards.MigrateDashboardRequest
	var migrateJson flags.JsonFlag

	cmd.Flags().Var(&migrateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&migrateReq.DisplayName, "display-name", migrateReq.DisplayName, `Display name for the new Lakeview dashboard.`)
	cmd.Flags().StringVar(&migrateReq.ParentPath, "parent-path", migrateReq.ParentPath, `The workspace path of the folder to contain the migrated Lakeview dashboard.`)
	cmd.Flags().BoolVar(&migrateReq.UpdateParameterSyntax, "update-parameter-syntax", migrateReq.UpdateParameterSyntax, `Flag to indicate if mustache parameter syntax ({{ param }}) should be auto-updated to named syntax (:param) when converting datasets in the dashboard.`)

	cmd.Use = "migrate SOURCE_DASHBOARD_ID"
	cmd.Short = `Migrate dashboard.`
	cmd.Long = `Migrate dashboard.
  
  Migrates a classic SQL dashboard to Lakeview.

  Arguments:
    SOURCE_DASHBOARD_ID: UUID of the dashboard to be migrated.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'source_dashboard_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := migrateJson.Unmarshal(&migrateReq)
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
			migrateReq.SourceDashboardId = args[0]
		}

		response, err := w.Lakeview.Migrate(ctx, migrateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range migrateOverrides {
		fn(cmd, &migrateReq)
	}

	return cmd
}

// start publish command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var publishOverrides []func(
	*cobra.Command,
	*dashboards.PublishRequest,
)

func newPublish() *cobra.Command {
	cmd := &cobra.Command{}

	var publishReq dashboards.PublishRequest
	var publishJson flags.JsonFlag

	cmd.Flags().Var(&publishJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&publishReq.EmbedCredentials, "embed-credentials", publishReq.EmbedCredentials, `Flag to indicate if the publisher's credentials should be embedded in the published dashboard.`)
	cmd.Flags().StringVar(&publishReq.WarehouseId, "warehouse-id", publishReq.WarehouseId, `The ID of the warehouse that can be used to override the warehouse which was set in the draft.`)

	cmd.Use = "publish DASHBOARD_ID"
	cmd.Short = `Publish dashboard.`
	cmd.Long = `Publish dashboard.
  
  Publish the current draft dashboard.

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to be published.`

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
			diags := publishJson.Unmarshal(&publishReq)
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
		publishReq.DashboardId = args[0]

		response, err := w.Lakeview.Publish(ctx, publishReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range publishOverrides {
		fn(cmd, &publishReq)
	}

	return cmd
}

// start trash command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var trashOverrides []func(
	*cobra.Command,
	*dashboards.TrashDashboardRequest,
)

func newTrash() *cobra.Command {
	cmd := &cobra.Command{}

	var trashReq dashboards.TrashDashboardRequest

	cmd.Use = "trash DASHBOARD_ID"
	cmd.Short = `Trash dashboard.`
	cmd.Long = `Trash dashboard.
  
  Trash a dashboard.

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		trashReq.DashboardId = args[0]

		err = w.Lakeview.Trash(ctx, trashReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range trashOverrides {
		fn(cmd, &trashReq)
	}

	return cmd
}

// start unpublish command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var unpublishOverrides []func(
	*cobra.Command,
	*dashboards.UnpublishDashboardRequest,
)

func newUnpublish() *cobra.Command {
	cmd := &cobra.Command{}

	var unpublishReq dashboards.UnpublishDashboardRequest

	cmd.Use = "unpublish DASHBOARD_ID"
	cmd.Short = `Unpublish dashboard.`
	cmd.Long = `Unpublish dashboard.
  
  Unpublish the dashboard.

  Arguments:
    DASHBOARD_ID: UUID identifying the published dashboard.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		unpublishReq.DashboardId = args[0]

		err = w.Lakeview.Unpublish(ctx, unpublishReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range unpublishOverrides {
		fn(cmd, &unpublishReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*dashboards.UpdateDashboardRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq dashboards.UpdateDashboardRequest
	updateReq.Dashboard = dashboards.Dashboard{}
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Dashboard.DisplayName, "display-name", updateReq.Dashboard.DisplayName, `The display name of the dashboard.`)
	cmd.Flags().StringVar(&updateReq.Dashboard.SerializedDashboard, "serialized-dashboard", updateReq.Dashboard.SerializedDashboard, `The contents of the dashboard in serialized string form.`)
	cmd.Flags().StringVar(&updateReq.Dashboard.WarehouseId, "warehouse-id", updateReq.Dashboard.WarehouseId, `The warehouse ID used to run the dashboard.`)

	cmd.Use = "update DASHBOARD_ID"
	cmd.Short = `Update dashboard.`
	cmd.Long = `Update dashboard.
  
  Update a draft dashboard.

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard.`

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
			diags := updateJson.Unmarshal(&updateReq.Dashboard)
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
		updateReq.DashboardId = args[0]

		response, err := w.Lakeview.Update(ctx, updateReq)
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

// start update-schedule command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateScheduleOverrides []func(
	*cobra.Command,
	*dashboards.UpdateScheduleRequest,
)

func newUpdateSchedule() *cobra.Command {
	cmd := &cobra.Command{}

	var updateScheduleReq dashboards.UpdateScheduleRequest
	updateScheduleReq.Schedule = dashboards.Schedule{}
	var updateScheduleJson flags.JsonFlag

	cmd.Flags().Var(&updateScheduleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateScheduleReq.Schedule.DisplayName, "display-name", updateScheduleReq.Schedule.DisplayName, `The display name for schedule.`)
	cmd.Flags().Var(&updateScheduleReq.Schedule.PauseStatus, "pause-status", `The status indicates whether this schedule is paused or not. Supported values: [PAUSED, UNPAUSED]`)
	cmd.Flags().StringVar(&updateScheduleReq.Schedule.WarehouseId, "warehouse-id", updateScheduleReq.Schedule.WarehouseId, `The warehouse id to run the dashboard with for the schedule.`)

	cmd.Use = "update-schedule DASHBOARD_ID SCHEDULE_ID"
	cmd.Short = `Update dashboard schedule.`
	cmd.Long = `Update dashboard schedule.

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to which the schedule belongs.
    SCHEDULE_ID: UUID identifying the schedule.`

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
			diags := updateScheduleJson.Unmarshal(&updateScheduleReq.Schedule)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		updateScheduleReq.DashboardId = args[0]
		updateScheduleReq.ScheduleId = args[1]

		response, err := w.Lakeview.UpdateSchedule(ctx, updateScheduleReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateScheduleOverrides {
		fn(cmd, &updateScheduleReq)
	}

	return cmd
}

// end service Lakeview
