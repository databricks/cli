// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package alerts_v2

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	alertsv2 "github.com/databricks/cli/libs/sdktypes/alerts/v2"
	sdkalertsv2 "github.com/databricks/sdk-go/alerts/v2"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*alertsv2.CreateAlertRequest,
)

func newCreateAlert() *cobra.Command {
	cmd := &cobra.Command{}
	var createAlertReq alertsv2.CreateAlertRequest
	createAlertReq.Alert = &alertsv2.Alert{}

	var createAlertJson flags.JsonFlag

	cmd.Flags().Var(&createAlertJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createAlertReq.Alert.CustomDescription, "custom-description", `Custom description for the alert.`)
	flags.StringPtrVar(cmd.Flags(), &createAlertReq.Alert.CustomSummary, "custom-summary", `Custom summary for the alert.`)
	flags.EnumVar(cmd.Flags(), &createAlertReq.Alert.CustomTemplateFormat, "custom-template-format", `The format used to interpret the custom_summary and custom_description templates. Supported values: [HTML, MARKDOWN]`, "CustomTemplateFormat", "HTML", "MARKDOWN")
	// TODO: complex arg: effective_run_as
	// TODO: array: parameters
	flags.StringPtrVar(cmd.Flags(), &createAlertReq.Alert.ParentPath, "parent-path", `The workspace path of the folder containing the alert.`)
	// TODO: complex arg: run_as
	flags.StringPtrVar(cmd.Flags(), &createAlertReq.Alert.RunAsUserName, "run-as-user-name", `The run as username or application ID of service principal.`)

	cmd.Use = "create-alert DISPLAY_NAME QUERY_TEXT WAREHOUSE_ID EVALUATION SCHEDULE"
	cmd.Short = `Create an alert.`
	cmd.Long = `Create an alert.

  Create Alert

  Arguments:
    DISPLAY_NAME: The display name of the alert.
    QUERY_TEXT: Text of the query to be run.
    WAREHOUSE_ID: ID of the SQL warehouse attached to the alert.
    EVALUATION:
    SCHEDULE: `

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'display_name', 'query_text', 'warehouse_id', 'evaluation', 'schedule' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(5)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createAlertJson.Unmarshal(&createAlertReq.Alert)
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
			createAlertReq.Alert.DisplayName = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createAlertReq.Alert.QueryText = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createAlertReq.Alert.WarehouseId = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			_ = args[3] // TODO: complex arg: evaluation
		}
		if !cmd.Flags().Changed("json") {
			_ = args[4] // TODO: complex arg: schedule
		}

		sdkReq, err := alertsv2.CreateAlertRequestToSDK(&createAlertReq)
		if err != nil {
			return err
		}
		response, err := w.Alerts.CreateAlert(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := alertsv2.AlertFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkalertsv2.GetAlertRequest,
)

func newGetAlert() *cobra.Command {
	cmd := &cobra.Command{}
	var getAlertReq sdkalertsv2.GetAlertRequest

	cmd.Use = "get-alert ID"
	cmd.Short = `Get an alert.`
	cmd.Long = `Get an alert.

  Gets an alert.`

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

		getAlertReq.Id = new(args[0])

		sdkReq := &getAlertReq
		response, err := w.Alerts.GetAlert(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := alertsv2.AlertFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*alertsv2.ListAlertsRequest,
)

func newListAlerts() *cobra.Command {
	cmd := &cobra.Command{}
	var listAlertsReq alertsv2.ListAlertsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listAlertsLimit int

	flags.IntPtrVar(cmd.Flags(), &listAlertsReq.PageSize, "page-size", ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listAlertsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listAlertsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-alerts"
	cmd.Short = `List alerts.`
	cmd.Long = `List alerts.

  Gets a list of alerts accessible to the user, ordered by creation time.`

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

		sdkReq, err := alertsv2.ListAlertsRequestToSDK(&listAlertsReq)
		if err != nil {
			return err
		}
		if listAlertsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listAlertsLimit)
		}
		if listAlertsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listAlertsLimit)
		}
		response := w.Alerts.ListAlertsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, alertsv2.AlertFromSDK))
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
	*alertsv2.TrashAlertRequest,
)

func newTrashAlert() *cobra.Command {
	cmd := &cobra.Command{}
	var trashAlertReq alertsv2.TrashAlertRequest

	flags.BoolPtrVar(cmd.Flags(), &trashAlertReq.Purge, "purge", `Whether to permanently delete the alert.`)

	cmd.Use = "trash-alert ID"
	cmd.Short = `Delete an alert (legacy TrashAlert).`
	cmd.Long = `Delete an alert (legacy TrashAlert).

  Moves an alert to the trash. Trashed alerts immediately disappear from list
  views, and can no longer trigger. You can restore a trashed alert through the
  UI. A trashed alert is permanently deleted after 30 days.`

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

		trashAlertReq.Id = new(args[0])

		sdkReq, err := alertsv2.TrashAlertRequestToSDK(&trashAlertReq)
		if err != nil {
			return err
		}
		response, err := w.Alerts.TrashAlert(ctx, *sdkReq)
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
	*alertsv2.UpdateAlertRequest,
)

func newUpdateAlert() *cobra.Command {
	cmd := &cobra.Command{}
	var updateAlertReq alertsv2.UpdateAlertRequest
	updateAlertReq.Alert = &alertsv2.Alert{}

	var updateAlertJson flags.JsonFlag

	cmd.Flags().Var(&updateAlertJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateAlertReq.Alert.CustomDescription, "custom-description", `Custom description for the alert.`)
	flags.StringPtrVar(cmd.Flags(), &updateAlertReq.Alert.CustomSummary, "custom-summary", `Custom summary for the alert.`)
	flags.EnumVar(cmd.Flags(), &updateAlertReq.Alert.CustomTemplateFormat, "custom-template-format", `The format used to interpret the custom_summary and custom_description templates. Supported values: [HTML, MARKDOWN]`, "CustomTemplateFormat", "HTML", "MARKDOWN")
	// TODO: complex arg: effective_run_as
	// TODO: array: parameters
	flags.StringPtrVar(cmd.Flags(), &updateAlertReq.Alert.ParentPath, "parent-path", `The workspace path of the folder containing the alert.`)
	// TODO: complex arg: run_as
	flags.StringPtrVar(cmd.Flags(), &updateAlertReq.Alert.RunAsUserName, "run-as-user-name", `The run as username or application ID of service principal.`)

	cmd.Use = "update-alert ID UPDATE_MASK DISPLAY_NAME QUERY_TEXT WAREHOUSE_ID EVALUATION SCHEDULE"
	cmd.Short = `Update an alert.`
	cmd.Long = `Update an alert.

  Update alert

  Arguments:
    ID: The canonical identifier of the alert to retrieve information about.
    UPDATE_MASK: The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.

      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.
    DISPLAY_NAME: The display name of the alert.
    QUERY_TEXT: Text of the query to be run.
    WAREHOUSE_ID: ID of the SQL warehouse attached to the alert.
    EVALUATION:
    SCHEDULE: `

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only ID, UPDATE_MASK as positional arguments. Provide 'display_name', 'query_text', 'warehouse_id', 'evaluation', 'schedule' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(7)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateAlertJson.Unmarshal(&updateAlertReq.Alert)
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
		updateAlertReq.Alert.Id = new(args[0])
		updateAlertReq.UpdateMask = new(args[1])
		if !cmd.Flags().Changed("json") {
			updateAlertReq.Alert.DisplayName = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			updateAlertReq.Alert.QueryText = new(args[3])
		}
		if !cmd.Flags().Changed("json") {
			updateAlertReq.Alert.WarehouseId = new(args[4])
		}
		if !cmd.Flags().Changed("json") {
			_ = args[5] // TODO: complex arg: evaluation
		}
		if !cmd.Flags().Changed("json") {
			_ = args[6] // TODO: complex arg: schedule
		}

		sdkReq, err := alertsv2.UpdateAlertRequestToSDK(&updateAlertReq)
		if err != nil {
			return err
		}
		response, err := w.Alerts.UpdateAlert(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := alertsv2.AlertFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
