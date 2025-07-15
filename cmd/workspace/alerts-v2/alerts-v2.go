// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package alerts_v2

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
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
		Use:     "alerts-v2",
		Short:   `New version of SQL Alerts.`,
		Long:    `New version of SQL Alerts`,
		GroupID: "sql",
		Annotations: map[string]string{
			"package": "sql",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateAlert())
	cmd.AddCommand(newGetAlert())
	cmd.AddCommand(newListAlerts())
	cmd.AddCommand(newTrashAlert())
	cmd.AddCommand(newUpdateAlert())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-alert command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createAlertOverrides []func(
	*cobra.Command,
	*sql.CreateAlertV2Request,
)

func newCreateAlert() *cobra.Command {
	cmd := &cobra.Command{}

	var createAlertReq sql.CreateAlertV2Request
	createAlertReq.Alert = sql.AlertV2{}
	var createAlertJson flags.JsonFlag

	cmd.Flags().Var(&createAlertJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createAlertReq.Alert.CustomDescription, "custom-description", createAlertReq.Alert.CustomDescription, `Custom description for the alert.`)
	cmd.Flags().StringVar(&createAlertReq.Alert.CustomSummary, "custom-summary", createAlertReq.Alert.CustomSummary, `Custom summary for the alert.`)
	cmd.Flags().StringVar(&createAlertReq.Alert.DisplayName, "display-name", createAlertReq.Alert.DisplayName, `The display name of the alert.`)
	// TODO: complex arg: evaluation
	cmd.Flags().StringVar(&createAlertReq.Alert.ParentPath, "parent-path", createAlertReq.Alert.ParentPath, `The workspace path of the folder containing the alert.`)
	cmd.Flags().StringVar(&createAlertReq.Alert.QueryText, "query-text", createAlertReq.Alert.QueryText, `Text of the query to be run.`)
	cmd.Flags().StringVar(&createAlertReq.Alert.RunAsUserName, "run-as-user-name", createAlertReq.Alert.RunAsUserName, `The run as username or application ID of service principal.`)
	// TODO: complex arg: schedule
	cmd.Flags().StringVar(&createAlertReq.Alert.WarehouseId, "warehouse-id", createAlertReq.Alert.WarehouseId, `ID of the SQL warehouse attached to the alert.`)

	cmd.Use = "create-alert"
	cmd.Short = `Create an alert.`
	cmd.Long = `Create an alert.
  
  Create Alert`

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
			diags := createAlertJson.Unmarshal(&createAlertReq.Alert)
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

		response, err := w.AlertsV2.CreateAlert(ctx, createAlertReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createAlertOverrides {
		fn(cmd, &createAlertReq)
	}

	return cmd
}

// start get-alert command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getAlertOverrides []func(
	*cobra.Command,
	*sql.GetAlertV2Request,
)

func newGetAlert() *cobra.Command {
	cmd := &cobra.Command{}

	var getAlertReq sql.GetAlertV2Request

	cmd.Use = "get-alert ID"
	cmd.Short = `Get an alert.`
	cmd.Long = `Get an alert.
  
  Gets an alert.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No ID argument specified. Loading names for Alerts V2 drop-down."
			names, err := w.AlertsV2.AlertV2DisplayNameToIdMap(ctx, sql.ListAlertsV2Request{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Alerts V2 drop-down. Please manually specify required arguments. Original error: %w", err)
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
		getAlertReq.Id = args[0]

		response, err := w.AlertsV2.GetAlert(ctx, getAlertReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getAlertOverrides {
		fn(cmd, &getAlertReq)
	}

	return cmd
}

// start list-alerts command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listAlertsOverrides []func(
	*cobra.Command,
	*sql.ListAlertsV2Request,
)

func newListAlerts() *cobra.Command {
	cmd := &cobra.Command{}

	var listAlertsReq sql.ListAlertsV2Request

	cmd.Flags().IntVar(&listAlertsReq.PageSize, "page-size", listAlertsReq.PageSize, ``)
	cmd.Flags().StringVar(&listAlertsReq.PageToken, "page-token", listAlertsReq.PageToken, ``)

	cmd.Use = "list-alerts"
	cmd.Short = `List alerts.`
	cmd.Long = `List alerts.
  
  Gets a list of alerts accessible to the user, ordered by creation time.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.AlertsV2.ListAlerts(ctx, listAlertsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listAlertsOverrides {
		fn(cmd, &listAlertsReq)
	}

	return cmd
}

// start trash-alert command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var trashAlertOverrides []func(
	*cobra.Command,
	*sql.TrashAlertV2Request,
)

func newTrashAlert() *cobra.Command {
	cmd := &cobra.Command{}

	var trashAlertReq sql.TrashAlertV2Request

	cmd.Use = "trash-alert ID"
	cmd.Short = `Delete an alert.`
	cmd.Long = `Delete an alert.
  
  Moves an alert to the trash. Trashed alerts immediately disappear from list
  views, and can no longer trigger. You can restore a trashed alert through the
  UI. A trashed alert is permanently deleted after 30 days.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No ID argument specified. Loading names for Alerts V2 drop-down."
			names, err := w.AlertsV2.AlertV2DisplayNameToIdMap(ctx, sql.ListAlertsV2Request{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Alerts V2 drop-down. Please manually specify required arguments. Original error: %w", err)
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
		trashAlertReq.Id = args[0]

		err = w.AlertsV2.TrashAlert(ctx, trashAlertReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range trashAlertOverrides {
		fn(cmd, &trashAlertReq)
	}

	return cmd
}

// start update-alert command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateAlertOverrides []func(
	*cobra.Command,
	*sql.UpdateAlertV2Request,
)

func newUpdateAlert() *cobra.Command {
	cmd := &cobra.Command{}

	var updateAlertReq sql.UpdateAlertV2Request
	updateAlertReq.Alert = sql.AlertV2{}
	var updateAlertJson flags.JsonFlag

	cmd.Flags().Var(&updateAlertJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateAlertReq.Alert.CustomDescription, "custom-description", updateAlertReq.Alert.CustomDescription, `Custom description for the alert.`)
	cmd.Flags().StringVar(&updateAlertReq.Alert.CustomSummary, "custom-summary", updateAlertReq.Alert.CustomSummary, `Custom summary for the alert.`)
	cmd.Flags().StringVar(&updateAlertReq.Alert.DisplayName, "display-name", updateAlertReq.Alert.DisplayName, `The display name of the alert.`)
	// TODO: complex arg: evaluation
	cmd.Flags().StringVar(&updateAlertReq.Alert.ParentPath, "parent-path", updateAlertReq.Alert.ParentPath, `The workspace path of the folder containing the alert.`)
	cmd.Flags().StringVar(&updateAlertReq.Alert.QueryText, "query-text", updateAlertReq.Alert.QueryText, `Text of the query to be run.`)
	cmd.Flags().StringVar(&updateAlertReq.Alert.RunAsUserName, "run-as-user-name", updateAlertReq.Alert.RunAsUserName, `The run as username or application ID of service principal.`)
	// TODO: complex arg: schedule
	cmd.Flags().StringVar(&updateAlertReq.Alert.WarehouseId, "warehouse-id", updateAlertReq.Alert.WarehouseId, `ID of the SQL warehouse attached to the alert.`)

	cmd.Use = "update-alert ID"
	cmd.Short = `Update an alert.`
	cmd.Long = `Update an alert.
  
  Update alert

  Arguments:
    ID: UUID identifying the alert.`

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
			diags := updateAlertJson.Unmarshal(&updateAlertReq.Alert)
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
		updateAlertReq.Id = args[0]

		response, err := w.AlertsV2.UpdateAlert(ctx, updateAlertReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateAlertOverrides {
		fn(cmd, &updateAlertReq)
	}

	return cmd
}

// end service AlertsV2
