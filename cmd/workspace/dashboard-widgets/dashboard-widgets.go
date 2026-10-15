// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package dashboard_widgets

import (
	"errors"
	"fmt"

	"maps"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard-widgets",
		Short: `This is an evolving API that facilitates the addition and removal of widgets from existing dashboards within the Databricks Workspace.`,
		Long: `This is an evolving API that facilitates the addition and removal of widgets
  from existing dashboards within the Databricks Workspace. Data structures may
  change over time.`,
		GroupID: "sql",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
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
	*map[string]any,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq map[string]any
	createReq = map[string]any{}
	var createDashboardIdRaw string
	var createTextRaw string
	var createVisualizationIdRaw string
	var createWidthRaw int

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createTextRaw, "text", createTextRaw, `If this is a textbox widget, the application displays this text.`)
	cmd.Flags().StringVar(&createVisualizationIdRaw, "visualization-id", createVisualizationIdRaw, `Query Visualization ID returned by :method:queryvisualizations/create.`)

	cmd.Use = "create"
	cmd.Short = `Add widget to a dashboard.`
	cmd.Long = `Add widget to a dashboard.

  Adds a widget to a dashboard`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("dashboard-id") || createDashboardIdRaw != "" {
			createReq["dashboard_id"] = createDashboardIdRaw
		}
		if cmd.Flags().Changed("text") || createTextRaw != "" {
			createReq["text"] = createTextRaw
		}
		if cmd.Flags().Changed("visualization-id") || createVisualizationIdRaw != "" {
			createReq["visualization_id"] = createVisualizationIdRaw
		}
		if cmd.Flags().Changed("width") || createWidthRaw != 0 {
			createReq["width"] = createWidthRaw
		}
		rawPath := "/api/2.0/preview/sql/widgets"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, createReq, &response)
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
	*map[string]any,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq map[string]any
	deleteReq = map[string]any{}
	var deleteIdRaw string

	cmd.Use = "delete ID"
	cmd.Short = `Remove widget.`
	cmd.Long = `Remove widget.

  Removes a widget from a dashboard

  Arguments:
    ID: Widget ID returned by :method:dashboardwidgets/create`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/widgets/%s", fmt.Sprint(deleteIdRaw))

		maps.Copy(rawQuery, deleteReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, nil)
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

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq map[string]any
	updateReq = map[string]any{}
	var updateDashboardIdRaw string
	var updateIdRaw string
	var updateTextRaw string
	var updateVisualizationIdRaw string
	var updateWidthRaw int

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateTextRaw, "text", updateTextRaw, `If this is a textbox widget, the application displays this text.`)
	cmd.Flags().StringVar(&updateVisualizationIdRaw, "visualization-id", updateVisualizationIdRaw, `Query Visualization ID returned by :method:queryvisualizations/create.`)

	cmd.Use = "update ID"
	cmd.Short = `Update existing widget.`
	cmd.Long = `Update existing widget.

  Updates an existing widget

  Arguments:
    ID: Widget ID returned by :method:dashboardwidgets/create`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

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
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}
		updateIdRaw = args[0]

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("dashboard-id") || updateDashboardIdRaw != "" {
			updateReq["dashboard_id"] = updateDashboardIdRaw
		}
		if cmd.Flags().Changed("text") || updateTextRaw != "" {
			updateReq["text"] = updateTextRaw
		}
		if cmd.Flags().Changed("visualization-id") || updateVisualizationIdRaw != "" {
			updateReq["visualization_id"] = updateVisualizationIdRaw
		}
		if cmd.Flags().Changed("width") || updateWidthRaw != 0 {
			updateReq["width"] = updateWidthRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/widgets/%s", fmt.Sprint(updateIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, updateReq, &response)
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

// end service DashboardWidgets
