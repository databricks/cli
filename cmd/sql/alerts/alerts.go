// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package alerts

import (
	"fmt"

	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/sql"
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

// start create command

var createReq sql.EditAlert
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().IntVar(&createReq.Rearm, "rearm", createReq.Rearm, `Number of seconds after being triggered before the alert rearms itself and can be triggered again.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create an alert.`,
	Long: `Create an alert.
  
  Creates an alert. An alert is a Databricks SQL object that periodically runs a
  query, evaluates a condition of its result, and notifies users or alert
  destinations if the condition was met.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		createReq.Name = args[0]
		_, err = fmt.Sscan(args[1], &createReq.Options)
		if err != nil {
			return fmt.Errorf("invalid OPTIONS: %s", args[1])
		}
		createReq.QueryId = args[2]
		createReq.AlertId = args[3]

		response, err := w.Alerts.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start create-schedule command

var createScheduleReq sql.CreateRefreshSchedule

func init() {
	Cmd.AddCommand(createScheduleCmd)
	// TODO: short flags

	createScheduleCmd.Flags().StringVar(&createScheduleReq.DataSourceId, "data-source-id", createScheduleReq.DataSourceId, `ID of the SQL warehouse to refresh with.`)

}

var createScheduleCmd = &cobra.Command{
	Use:   "create-schedule CRON ALERT_ID",
	Short: `Create a refresh schedule.`,
	Long: `Create a refresh schedule.
  
  Creates a new refresh schedule for an alert.
  
  **Note:** The structure of refresh schedules is subject to change.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		createScheduleReq.Cron = args[0]
		createScheduleReq.AlertId = args[1]

		response, err := w.Alerts.CreateSchedule(ctx, createScheduleReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq sql.DeleteAlertRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete ALERT_ID",
	Short: `Delete an alert.`,
	Long: `Delete an alert.
  
  Deletes an alert. Deleted alerts are no longer accessible and cannot be
  restored. **Note:** Unlike queries and dashboards, alerts cannot be moved to
  the trash.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Alerts.AlertNameToIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		deleteReq.AlertId = args[0]

		err = w.Alerts.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start delete-schedule command

var deleteScheduleReq sql.DeleteScheduleRequest

func init() {
	Cmd.AddCommand(deleteScheduleCmd)
	// TODO: short flags

}

var deleteScheduleCmd = &cobra.Command{
	Use:   "delete-schedule ALERT_ID SCHEDULE_ID",
	Short: `Delete a refresh schedule.`,
	Long: `Delete a refresh schedule.
  
  Deletes an alert's refresh schedule. The refresh schedule specifies when to
  refresh and evaluate the associated query result.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		deleteScheduleReq.AlertId = args[0]
		deleteScheduleReq.ScheduleId = args[1]

		err = w.Alerts.DeleteSchedule(ctx, deleteScheduleReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq sql.GetAlertRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get ALERT_ID",
	Short: `Get an alert.`,
	Long: `Get an alert.
  
  Gets an alert.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Alerts.AlertNameToIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		getReq.AlertId = args[0]

		response, err := w.Alerts.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start get-subscriptions command

var getSubscriptionsReq sql.GetSubscriptionsRequest

func init() {
	Cmd.AddCommand(getSubscriptionsCmd)
	// TODO: short flags

}

var getSubscriptionsCmd = &cobra.Command{
	Use:   "get-subscriptions ALERT_ID",
	Short: `Get an alert's subscriptions.`,
	Long: `Get an alert's subscriptions.
  
  Get the subscriptions for an alert. An alert subscription represents exactly
  one recipient being notified whenever the alert is triggered. The alert
  recipient is specified by either the user field or the destination field.
  The user field is ignored if destination is non-null.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Alerts.AlertNameToIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		getSubscriptionsReq.AlertId = args[0]

		response, err := w.Alerts.GetSubscriptions(ctx, getSubscriptionsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get alerts.`,
	Long: `Get alerts.
  
  Gets a list of alerts.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Alerts.List(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list-schedules command

var listSchedulesReq sql.ListSchedulesRequest

func init() {
	Cmd.AddCommand(listSchedulesCmd)
	// TODO: short flags

}

var listSchedulesCmd = &cobra.Command{
	Use:   "list-schedules ALERT_ID",
	Short: `Get refresh schedules.`,
	Long: `Get refresh schedules.
  
  Gets the refresh schedules for the specified alert. Alerts can have refresh
  schedules that specify when to refresh and evaluate the associated query
  result.
  
  **Note:** Although refresh schedules are returned in a list, only one refresh
  schedule per alert is currently supported. The structure of refresh schedules
  is subject to change.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Alerts.AlertNameToIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		listSchedulesReq.AlertId = args[0]

		response, err := w.Alerts.ListSchedules(ctx, listSchedulesReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start subscribe command

var subscribeReq sql.CreateSubscription

func init() {
	Cmd.AddCommand(subscribeCmd)
	// TODO: short flags

	subscribeCmd.Flags().StringVar(&subscribeReq.DestinationId, "destination-id", subscribeReq.DestinationId, `ID of the alert subscriber (if subscribing an alert destination).`)
	subscribeCmd.Flags().Int64Var(&subscribeReq.UserId, "user-id", subscribeReq.UserId, `ID of the alert subscriber (if subscribing a user).`)

}

var subscribeCmd = &cobra.Command{
	Use:   "subscribe ALERT_ID",
	Short: `Subscribe to an alert.`,
	Long:  `Subscribe to an alert.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Alerts.AlertNameToIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "ID of the alert")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have id of the alert")
		}
		subscribeReq.AlertId = args[0]

		response, err := w.Alerts.Subscribe(ctx, subscribeReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start unsubscribe command

var unsubscribeReq sql.UnsubscribeRequest

func init() {
	Cmd.AddCommand(unsubscribeCmd)
	// TODO: short flags

}

var unsubscribeCmd = &cobra.Command{
	Use:   "unsubscribe ALERT_ID SUBSCRIPTION_ID",
	Short: `Unsubscribe to an alert.`,
	Long: `Unsubscribe to an alert.
  
  Unsubscribes a user or a destination to an alert.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		unsubscribeReq.AlertId = args[0]
		unsubscribeReq.SubscriptionId = args[1]

		err = w.Alerts.Unsubscribe(ctx, unsubscribeReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq sql.EditAlert
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().IntVar(&updateReq.Rearm, "rearm", updateReq.Rearm, `Number of seconds after being triggered before the alert rearms itself and can be triggered again.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update an alert.`,
	Long: `Update an alert.
  
  Updates an alert.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		updateReq.Name = args[0]
		_, err = fmt.Sscan(args[1], &updateReq.Options)
		if err != nil {
			return fmt.Errorf("invalid OPTIONS: %s", args[1])
		}
		updateReq.QueryId = args[2]
		updateReq.AlertId = args[3]

		err = w.Alerts.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Alerts
