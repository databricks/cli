// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package lakeview

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	lakeview "github.com/databricks/cli/libs/sdktypes/lakeview/v1"
	sdklakeview "github.com/databricks/sdk-go/lakeview/v1"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	cmd.AddCommand(newRevert())
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
	*lakeview.CreateDashboardRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq lakeview.CreateDashboardRequest
	createReq.Dashboard = &lakeview.DashboardCreate{}

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.DatasetCatalog, "dataset-catalog", `Sets the default catalog for all datasets in this dashboard.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.DatasetSchema, "dataset-schema", `Sets the default schema for all datasets in this dashboard.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.Dashboard.DisplayName, "display-name", `The display name of the dashboard.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.Dashboard.SerializedDashboard, "serialized-dashboard", `The contents of the dashboard in serialized string form.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.Dashboard.WarehouseId, "warehouse-id", `The warehouse ID used to run the dashboard.`)

	cmd.Use = "create"
	cmd.Short = `Create dashboard.`
	cmd.Long = `Create dashboard.

  Create a draft dashboard.

  Requires the [Databricks SQL access] entitlement. Grant Databricks SQL access
  in addition to Workspace access.

  [Databricks SQL access]: https://docs.databricks.com/security/auth/entitlements`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq.Dashboard)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}

		sdkReq, err := lakeview.CreateDashboardRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Lakeview.CreateDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := lakeview.DashboardFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*lakeview.CreateScheduleRequest,
)

func newCreateSchedule() *cobra.Command {
	cmd := &cobra.Command{}
	var createScheduleReq lakeview.CreateScheduleRequest
	createScheduleReq.Schedule = &lakeview.ScheduleCreate{}

	var createScheduleJson flags.JsonFlag

	cmd.Flags().Var(&createScheduleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createScheduleReq.Schedule.DisplayName, "display-name", `The display name for schedule.`)
	flags.EnumVar(cmd.Flags(), &createScheduleReq.Schedule.PauseStatus, "pause-status", `The status indicates whether this schedule is paused or not. Supported values: [PAUSED, UNPAUSED]`, "SchedulePauseStatus", "PAUSED", "UNPAUSED")
	flags.StringPtrVar(cmd.Flags(), &createScheduleReq.Schedule.WarehouseId, "warehouse-id", `The warehouse id to run the dashboard with for the schedule.`)

	cmd.Use = "create-schedule DASHBOARD_ID CRON_SCHEDULE"
	cmd.Short = `Create dashboard schedule.`
	cmd.Long = `Create dashboard schedule.

  Create dashboard schedule.

  Requires the [Databricks SQL access] entitlement. Grant Databricks SQL access
  in addition to Workspace access.

  [Databricks SQL access]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to which the schedule belongs.
    CRON_SCHEDULE: The cron expression describing the frequency of the periodic refresh for
      this schedule.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only DASHBOARD_ID as positional arguments. Provide 'cron_schedule' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createScheduleJson.Unmarshal(&createScheduleReq.Schedule)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		createScheduleReq.Schedule.DashboardId = new(args[0])
		if !cmd.Flags().Changed("json") {
			_ = args[1] // TODO: complex arg: cron_schedule
		}

		sdkReq, err := lakeview.CreateScheduleRequestToSDK(&createScheduleReq)
		if err != nil {
			return err
		}
		response, err := w.Lakeview.CreateSchedule(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := lakeview.ScheduleFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*lakeview.CreateSubscriptionRequest,
)

func newCreateSubscription() *cobra.Command {
	cmd := &cobra.Command{}
	var createSubscriptionReq lakeview.CreateSubscriptionRequest
	createSubscriptionReq.Subscription = &lakeview.SubscriptionCreate{}

	var createSubscriptionJson flags.JsonFlag

	cmd.Flags().Var(&createSubscriptionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &createSubscriptionReq.Subscription.SkipNotify, "skip-notify", `Controls whether notifications are sent to the subscriber for scheduled dashboard refreshes.`)

	cmd.Use = "create-subscription DASHBOARD_ID SCHEDULE_ID SUBSCRIBER"
	cmd.Short = `Create schedule subscription.`
	cmd.Long = `Create schedule subscription.

  Create schedule subscription.

  The caller must be a workspace user with one of the following [entitlements]:
  Workspace access, Databricks SQL access, or Consumer access.

  Account-level users who are not members of the workspace cannot call this
  endpoint, even if the dashboard has been shared with them.

  [entitlements]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to which the subscription belongs.
    SCHEDULE_ID: UUID identifying the schedule to which the subscription belongs.
    SUBSCRIBER: Subscriber details for users and destinations to be added as subscribers
      to the schedule.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only DASHBOARD_ID, SCHEDULE_ID as positional arguments. Provide 'subscriber' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createSubscriptionJson.Unmarshal(&createSubscriptionReq.Subscription)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		createSubscriptionReq.Subscription.DashboardId = new(args[0])
		createSubscriptionReq.Subscription.ScheduleId = new(args[1])
		if !cmd.Flags().Changed("json") {
			_ = args[2] // TODO: complex arg: subscriber
		}

		sdkReq, err := lakeview.CreateSubscriptionRequestToSDK(&createSubscriptionReq)
		if err != nil {
			return err
		}
		response, err := w.Lakeview.CreateSubscription(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := lakeview.SubscriptionFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*lakeview.DeleteScheduleRequest,
)

func newDeleteSchedule() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteScheduleReq lakeview.DeleteScheduleRequest

	flags.StringPtrVar(cmd.Flags(), &deleteScheduleReq.Etag, "etag", `The etag for the schedule.`)

	cmd.Use = "delete-schedule DASHBOARD_ID SCHEDULE_ID"
	cmd.Short = `Delete dashboard schedule.`
	cmd.Long = `Delete dashboard schedule.

  Delete dashboard schedule.

  Requires the [Databricks SQL access] entitlement. Grant Databricks SQL access
  in addition to Workspace access.

  [Databricks SQL access]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to which the schedule belongs.
    SCHEDULE_ID: UUID identifying the schedule.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteScheduleReq.DashboardId = new(args[0])
		deleteScheduleReq.ScheduleId = new(args[1])

		sdkReq, err := lakeview.DeleteScheduleRequestToSDK(&deleteScheduleReq)
		if err != nil {
			return err
		}
		err = w.Lakeview.DeleteSchedule(ctx, *sdkReq)
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
	*lakeview.DeleteSubscriptionRequest,
)

func newDeleteSubscription() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteSubscriptionReq lakeview.DeleteSubscriptionRequest

	flags.StringPtrVar(cmd.Flags(), &deleteSubscriptionReq.Etag, "etag", `The etag for the subscription.`)

	cmd.Use = "delete-subscription DASHBOARD_ID SCHEDULE_ID SUBSCRIPTION_ID"
	cmd.Short = `Delete schedule subscription.`
	cmd.Long = `Delete schedule subscription.

  Delete schedule subscription.

  The caller must be a workspace user with one of the following [entitlements]:
  Workspace access, Databricks SQL access, or Consumer access.

  Account-level users who are not members of the workspace cannot call this
  endpoint, even if the dashboard has been shared with them.

  [entitlements]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard which the subscription belongs.
    SCHEDULE_ID: UUID identifying the schedule which the subscription belongs.
    SUBSCRIPTION_ID: UUID identifying the subscription.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteSubscriptionReq.DashboardId = new(args[0])
		deleteSubscriptionReq.ScheduleId = new(args[1])
		deleteSubscriptionReq.SubscriptionId = new(args[2])

		sdkReq, err := lakeview.DeleteSubscriptionRequestToSDK(&deleteSubscriptionReq)
		if err != nil {
			return err
		}
		err = w.Lakeview.DeleteSubscription(ctx, *sdkReq)
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
	*sdklakeview.GetDashboardRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdklakeview.GetDashboardRequest

	cmd.Use = "get DASHBOARD_ID"
	cmd.Short = `Get dashboard.`
	cmd.Long = `Get dashboard.

  Get a draft dashboard.

  Requires the [Databricks SQL access] entitlement. Grant Databricks SQL access
  in addition to Workspace access.

  [Databricks SQL access]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getReq.DashboardId = new(args[0])

		sdkReq := &getReq
		response, err := w.Lakeview.GetDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := lakeview.DashboardFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdklakeview.GetPublishedDashboardRequest,
)

func newGetPublished() *cobra.Command {
	cmd := &cobra.Command{}
	var getPublishedReq sdklakeview.GetPublishedDashboardRequest

	cmd.Use = "get-published DASHBOARD_ID"
	cmd.Short = `Get published dashboard.`
	cmd.Long = `Get published dashboard.

  Get the current published dashboard.

  The caller must be a workspace user with one of the following entitlements:
  Workspace access, Databricks SQL access, or Consumer access.

  Account-level users who are not members of the workspace cannot call this
  endpoint, even if the dashboard has been shared with them.

  Arguments:
    DASHBOARD_ID: UUID identifying the published dashboard.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getPublishedReq.DashboardId = new(args[0])

		sdkReq := &getPublishedReq
		response, err := w.Lakeview.GetPublishedDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := lakeview.PublishedDashboardFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdklakeview.GetScheduleRequest,
)

func newGetSchedule() *cobra.Command {
	cmd := &cobra.Command{}
	var getScheduleReq sdklakeview.GetScheduleRequest

	cmd.Use = "get-schedule DASHBOARD_ID SCHEDULE_ID"
	cmd.Short = `Get dashboard schedule.`
	cmd.Long = `Get dashboard schedule.

  Get dashboard schedule.

  The caller must be a workspace user with one of the following [entitlements]:
  Workspace access, Databricks SQL access, or Consumer access.

  Account-level users who are not members of the workspace cannot call this
  endpoint, even if the dashboard has been shared with them.

  [entitlements]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to which the schedule belongs.
    SCHEDULE_ID: UUID identifying the schedule.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getScheduleReq.DashboardId = new(args[0])
		getScheduleReq.ScheduleId = new(args[1])

		sdkReq := &getScheduleReq
		response, err := w.Lakeview.GetSchedule(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := lakeview.ScheduleFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdklakeview.GetSubscriptionRequest,
)

func newGetSubscription() *cobra.Command {
	cmd := &cobra.Command{}
	var getSubscriptionReq sdklakeview.GetSubscriptionRequest

	cmd.Use = "get-subscription DASHBOARD_ID SCHEDULE_ID SUBSCRIPTION_ID"
	cmd.Short = `Get schedule subscription.`
	cmd.Long = `Get schedule subscription.

  Get schedule subscription.

  The caller must be a workspace user with one of the following [entitlements]:
  Workspace access, Databricks SQL access, or Consumer access.

  Account-level users who are not members of the workspace cannot call this
  endpoint, even if the dashboard has been shared with them.

  [entitlements]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard which the subscription belongs.
    SCHEDULE_ID: UUID identifying the schedule which the subscription belongs.
    SUBSCRIPTION_ID: UUID identifying the subscription.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getSubscriptionReq.DashboardId = new(args[0])
		getSubscriptionReq.ScheduleId = new(args[1])
		getSubscriptionReq.SubscriptionId = new(args[2])

		sdkReq := &getSubscriptionReq
		response, err := w.Lakeview.GetSubscription(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := lakeview.SubscriptionFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*lakeview.ListDashboardsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq lakeview.ListDashboardsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", `The number of dashboards to return per page.`)
	flags.BoolPtrVar(cmd.Flags(), &listReq.ShowTrashed, "show-trashed", `The flag to include dashboards located in the trash.`)
	flags.EnumVar(cmd.Flags(), &listReq.View, "view", `DASHBOARD_VIEW_BASIC only includes summary metadata from the dashboard. Supported values: [DASHBOARD_VIEW_BASIC]`, "DashboardView", "DASHBOARD_VIEW_BASIC")

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List dashboards.`
	cmd.Long = `List dashboards.

  List dashboards.

  Requires the [Databricks SQL access] entitlement. Grant Databricks SQL access
  in addition to Workspace access.

  [Databricks SQL access]: https://docs.databricks.com/security/auth/entitlements`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := lakeview.ListDashboardsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Lakeview.ListDashboardsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, lakeview.DashboardFromSDK))
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
	*lakeview.ListSchedulesRequest,
)

func newListSchedules() *cobra.Command {
	cmd := &cobra.Command{}
	var listSchedulesReq lakeview.ListSchedulesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSchedulesLimit int

	flags.IntPtrVar(cmd.Flags(), &listSchedulesReq.PageSize, "page-size", `The number of schedules to return per page.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSchedulesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listSchedulesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-schedules DASHBOARD_ID"
	cmd.Short = `List dashboard schedules.`
	cmd.Long = `List dashboard schedules.

  List dashboard schedules.

  The caller must be a workspace user with one of the following [entitlements]:
  Workspace access, Databricks SQL access, or Consumer access.

  Account-level users who are not members of the workspace cannot call this
  endpoint, even if the dashboard has been shared with them.

  [entitlements]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to which the schedules belongs.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listSchedulesReq.DashboardId = new(args[0])

		sdkReq, err := lakeview.ListSchedulesRequestToSDK(&listSchedulesReq)
		if err != nil {
			return err
		}
		if listSchedulesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSchedulesLimit)
		}
		if listSchedulesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSchedulesLimit)
		}
		response := w.Lakeview.ListSchedulesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, lakeview.ScheduleFromSDK))
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
	*lakeview.ListSubscriptionsRequest,
)

func newListSubscriptions() *cobra.Command {
	cmd := &cobra.Command{}
	var listSubscriptionsReq lakeview.ListSubscriptionsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSubscriptionsLimit int

	flags.IntPtrVar(cmd.Flags(), &listSubscriptionsReq.PageSize, "page-size", `The number of subscriptions to return per page.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSubscriptionsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listSubscriptionsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-subscriptions DASHBOARD_ID SCHEDULE_ID"
	cmd.Short = `List schedule subscriptions.`
	cmd.Long = `List schedule subscriptions.

  List schedule subscriptions.

  The caller must be a workspace user with one of the following [entitlements]:
  Workspace access, Databricks SQL access, or Consumer access.

  Account-level users who are not members of the workspace cannot call this
  endpoint, even if the dashboard has been shared with them.

  [entitlements]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard which the subscriptions belongs.
    SCHEDULE_ID: UUID identifying the schedule which the subscriptions belongs.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listSubscriptionsReq.DashboardId = new(args[0])
		listSubscriptionsReq.ScheduleId = new(args[1])

		sdkReq, err := lakeview.ListSubscriptionsRequestToSDK(&listSubscriptionsReq)
		if err != nil {
			return err
		}
		if listSubscriptionsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSubscriptionsLimit)
		}
		if listSubscriptionsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSubscriptionsLimit)
		}
		response := w.Lakeview.ListSubscriptionsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, lakeview.SubscriptionFromSDK))
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
	*lakeview.MigrateDashboardRequest,
)

func newMigrate() *cobra.Command {
	cmd := &cobra.Command{}
	var migrateReq lakeview.MigrateDashboardRequest

	var migrateJson flags.JsonFlag

	cmd.Flags().Var(&migrateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &migrateReq.DisplayName, "display-name", `Display name for the new Lakeview dashboard.`)
	flags.StringPtrVar(cmd.Flags(), &migrateReq.ParentPath, "parent-path", `The workspace path of the folder to contain the migrated Lakeview dashboard.`)
	flags.BoolPtrVar(cmd.Flags(), &migrateReq.UpdateParameterSyntax, "update-parameter-syntax", `Flag to indicate if mustache parameter syntax ({{ param }}) should be auto-updated to named syntax (:param) when converting datasets in the dashboard.`)

	cmd.Use = "migrate SOURCE_DASHBOARD_ID"
	cmd.Short = `Migrate dashboard.`
	cmd.Long = `Migrate dashboard.

  Deprecated: Legacy dashboard migration is no longer supported. Use Lakeview
  (AI/BI) dashboards instead.

  Arguments:
    SOURCE_DASHBOARD_ID: UUID of the dashboard to be migrated.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'source_dashboard_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := migrateJson.Unmarshal(&migrateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			migrateReq.SourceDashboardId = new(args[0])
		}

		sdkReq, err := lakeview.MigrateDashboardRequestToSDK(&migrateReq)
		if err != nil {
			return err
		}
		response, err := w.Lakeview.MigrateDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := lakeview.DashboardFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*lakeview.PublishDashboardRequest,
)

func newPublish() *cobra.Command {
	cmd := &cobra.Command{}
	var publishReq lakeview.PublishDashboardRequest

	var publishJson flags.JsonFlag

	cmd.Flags().Var(&publishJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &publishReq.EmbedCredentials, "embed-credentials", `Flag to indicate if the publisher's credentials should be embedded in the published dashboard.`)
	flags.StringPtrVar(cmd.Flags(), &publishReq.WarehouseId, "warehouse-id", `The ID of the warehouse that can be used to override the warehouse which was set in the draft.`)

	cmd.Use = "publish DASHBOARD_ID"
	cmd.Short = `Publish dashboard.`
	cmd.Long = `Publish dashboard.

  Publish the current draft dashboard.

  Requires the [Databricks SQL access] entitlement. Grant Databricks SQL access
  in addition to Workspace access.

  [Databricks SQL access]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to be published.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
			diags := publishJson.Unmarshal(&publishReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		publishReq.DashboardId = new(args[0])

		sdkReq, err := lakeview.PublishDashboardRequestToSDK(&publishReq)
		if err != nil {
			return err
		}
		response, err := w.Lakeview.PublishDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := lakeview.PublishedDashboardFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start revert command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var revertOverrides []func(
	*cobra.Command,
	*lakeview.RevertDashboardRequest,
)

func newRevert() *cobra.Command {
	cmd := &cobra.Command{}
	var revertReq lakeview.RevertDashboardRequest

	var revertJson flags.JsonFlag

	cmd.Flags().Var(&revertJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "revert DASHBOARD_ID"
	cmd.Short = `Revert dashboard.`
	cmd.Long = `Revert dashboard.

  Revert a dashboard's definition in draft mode to the last published version.

  Requires the [Databricks SQL access] entitlement. Grant Databricks SQL access
  in addition to Workspace access.

  [Databricks SQL access]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
			diags := revertJson.Unmarshal(&revertReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		revertReq.DashboardId = new(args[0])

		sdkReq, err := lakeview.RevertDashboardRequestToSDK(&revertReq)
		if err != nil {
			return err
		}
		response, err := w.Lakeview.RevertDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := lakeview.RevertDashboardResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range revertOverrides {
		fn(cmd, &revertReq)
	}

	return cmd
}

// start trash command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var trashOverrides []func(
	*cobra.Command,
	*sdklakeview.TrashDashboardRequest,
)

func newTrash() *cobra.Command {
	cmd := &cobra.Command{}
	var trashReq sdklakeview.TrashDashboardRequest

	cmd.Use = "trash DASHBOARD_ID"
	cmd.Short = `Trash dashboard.`
	cmd.Long = `Trash dashboard.

  Trash a dashboard.

  Requires the [Databricks SQL access] entitlement. Grant Databricks SQL access
  in addition to Workspace access.

  [Databricks SQL access]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		trashReq.DashboardId = new(args[0])

		sdkReq := &trashReq
		response, err := w.Lakeview.TrashDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
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
	*sdklakeview.UnpublishDashboardRequest,
)

func newUnpublish() *cobra.Command {
	cmd := &cobra.Command{}
	var unpublishReq sdklakeview.UnpublishDashboardRequest

	cmd.Use = "unpublish DASHBOARD_ID"
	cmd.Short = `Unpublish dashboard.`
	cmd.Long = `Unpublish dashboard.

  Unpublish the dashboard.

  Requires the [Databricks SQL access] entitlement. Grant Databricks SQL access
  in addition to Workspace access.

  [Databricks SQL access]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the published dashboard.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		unpublishReq.DashboardId = new(args[0])

		sdkReq := &unpublishReq
		response, err := w.Lakeview.UnpublishDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
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
	*lakeview.UpdateDashboardRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq lakeview.UpdateDashboardRequest
	updateReq.Dashboard = &lakeview.Dashboard{}

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateReq.DatasetCatalog, "dataset-catalog", `Sets the default catalog for all datasets in this dashboard.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.DatasetSchema, "dataset-schema", `Sets the default schema for all datasets in this dashboard.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Dashboard.DisplayName, "display-name", `The display name of the dashboard.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Dashboard.SerializedDashboard, "serialized-dashboard", `The contents of the dashboard in serialized string form.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Dashboard.WarehouseId, "warehouse-id", `The warehouse ID used to run the dashboard.`)

	cmd.Use = "update DASHBOARD_ID"
	cmd.Short = `Update dashboard.`
	cmd.Long = `Update dashboard.

  Update a draft dashboard.

  Requires the [Databricks SQL access] entitlement. Grant Databricks SQL access
  in addition to Workspace access.

  [Databricks SQL access]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
			diags := updateJson.Unmarshal(&updateReq.Dashboard)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateReq.Dashboard.DashboardId = new(args[0])

		sdkReq, err := lakeview.UpdateDashboardRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Lakeview.UpdateDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := lakeview.DashboardFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*lakeview.UpdateScheduleRequest,
)

func newUpdateSchedule() *cobra.Command {
	cmd := &cobra.Command{}
	var updateScheduleReq lakeview.UpdateScheduleRequest
	updateScheduleReq.Schedule = &lakeview.Schedule{}

	var updateScheduleJson flags.JsonFlag

	cmd.Flags().Var(&updateScheduleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateScheduleReq.Schedule.DisplayName, "display-name", `The display name for schedule.`)
	flags.EnumVar(cmd.Flags(), &updateScheduleReq.Schedule.PauseStatus, "pause-status", `The status indicates whether this schedule is paused or not. Supported values: [PAUSED, UNPAUSED]`, "SchedulePauseStatus", "PAUSED", "UNPAUSED")
	flags.StringPtrVar(cmd.Flags(), &updateScheduleReq.Schedule.WarehouseId, "warehouse-id", `The warehouse id to run the dashboard with for the schedule.`)

	cmd.Use = "update-schedule DASHBOARD_ID SCHEDULE_ID CRON_SCHEDULE"
	cmd.Short = `Update dashboard schedule.`
	cmd.Long = `Update dashboard schedule.

  Update dashboard schedule.

  Requires the [Databricks SQL access] entitlement. Grant Databricks SQL access
  in addition to Workspace access.

  [Databricks SQL access]: https://docs.databricks.com/security/auth/entitlements

  Arguments:
    DASHBOARD_ID: UUID identifying the dashboard to which the schedule belongs.
    SCHEDULE_ID: UUID identifying the schedule.
    CRON_SCHEDULE: The cron expression describing the frequency of the periodic refresh for
      this schedule.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only DASHBOARD_ID, SCHEDULE_ID as positional arguments. Provide 'cron_schedule' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateScheduleJson.Unmarshal(&updateScheduleReq.Schedule)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateScheduleReq.Schedule.DashboardId = new(args[0])
		updateScheduleReq.Schedule.ScheduleId = new(args[1])
		if !cmd.Flags().Changed("json") {
			_ = args[2] // TODO: complex arg: cron_schedule
		}

		sdkReq, err := lakeview.UpdateScheduleRequestToSDK(&updateScheduleReq)
		if err != nil {
			return err
		}
		response, err := w.Lakeview.UpdateSchedule(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := lakeview.ScheduleFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
