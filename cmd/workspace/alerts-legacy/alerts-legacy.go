// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package alerts_legacy

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
		Use:   "alerts-legacy",
		Short: `The alerts API can be used to perform CRUD operations on alerts.`,
		Long: `The alerts API can be used to perform CRUD operations on alerts. An alert is a
  Databricks SQL object that periodically runs a query, evaluates a condition of
  its result, and notifies one or more users and/or notification destinations if
  the condition was met. Alerts can be scheduled using the sql_task type of
  the Jobs API, e.g. :method:jobs/create.
  
  **Note**: A new version of the Databricks SQL API is now available. Please see
  the latest version. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`,
		GroupID: "sql",
		Annotations: map[string]string{
			"package": "sql",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newUpdate())

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
	*sql.CreateAlert,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq sql.CreateAlert
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Parent, "parent", createReq.Parent, `The identifier of the workspace folder containing the object.`)
	cmd.Flags().IntVar(&createReq.Rearm, "rearm", createReq.Rearm, `Number of seconds after being triggered before the alert rearms itself and can be triggered again.`)

	cmd.Use = "create"
	cmd.Short = `Create an alert.`
	cmd.Long = `Create an alert.
  
  Creates an alert. An alert is a Databricks SQL object that periodically runs a
  query, evaluates a condition of its result, and notifies users or notification
  destinations if the condition was met.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:alerts/create instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
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

		response, err := w.AlertsLegacy.Create(ctx, createReq)
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

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*sql.DeleteAlertsLegacyRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq sql.DeleteAlertsLegacyRequest

	cmd.Use = "delete ALERT_ID"
	cmd.Short = `Delete an alert.`
	cmd.Long = `Delete an alert.
  
  Deletes an alert. Deleted alerts are no longer accessible and cannot be
  restored. **Note**: Unlike queries and dashboards, alerts cannot be moved to
  the trash.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:alerts/delete instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.AlertId = args[0]

		err = w.AlertsLegacy.Delete(ctx, deleteReq)
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*sql.GetAlertsLegacyRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq sql.GetAlertsLegacyRequest

	cmd.Use = "get ALERT_ID"
	cmd.Short = `Get an alert.`
	cmd.Long = `Get an alert.
  
  Gets an alert.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:alerts/get instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.AlertId = args[0]

		response, err := w.AlertsLegacy.Get(ctx, getReq)
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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list"
	cmd.Short = `Get alerts.`
	cmd.Long = `Get alerts.
  
  Gets a list of alerts.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:alerts/list instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response, err := w.AlertsLegacy.List(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*sql.EditAlert,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq sql.EditAlert
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&updateReq.Rearm, "rearm", updateReq.Rearm, `Number of seconds after being triggered before the alert rearms itself and can be triggered again.`)

	cmd.Use = "update ALERT_ID"
	cmd.Short = `Update an alert.`
	cmd.Long = `Update an alert.
  
  Updates an alert.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:alerts/update instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

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
			diags := updateJson.Unmarshal(&updateReq)
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
		updateReq.AlertId = args[0]

		err = w.AlertsLegacy.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
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

// end service AlertsLegacy
