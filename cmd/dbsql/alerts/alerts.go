package alerts

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/dbsql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "alerts",
	Short: `The alerts API can be used to perform CRUD operations on alerts.`,
}

var createAlertReq dbsql.EditAlert

func init() {
	Cmd.AddCommand(createAlertCmd)
	// TODO: short flags

	createAlertCmd.Flags().StringVar(&createAlertReq.AlertId, "alert-id", "", ``)
	createAlertCmd.Flags().StringVar(&createAlertReq.Name, "name", "", `Name of the alert.`)
	// TODO: complex arg: options
	createAlertCmd.Flags().StringVar(&createAlertReq.QueryId, "query-id", "", `ID of the query evaluated by the alert.`)
	createAlertCmd.Flags().IntVar(&createAlertReq.Rearm, "rearm", 0, `Number of seconds after being triggered before the alert rearms itself and can be triggered again.`)

}

var createAlertCmd = &cobra.Command{
	Use:   "create-alert",
	Short: `Create an alert.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Alerts.CreateAlert(ctx, createAlertReq)
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

var createScheduleReq dbsql.CreateRefreshSchedule

func init() {
	Cmd.AddCommand(createScheduleCmd)
	// TODO: short flags

	createScheduleCmd.Flags().StringVar(&createScheduleReq.AlertId, "alert-id", "", ``)
	createScheduleCmd.Flags().StringVar(&createScheduleReq.Cron, "cron", "", `Cron string representing the refresh schedule.`)
	createScheduleCmd.Flags().StringVar(&createScheduleReq.DataSourceId, "data-source-id", "", `ID of the SQL warehouse to refresh with.`)

}

var createScheduleCmd = &cobra.Command{
	Use:   "create-schedule",
	Short: `Create a refresh schedule.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Alerts.CreateSchedule(ctx, createScheduleReq)
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

var deleteAlertReq dbsql.DeleteAlertRequest

func init() {
	Cmd.AddCommand(deleteAlertCmd)
	// TODO: short flags

	deleteAlertCmd.Flags().StringVar(&deleteAlertReq.AlertId, "alert-id", "", ``)

}

var deleteAlertCmd = &cobra.Command{
	Use:   "delete-alert",
	Short: `Delete an alert.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	deleteScheduleCmd.Flags().StringVar(&deleteScheduleReq.AlertId, "alert-id", "", ``)
	deleteScheduleCmd.Flags().StringVar(&deleteScheduleReq.ScheduleId, "schedule-id", "", ``)

}

var deleteScheduleCmd = &cobra.Command{
	Use:   "delete-schedule",
	Short: `Delete a refresh schedule.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	getAlertCmd.Flags().StringVar(&getAlertReq.AlertId, "alert-id", "", ``)

}

var getAlertCmd = &cobra.Command{
	Use:   "get-alert",
	Short: `Get an alert.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Alerts.GetAlert(ctx, getAlertReq)
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

var getSubscriptionsReq dbsql.GetSubscriptionsRequest

func init() {
	Cmd.AddCommand(getSubscriptionsCmd)
	// TODO: short flags

	getSubscriptionsCmd.Flags().StringVar(&getSubscriptionsReq.AlertId, "alert-id", "", ``)

}

var getSubscriptionsCmd = &cobra.Command{
	Use:   "get-subscriptions",
	Short: `Get an alert's subscriptions.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Alerts.GetSubscriptions(ctx, getSubscriptionsReq)
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

func init() {
	Cmd.AddCommand(listAlertsCmd)

}

var listAlertsCmd = &cobra.Command{
	Use:   "list-alerts",
	Short: `Get alerts.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Alerts.ListAlerts(ctx)
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

var listSchedulesReq dbsql.ListSchedulesRequest

func init() {
	Cmd.AddCommand(listSchedulesCmd)
	// TODO: short flags

	listSchedulesCmd.Flags().StringVar(&listSchedulesReq.AlertId, "alert-id", "", ``)

}

var listSchedulesCmd = &cobra.Command{
	Use:   "list-schedules",
	Short: `Get refresh schedules.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Alerts.ListSchedules(ctx, listSchedulesReq)
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

var subscribeReq dbsql.CreateSubscription

func init() {
	Cmd.AddCommand(subscribeCmd)
	// TODO: short flags

	subscribeCmd.Flags().StringVar(&subscribeReq.AlertId, "alert-id", "", `ID of the alert.`)
	subscribeCmd.Flags().StringVar(&subscribeReq.DestinationId, "destination-id", "", `ID of the alert subscriber (if subscribing an alert destination).`)
	subscribeCmd.Flags().Int64Var(&subscribeReq.UserId, "user-id", 0, `ID of the alert subscriber (if subscribing a user).`)

}

var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: `Subscribe to an alert.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Alerts.Subscribe(ctx, subscribeReq)
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

var unsubscribeReq dbsql.UnsubscribeRequest

func init() {
	Cmd.AddCommand(unsubscribeCmd)
	// TODO: short flags

	unsubscribeCmd.Flags().StringVar(&unsubscribeReq.AlertId, "alert-id", "", ``)
	unsubscribeCmd.Flags().StringVar(&unsubscribeReq.SubscriptionId, "subscription-id", "", ``)

}

var unsubscribeCmd = &cobra.Command{
	Use:   "unsubscribe",
	Short: `Unsubscribe to an alert.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	updateAlertCmd.Flags().StringVar(&updateAlertReq.AlertId, "alert-id", "", ``)
	updateAlertCmd.Flags().StringVar(&updateAlertReq.Name, "name", "", `Name of the alert.`)
	// TODO: complex arg: options
	updateAlertCmd.Flags().StringVar(&updateAlertReq.QueryId, "query-id", "", `ID of the query evaluated by the alert.`)
	updateAlertCmd.Flags().IntVar(&updateAlertReq.Rearm, "rearm", 0, `Number of seconds after being triggered before the alert rearms itself and can be triggered again.`)

}

var updateAlertCmd = &cobra.Command{
	Use:   "update-alert",
	Short: `Update an alert.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Alerts.UpdateAlert(ctx, updateAlertReq)
		if err != nil {
			return err
		}

		return nil
	},
}
