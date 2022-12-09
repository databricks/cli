package alerts

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/dbsql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "alerts",
	Short: `The alerts API can be used to perform CRUD operations on alerts.`,
	Long: `The alerts API can be used to perform CRUD operations on alerts. An alert is a
  Databricks SQL object that periodically runs a query, evaluates a condition of
  its result, and notifies one or more users and/or alert destinations if the
  condition was met.`,
}

var createAlertReq dbsql.EditAlert

func init() {
	Cmd.AddCommand(createAlertCmd)
	// TODO: short flags

	createAlertCmd.Flags().StringVar(&createAlertReq.AlertId, "alert-id", createAlertReq.AlertId, ``)
	createAlertCmd.Flags().StringVar(&createAlertReq.Name, "name", createAlertReq.Name, `Name of the alert.`)
	// TODO: complex arg: options
	createAlertCmd.Flags().StringVar(&createAlertReq.QueryId, "query-id", createAlertReq.QueryId, `ID of the query evaluated by the alert.`)
	createAlertCmd.Flags().IntVar(&createAlertReq.Rearm, "rearm", createAlertReq.Rearm, `Number of seconds after being triggered before the alert rearms itself and can be triggered again.`)

}

var createAlertCmd = &cobra.Command{
	Use:   "create-alert",
	Short: `Create an alert.`,
	Long: `Create an alert.
  
  Creates an alert. An alert is a Databricks SQL object that periodically runs a
  query, evaluates a condition of its result, and notifies users or alert
  destinations if the condition was met.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Alerts.CreateAlert(ctx, createAlertReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var createScheduleReq dbsql.CreateRefreshSchedule

func init() {
	Cmd.AddCommand(createScheduleCmd)
	// TODO: short flags

	createScheduleCmd.Flags().StringVar(&createScheduleReq.AlertId, "alert-id", createScheduleReq.AlertId, ``)
	createScheduleCmd.Flags().StringVar(&createScheduleReq.Cron, "cron", createScheduleReq.Cron, `Cron string representing the refresh schedule.`)
	createScheduleCmd.Flags().StringVar(&createScheduleReq.DataSourceId, "data-source-id", createScheduleReq.DataSourceId, `ID of the SQL warehouse to refresh with.`)

}

var createScheduleCmd = &cobra.Command{
	Use:   "create-schedule",
	Short: `Create a refresh schedule.`,
	Long: `Create a refresh schedule.
  
  Creates a new refresh schedule for an alert.
  
  **Note:** The structure of refresh schedules is subject to change.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Alerts.CreateSchedule(ctx, createScheduleReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var deleteAlertReq dbsql.DeleteAlertRequest

func init() {
	Cmd.AddCommand(deleteAlertCmd)
	// TODO: short flags

	deleteAlertCmd.Flags().StringVar(&deleteAlertReq.AlertId, "alert-id", deleteAlertReq.AlertId, ``)

}

var deleteAlertCmd = &cobra.Command{
	Use:   "delete-alert",
	Short: `Delete an alert.`,
	Long: `Delete an alert.
  
  Deletes an alert. Deleted alerts are no longer accessible and cannot be
  restored. **Note:** Unlike queries and dashboards, alerts cannot be moved to
  the trash.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Alerts.DeleteAlert(ctx, deleteAlertReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var deleteScheduleReq dbsql.DeleteScheduleRequest

func init() {
	Cmd.AddCommand(deleteScheduleCmd)
	// TODO: short flags

	deleteScheduleCmd.Flags().StringVar(&deleteScheduleReq.AlertId, "alert-id", deleteScheduleReq.AlertId, ``)
	deleteScheduleCmd.Flags().StringVar(&deleteScheduleReq.ScheduleId, "schedule-id", deleteScheduleReq.ScheduleId, ``)

}

var deleteScheduleCmd = &cobra.Command{
	Use:   "delete-schedule",
	Short: `Delete a refresh schedule.`,
	Long: `Delete a refresh schedule.
  
  Deletes an alert's refresh schedule. The refresh schedule specifies when to
  refresh and evaluate the associated query result.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Alerts.DeleteSchedule(ctx, deleteScheduleReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var getAlertReq dbsql.GetAlertRequest

func init() {
	Cmd.AddCommand(getAlertCmd)
	// TODO: short flags

	getAlertCmd.Flags().StringVar(&getAlertReq.AlertId, "alert-id", getAlertReq.AlertId, ``)

}

var getAlertCmd = &cobra.Command{
	Use:   "get-alert",
	Short: `Get an alert.`,
	Long: `Get an alert.
  
  Gets an alert.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Alerts.GetAlert(ctx, getAlertReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var getSubscriptionsReq dbsql.GetSubscriptionsRequest

func init() {
	Cmd.AddCommand(getSubscriptionsCmd)
	// TODO: short flags

	getSubscriptionsCmd.Flags().StringVar(&getSubscriptionsReq.AlertId, "alert-id", getSubscriptionsReq.AlertId, ``)

}

var getSubscriptionsCmd = &cobra.Command{
	Use:   "get-subscriptions",
	Short: `Get an alert's subscriptions.`,
	Long: `Get an alert's subscriptions.
  
  Get the subscriptions for an alert. An alert subscription represents exactly
  one recipient being notified whenever the alert is triggered. The alert
  recipient is specified by either the user field or the destination field.
  The user field is ignored if destination is non-null.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Alerts.GetSubscriptions(ctx, getSubscriptionsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

func init() {
	Cmd.AddCommand(listAlertsCmd)

}

var listAlertsCmd = &cobra.Command{
	Use:   "list-alerts",
	Short: `Get alerts.`,
	Long: `Get alerts.
  
  Gets a list of alerts.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Alerts.ListAlerts(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var listSchedulesReq dbsql.ListSchedulesRequest

func init() {
	Cmd.AddCommand(listSchedulesCmd)
	// TODO: short flags

	listSchedulesCmd.Flags().StringVar(&listSchedulesReq.AlertId, "alert-id", listSchedulesReq.AlertId, ``)

}

var listSchedulesCmd = &cobra.Command{
	Use:   "list-schedules",
	Short: `Get refresh schedules.`,
	Long: `Get refresh schedules.
  
  Gets the refresh schedules for the specified alert. Alerts can have refresh
  schedules that specify when to refresh and evaluate the associated query
  result.
  
  **Note:** Although refresh schedules are returned in a list, only one refresh
  schedule per alert is currently supported. The structure of refresh schedules
  is subject to change.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Alerts.ListSchedules(ctx, listSchedulesReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var subscribeReq dbsql.CreateSubscription

func init() {
	Cmd.AddCommand(subscribeCmd)
	// TODO: short flags

	subscribeCmd.Flags().StringVar(&subscribeReq.AlertId, "alert-id", subscribeReq.AlertId, `ID of the alert.`)
	subscribeCmd.Flags().StringVar(&subscribeReq.DestinationId, "destination-id", subscribeReq.DestinationId, `ID of the alert subscriber (if subscribing an alert destination).`)
	subscribeCmd.Flags().Int64Var(&subscribeReq.UserId, "user-id", subscribeReq.UserId, `ID of the alert subscriber (if subscribing a user).`)

}

var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: `Subscribe to an alert.`,
	Long:  `Subscribe to an alert.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Alerts.Subscribe(ctx, subscribeReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var unsubscribeReq dbsql.UnsubscribeRequest

func init() {
	Cmd.AddCommand(unsubscribeCmd)
	// TODO: short flags

	unsubscribeCmd.Flags().StringVar(&unsubscribeReq.AlertId, "alert-id", unsubscribeReq.AlertId, ``)
	unsubscribeCmd.Flags().StringVar(&unsubscribeReq.SubscriptionId, "subscription-id", unsubscribeReq.SubscriptionId, ``)

}

var unsubscribeCmd = &cobra.Command{
	Use:   "unsubscribe",
	Short: `Unsubscribe to an alert.`,
	Long: `Unsubscribe to an alert.
  
  Unsubscribes a user or a destination to an alert.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Alerts.Unsubscribe(ctx, unsubscribeReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var updateAlertReq dbsql.EditAlert

func init() {
	Cmd.AddCommand(updateAlertCmd)
	// TODO: short flags

	updateAlertCmd.Flags().StringVar(&updateAlertReq.AlertId, "alert-id", updateAlertReq.AlertId, ``)
	updateAlertCmd.Flags().StringVar(&updateAlertReq.Name, "name", updateAlertReq.Name, `Name of the alert.`)
	// TODO: complex arg: options
	updateAlertCmd.Flags().StringVar(&updateAlertReq.QueryId, "query-id", updateAlertReq.QueryId, `ID of the query evaluated by the alert.`)
	updateAlertCmd.Flags().IntVar(&updateAlertReq.Rearm, "rearm", updateAlertReq.Rearm, `Number of seconds after being triggered before the alert rearms itself and can be triggered again.`)

}

var updateAlertCmd = &cobra.Command{
	Use:   "update-alert",
	Short: `Update an alert.`,
	Long: `Update an alert.
  
  Updates an alert.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Alerts.UpdateAlert(ctx, updateAlertReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Alerts
