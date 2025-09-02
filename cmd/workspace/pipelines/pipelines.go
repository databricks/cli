// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package pipelines

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipelines",
		Short: `The Delta Live Tables API allows you to create, edit, delete, start, and view details about pipelines.`,
		Long: `The Delta Live Tables API allows you to create, edit, delete, start, and view
  details about pipelines.
  
  Delta Live Tables is a framework for building reliable, maintainable, and
  testable data processing pipelines. You define the transformations to perform
  on your data, and Delta Live Tables manages task orchestration, cluster
  management, monitoring, data quality, and error handling.
  
  Instead of defining your data pipelines using a series of separate Apache
  Spark tasks, Delta Live Tables manages how your data is transformed based on a
  target schema you define for each processing step. You can also enforce data
  quality with Delta Live Tables expectations. Expectations allow you to define
  expected data quality and specify how to handle records that fail those
  expectations.`,
		GroupID: "pipelines",
		Annotations: map[string]string{
			"package": "pipelines",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newGetUpdate())
	cmd.AddCommand(newListPipelineEvents())
	cmd.AddCommand(newListPipelines())
	cmd.AddCommand(newListUpdates())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newStartUpdate())
	cmd.AddCommand(newStop())
	cmd.AddCommand(newUpdate())
	cmd.AddCommand(newUpdatePermissions())

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
	*pipelines.CreatePipeline,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq pipelines.CreatePipeline
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create a pipeline.`
	cmd.Long = `Create a pipeline.
  
  Creates a new data processing pipeline based on the requested configuration.
  If successful, this method returns the ID of the new pipeline.`

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

		response, err := w.Pipelines.Create(ctx, createReq)
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
	*pipelines.DeletePipelineRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq pipelines.DeletePipelineRequest

	cmd.Use = "delete PIPELINE_ID"
	cmd.Short = `Delete a pipeline.`
	cmd.Long = `Delete a pipeline.
  
  Deletes a pipeline. Deleting a pipeline is a permanent action that stops and
  removes the pipeline and its tables. You cannot undo this action.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PIPELINE_ID argument specified. Loading names for Pipelines drop-down."
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Pipelines drop-down. Please manually specify required arguments. Original error: %w", err)
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
		deleteReq.PipelineId = args[0]

		err = w.Pipelines.Delete(ctx, deleteReq)
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
	*pipelines.GetPipelineRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq pipelines.GetPipelineRequest

	cmd.Use = "get PIPELINE_ID"
	cmd.Short = `Get a pipeline.`
	cmd.Long = `Get a pipeline.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PIPELINE_ID argument specified. Loading names for Pipelines drop-down."
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Pipelines drop-down. Please manually specify required arguments. Original error: %w", err)
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
		getReq.PipelineId = args[0]

		response, err := w.Pipelines.Get(ctx, getReq)
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

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*pipelines.GetPipelinePermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionLevelsReq pipelines.GetPipelinePermissionLevelsRequest

	cmd.Use = "get-permission-levels PIPELINE_ID"
	cmd.Short = `Get pipeline permission levels.`
	cmd.Long = `Get pipeline permission levels.
  
  Gets the permission levels that a user can have on an object.

  Arguments:
    PIPELINE_ID: The pipeline for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PIPELINE_ID argument specified. Loading names for Pipelines drop-down."
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Pipelines drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The pipeline for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the pipeline for which to get or manage permissions")
		}
		getPermissionLevelsReq.PipelineId = args[0]

		response, err := w.Pipelines.GetPermissionLevels(ctx, getPermissionLevelsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionLevelsOverrides {
		fn(cmd, &getPermissionLevelsReq)
	}

	return cmd
}

// start get-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionsOverrides []func(
	*cobra.Command,
	*pipelines.GetPipelinePermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionsReq pipelines.GetPipelinePermissionsRequest

	cmd.Use = "get-permissions PIPELINE_ID"
	cmd.Short = `Get pipeline permissions.`
	cmd.Long = `Get pipeline permissions.
  
  Gets the permissions of a pipeline. Pipelines can inherit permissions from
  their root object.

  Arguments:
    PIPELINE_ID: The pipeline for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PIPELINE_ID argument specified. Loading names for Pipelines drop-down."
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Pipelines drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The pipeline for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the pipeline for which to get or manage permissions")
		}
		getPermissionsReq.PipelineId = args[0]

		response, err := w.Pipelines.GetPermissions(ctx, getPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionsOverrides {
		fn(cmd, &getPermissionsReq)
	}

	return cmd
}

// start get-update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getUpdateOverrides []func(
	*cobra.Command,
	*pipelines.GetUpdateRequest,
)

func newGetUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var getUpdateReq pipelines.GetUpdateRequest

	cmd.Use = "get-update PIPELINE_ID UPDATE_ID"
	cmd.Short = `Get a pipeline update.`
	cmd.Long = `Get a pipeline update.
  
  Gets an update from an active pipeline.

  Arguments:
    PIPELINE_ID: The ID of the pipeline.
    UPDATE_ID: The ID of the update.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getUpdateReq.PipelineId = args[0]
		getUpdateReq.UpdateId = args[1]

		response, err := w.Pipelines.GetUpdate(ctx, getUpdateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getUpdateOverrides {
		fn(cmd, &getUpdateReq)
	}

	return cmd
}

// start list-pipeline-events command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listPipelineEventsOverrides []func(
	*cobra.Command,
	*pipelines.ListPipelineEventsRequest,
)

func newListPipelineEvents() *cobra.Command {
	cmd := &cobra.Command{}

	var listPipelineEventsReq pipelines.ListPipelineEventsRequest

	cmd.Flags().StringVar(&listPipelineEventsReq.Filter, "filter", listPipelineEventsReq.Filter, `Criteria to select a subset of results, expressed using a SQL-like syntax.`)
	cmd.Flags().IntVar(&listPipelineEventsReq.MaxResults, "max-results", listPipelineEventsReq.MaxResults, `Max number of entries to return in a single page.`)
	// TODO: array: order_by
	cmd.Flags().StringVar(&listPipelineEventsReq.PageToken, "page-token", listPipelineEventsReq.PageToken, `Page token returned by previous call.`)

	cmd.Use = "list-pipeline-events PIPELINE_ID"
	cmd.Short = `List pipeline events.`
	cmd.Long = `List pipeline events.
  
  Retrieves events for a pipeline.

  Arguments:
    PIPELINE_ID: The pipeline to return events for.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PIPELINE_ID argument specified. Loading names for Pipelines drop-down."
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Pipelines drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The pipeline to return events for")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the pipeline to return events for")
		}
		listPipelineEventsReq.PipelineId = args[0]

		response := w.Pipelines.ListPipelineEvents(ctx, listPipelineEventsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listPipelineEventsOverrides {
		fn(cmd, &listPipelineEventsReq)
	}

	return cmd
}

// start list-pipelines command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listPipelinesOverrides []func(
	*cobra.Command,
	*pipelines.ListPipelinesRequest,
)

func newListPipelines() *cobra.Command {
	cmd := &cobra.Command{}

	var listPipelinesReq pipelines.ListPipelinesRequest

	cmd.Flags().StringVar(&listPipelinesReq.Filter, "filter", listPipelinesReq.Filter, `Select a subset of results based on the specified criteria.`)
	cmd.Flags().IntVar(&listPipelinesReq.MaxResults, "max-results", listPipelinesReq.MaxResults, `The maximum number of entries to return in a single page.`)
	// TODO: array: order_by
	cmd.Flags().StringVar(&listPipelinesReq.PageToken, "page-token", listPipelinesReq.PageToken, `Page token returned by previous call.`)

	cmd.Use = "list-pipelines"
	cmd.Short = `List pipelines.`
	cmd.Long = `List pipelines.
  
  Lists pipelines defined in the Delta Live Tables system.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Pipelines.ListPipelines(ctx, listPipelinesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listPipelinesOverrides {
		fn(cmd, &listPipelinesReq)
	}

	return cmd
}

// start list-updates command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listUpdatesOverrides []func(
	*cobra.Command,
	*pipelines.ListUpdatesRequest,
)

func newListUpdates() *cobra.Command {
	cmd := &cobra.Command{}

	var listUpdatesReq pipelines.ListUpdatesRequest

	cmd.Flags().IntVar(&listUpdatesReq.MaxResults, "max-results", listUpdatesReq.MaxResults, `Max number of entries to return in a single page.`)
	cmd.Flags().StringVar(&listUpdatesReq.PageToken, "page-token", listUpdatesReq.PageToken, `Page token returned by previous call.`)
	cmd.Flags().StringVar(&listUpdatesReq.UntilUpdateId, "until-update-id", listUpdatesReq.UntilUpdateId, `If present, returns updates until and including this update_id.`)

	cmd.Use = "list-updates PIPELINE_ID"
	cmd.Short = `List pipeline updates.`
	cmd.Long = `List pipeline updates.
  
  List updates for an active pipeline.

  Arguments:
    PIPELINE_ID: The pipeline to return updates for.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PIPELINE_ID argument specified. Loading names for Pipelines drop-down."
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Pipelines drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The pipeline to return updates for")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the pipeline to return updates for")
		}
		listUpdatesReq.PipelineId = args[0]

		response, err := w.Pipelines.ListUpdates(ctx, listUpdatesReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listUpdatesOverrides {
		fn(cmd, &listUpdatesReq)
	}

	return cmd
}

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*pipelines.PipelinePermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var setPermissionsReq pipelines.PipelinePermissionsRequest
	var setPermissionsJson flags.JsonFlag

	cmd.Flags().Var(&setPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-permissions PIPELINE_ID"
	cmd.Short = `Set pipeline permissions.`
	cmd.Long = `Set pipeline permissions.
  
  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.

  Arguments:
    PIPELINE_ID: The pipeline for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := setPermissionsJson.Unmarshal(&setPermissionsReq)
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
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PIPELINE_ID argument specified. Loading names for Pipelines drop-down."
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Pipelines drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The pipeline for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the pipeline for which to get or manage permissions")
		}
		setPermissionsReq.PipelineId = args[0]

		response, err := w.Pipelines.SetPermissions(ctx, setPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setPermissionsOverrides {
		fn(cmd, &setPermissionsReq)
	}

	return cmd
}

// start start-update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var startUpdateOverrides []func(
	*cobra.Command,
	*pipelines.StartUpdate,
)

func newStartUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var startUpdateReq pipelines.StartUpdate
	var startUpdateJson flags.JsonFlag

	cmd.Flags().Var(&startUpdateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&startUpdateReq.Cause, "cause", `Supported values: [
  API_CALL,
  INFRASTRUCTURE_MAINTENANCE,
  JOB_TASK,
  RETRY_ON_FAILURE,
  SCHEMA_CHANGE,
  SERVICE_UPGRADE,
  USER_ACTION,
]`)
	cmd.Flags().BoolVar(&startUpdateReq.FullRefresh, "full-refresh", startUpdateReq.FullRefresh, `If true, this update will reset all tables before running.`)
	// TODO: array: full_refresh_selection
	// TODO: array: refresh_selection
	cmd.Flags().BoolVar(&startUpdateReq.ValidateOnly, "validate-only", startUpdateReq.ValidateOnly, `If true, this update only validates the correctness of pipeline source code but does not materialize or publish any datasets.`)

	cmd.Use = "start-update PIPELINE_ID"
	cmd.Short = `Start a pipeline.`
	cmd.Long = `Start a pipeline.
  
  Starts a new update for the pipeline. If there is already an active update for
  the pipeline, the request will fail and the active update will remain running.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := startUpdateJson.Unmarshal(&startUpdateReq)
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
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PIPELINE_ID argument specified. Loading names for Pipelines drop-down."
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Pipelines drop-down. Please manually specify required arguments. Original error: %w", err)
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
		startUpdateReq.PipelineId = args[0]

		response, err := w.Pipelines.StartUpdate(ctx, startUpdateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range startUpdateOverrides {
		fn(cmd, &startUpdateReq)
	}

	return cmd
}

// start stop command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var stopOverrides []func(
	*cobra.Command,
	*pipelines.StopRequest,
)

func newStop() *cobra.Command {
	cmd := &cobra.Command{}

	var stopReq pipelines.StopRequest

	var stopSkipWait bool
	var stopTimeout time.Duration

	cmd.Flags().BoolVar(&stopSkipWait, "no-wait", stopSkipWait, `do not wait to reach IDLE state`)
	cmd.Flags().DurationVar(&stopTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach IDLE state`)

	cmd.Use = "stop PIPELINE_ID"
	cmd.Short = `Stop a pipeline.`
	cmd.Long = `Stop a pipeline.
  
  Stops the pipeline by canceling the active update. If there is no active
  update for the pipeline, this request is a no-op.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PIPELINE_ID argument specified. Loading names for Pipelines drop-down."
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Pipelines drop-down. Please manually specify required arguments. Original error: %w", err)
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
		stopReq.PipelineId = args[0]

		wait, err := w.Pipelines.Stop(ctx, stopReq)
		if err != nil {
			return err
		}
		if stopSkipWait {
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *pipelines.GetPipelineResponse) {
			statusMessage := i.Cause
			spinner <- statusMessage
		}).GetWithTimeout(stopTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range stopOverrides {
		fn(cmd, &stopReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*pipelines.EditPipeline,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq pipelines.EditPipeline
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&updateReq.AllowDuplicateNames, "allow-duplicate-names", updateReq.AllowDuplicateNames, `If false, deployment will fail if name has changed and conflicts the name of another pipeline.`)
	cmd.Flags().StringVar(&updateReq.BudgetPolicyId, "budget-policy-id", updateReq.BudgetPolicyId, `Budget policy of this pipeline.`)
	cmd.Flags().StringVar(&updateReq.Catalog, "catalog", updateReq.Catalog, `A catalog in Unity Catalog to publish data from this pipeline to.`)
	cmd.Flags().StringVar(&updateReq.Channel, "channel", updateReq.Channel, `DLT Release Channel that specifies which version to use.`)
	// TODO: array: clusters
	// TODO: map via StringToStringVar: configuration
	cmd.Flags().BoolVar(&updateReq.Continuous, "continuous", updateReq.Continuous, `Whether the pipeline is continuous or triggered.`)
	// TODO: complex arg: deployment
	cmd.Flags().BoolVar(&updateReq.Development, "development", updateReq.Development, `Whether the pipeline is in Development mode.`)
	cmd.Flags().StringVar(&updateReq.Edition, "edition", updateReq.Edition, `Pipeline product edition.`)
	// TODO: complex arg: environment
	// TODO: complex arg: event_log
	cmd.Flags().Int64Var(&updateReq.ExpectedLastModified, "expected-last-modified", updateReq.ExpectedLastModified, `If present, the last-modified time of the pipeline settings before the edit.`)
	// TODO: complex arg: filters
	// TODO: complex arg: gateway_definition
	cmd.Flags().StringVar(&updateReq.Id, "id", updateReq.Id, `Unique identifier for this pipeline.`)
	// TODO: complex arg: ingestion_definition
	// TODO: array: libraries
	cmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Friendly identifier for this pipeline.`)
	// TODO: array: notifications
	cmd.Flags().BoolVar(&updateReq.Photon, "photon", updateReq.Photon, `Whether Photon is enabled for this pipeline.`)
	// TODO: complex arg: restart_window
	cmd.Flags().StringVar(&updateReq.RootPath, "root-path", updateReq.RootPath, `Root path for this pipeline.`)
	// TODO: complex arg: run_as
	cmd.Flags().StringVar(&updateReq.Schema, "schema", updateReq.Schema, `The default schema (database) where tables are read from or published to.`)
	cmd.Flags().BoolVar(&updateReq.Serverless, "serverless", updateReq.Serverless, `Whether serverless compute is enabled for this pipeline.`)
	cmd.Flags().StringVar(&updateReq.Storage, "storage", updateReq.Storage, `DBFS root directory for storing checkpoints and tables.`)
	// TODO: map via StringToStringVar: tags
	cmd.Flags().StringVar(&updateReq.Target, "target", updateReq.Target, `Target schema (database) to add tables in this pipeline to.`)
	// TODO: complex arg: trigger

	cmd.Use = "update PIPELINE_ID"
	cmd.Short = `Edit a pipeline.`
	cmd.Long = `Edit a pipeline.
  
  Updates a pipeline with the supplied configuration.

  Arguments:
    PIPELINE_ID: Unique identifier for this pipeline.`

	cmd.Annotations = make(map[string]string)

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
		}
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PIPELINE_ID argument specified. Loading names for Pipelines drop-down."
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Pipelines drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Unique identifier for this pipeline")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have unique identifier for this pipeline")
		}
		updateReq.PipelineId = args[0]

		err = w.Pipelines.Update(ctx, updateReq)
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

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*pipelines.PipelinePermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePermissionsReq pipelines.PipelinePermissionsRequest
	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update-permissions PIPELINE_ID"
	cmd.Short = `Update pipeline permissions.`
	cmd.Long = `Update pipeline permissions.
  
  Updates the permissions on a pipeline. Pipelines can inherit permissions from
  their root object.

  Arguments:
    PIPELINE_ID: The pipeline for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
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
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PIPELINE_ID argument specified. Loading names for Pipelines drop-down."
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Pipelines drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The pipeline for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the pipeline for which to get or manage permissions")
		}
		updatePermissionsReq.PipelineId = args[0]

		response, err := w.Pipelines.UpdatePermissions(ctx, updatePermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updatePermissionsOverrides {
		fn(cmd, &updatePermissionsReq)
	}

	return cmd
}

// end service Pipelines
