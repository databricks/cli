// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package lakeview

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
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
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetPublished())
	cmd.AddCommand(newPublish())
	cmd.AddCommand(newTrash())
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
	*dashboards.CreateDashboardRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq dashboards.CreateDashboardRequest
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.ParentPath, "parent-path", createReq.ParentPath, `The workspace path of the folder containing the dashboard.`)
	cmd.Flags().StringVar(&createReq.SerializedDashboard, "serialized-dashboard", createReq.SerializedDashboard, `The contents of the dashboard in serialized string form.`)
	cmd.Flags().StringVar(&createReq.WarehouseId, "warehouse-id", createReq.WarehouseId, `The warehouse ID used to run the dashboard.`)

	cmd.Use = "create DISPLAY_NAME"
	cmd.Short = `Create dashboard.`
	cmd.Long = `Create dashboard.
  
  Create a draft dashboard.

  Arguments:
    DISPLAY_NAME: The display name of the dashboard.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'display_name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			createReq.DisplayName = args[0]
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*dashboards.GetLakeviewRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq dashboards.GetLakeviewRequest

	// TODO: short flags

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
		w := root.WorkspaceClient(ctx)

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
	*dashboards.GetPublishedRequest,
)

func newGetPublished() *cobra.Command {
	cmd := &cobra.Command{}

	var getPublishedReq dashboards.GetPublishedRequest

	// TODO: short flags

	cmd.Use = "get-published DASHBOARD_ID"
	cmd.Short = `Get published dashboard.`
	cmd.Long = `Get published dashboard.
  
  Get the current published dashboard.

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
		w := root.WorkspaceClient(ctx)

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

	// TODO: short flags
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
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = publishJson.Unmarshal(&publishReq)
			if err != nil {
				return err
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
	*dashboards.TrashRequest,
)

func newTrash() *cobra.Command {
	cmd := &cobra.Command{}

	var trashReq dashboards.TrashRequest

	// TODO: short flags

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
		w := root.WorkspaceClient(ctx)

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
	var updateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.DisplayName, "display-name", updateReq.DisplayName, `The display name of the dashboard.`)
	cmd.Flags().StringVar(&updateReq.Etag, "etag", updateReq.Etag, `The etag for the dashboard.`)
	cmd.Flags().StringVar(&updateReq.SerializedDashboard, "serialized-dashboard", updateReq.SerializedDashboard, `The contents of the dashboard in serialized string form.`)
	cmd.Flags().StringVar(&updateReq.WarehouseId, "warehouse-id", updateReq.WarehouseId, `The warehouse ID used to run the dashboard.`)

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
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
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

// end service Lakeview
