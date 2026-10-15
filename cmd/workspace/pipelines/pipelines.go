// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package pipelines

import (
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/sdk-go/options/lro"
	"github.com/spf13/cobra"

	accessmanagement "github.com/databricks/cli/libs/sdktypes/accessmanagement/v1"
	pipelines "github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	sdkaccessmanagement "github.com/databricks/sdk-go/accessmanagement/v1"
	sdkpipelines "github.com/databricks/sdk-go/pipelines/v2"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipelines",
		Short: `The Lakeflow Spark Declarative Pipelines API allows you to create, edit, delete, start, and view details about pipelines.`,
		Long: `The Lakeflow Spark Declarative Pipelines API allows you to create, edit,
  delete, start, and view details about pipelines.

  Spark Declarative Pipelines is a framework for building reliable,
  maintainable, and testable data processing pipelines. You define the
  transformations to perform on your data, and Spark Declarative Pipelines
  manages task orchestration, cluster management, monitoring, data quality, and
  error handling.

  Instead of defining your data pipelines using a series of separate Apache
  Spark tasks, Spark Declarative Pipelines manages how your data is transformed
  based on a target schema you define for each processing step. You can also
  enforce data quality with Spark Declarative Pipelines expectations.
  Expectations allow you to define expected data quality and specify how to
  handle records that fail those expectations.`,
		GroupID: "lakeflow",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newApplyEnvironment())
	cmd.AddCommand(newClone())
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

// start apply-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var applyEnvironmentOverrides []func(
	*cobra.Command,
	*pipelines.ApplyEnvironmentRequest,
)

func newApplyEnvironment() *cobra.Command {
	cmd := &cobra.Command{}
	var applyEnvironmentReq pipelines.ApplyEnvironmentRequest

	cmd.Use = "apply-environment PIPELINE_ID"
	cmd.Short = `*Public Preview* Apply the latest environment to the pipeline.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Apply the latest environment to the pipeline.

  Applies the current pipeline environment onto the pipeline compute. The
  environment applied can be used by subsequent dev-mode updates.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		applyEnvironmentReq.PipelineId = new(args[0])

		sdkReq, err := pipelines.ApplyEnvironmentRequestToSDK(&applyEnvironmentReq)
		if err != nil {
			return err
		}
		response, err := w.Pipelines.ApplyEnvironment(ctx, *sdkReq)
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
	for _, fn := range applyEnvironmentOverrides {
		fn(cmd, &applyEnvironmentReq)
	}

	return cmd
}

// start clone command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cloneOverrides []func(
	*cobra.Command,
	*pipelines.ClonePipelineRequest,
)

func newClone() *cobra.Command {
	cmd := &cobra.Command{}
	var cloneReq pipelines.ClonePipelineRequest

	var cloneJson flags.JsonFlag

	cmd.Flags().Var(&cloneJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "clone PIPELINE_ID"
	cmd.Short = `Clone a pipeline.`
	cmd.Long = `Clone a pipeline.

  Creates a new pipeline using Unity Catalog from a pipeline using Hive
  Metastore. This method returns the ID of the newly created clone.
  Additionally, this method starts an update for the newly created pipeline.

  Arguments:
    PIPELINE_ID: Source pipeline to clone from`

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
			diags := cloneJson.Unmarshal(&cloneReq)
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
		cloneReq.PipelineId = new(args[0])

		sdkReq, err := pipelines.ClonePipelineRequestToSDK(&cloneReq)
		if err != nil {
			return err
		}
		response, err := w.Pipelines.Clone(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := pipelines.ClonePipelineResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range cloneOverrides {
		fn(cmd, &cloneReq)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*pipelines.CreatePipelineRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq pipelines.CreatePipelineRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create a pipeline.`
	cmd.Long = `Create a pipeline.

  Creates a new data processing pipeline based on the requested configuration.
  If successful, this method returns the ID of the new pipeline.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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

		sdkReq, err := pipelines.CreatePipelineRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Pipelines.Create(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := pipelines.CreatePipelineResponseFromSDK(response)
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

	flags.BoolPtrVar(cmd.Flags(), &deleteReq.Cascade, "cascade", `If false, pipeline deletion will not cascade to its datasets (MVs, STs, Views).`)
	flags.BoolPtrVar(cmd.Flags(), &deleteReq.Force, "force", `If true, deletion will proceed even if resource cleanup fails.`)

	cmd.Use = "delete PIPELINE_ID"
	cmd.Short = `Delete a pipeline.`
	cmd.Long = `Delete a pipeline.

  Deletes a pipeline. If the pipeline publishes to Unity Catalog, pipeline
  deletion will cascade to all pipeline tables. Please reach out to Databricks
  support for assistance to undo this action.`

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

		deleteReq.PipelineId = new(args[0])

		sdkReq, err := pipelines.DeletePipelineRequestToSDK(&deleteReq)
		if err != nil {
			return err
		}
		response, err := w.Pipelines.Delete(ctx, *sdkReq)
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
	*sdkpipelines.GetPipelineRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkpipelines.GetPipelineRequest

	cmd.Use = "get PIPELINE_ID"
	cmd.Short = `Get a pipeline.`
	cmd.Long = `Get a pipeline.`

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

		getReq.PipelineId = new(args[0])

		sdkReq := &getReq
		response, err := w.Pipelines.Get(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := pipelines.GetPipelineResponseFromSDK(response)
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

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*sdkaccessmanagement.ListPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}
	var getPermissionLevelsReq sdkaccessmanagement.ListPermissionLevelsRequest

	cmd.Use = "get-permission-levels PIPELINE_ID"
	cmd.Short = `Get pipeline permission levels.`
	cmd.Long = `Get pipeline permission levels.

  Gets the permission levels that a user can have on an object.

  Arguments:
    PIPELINE_ID: The pipeline for which to get or manage permissions.`

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

		_ = args[0] // TODO: field not in sdk-go request: pipeline_id

		getPermissionLevelsReq.RequestObjectType = new("pipelines")

		sdkReq := &getPermissionLevelsReq
		response, err := w.AccessManagement.ListPermissionLevels(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.ListPermissionLevelsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkaccessmanagement.GetObjectPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var getPermissionsReq sdkaccessmanagement.GetObjectPermissionsRequest

	cmd.Use = "get-permissions PIPELINE_ID"
	cmd.Short = `Get pipeline permissions.`
	cmd.Long = `Get pipeline permissions.

  Gets the permissions of a pipeline. Pipelines can inherit permissions from
  their root object.

  Arguments:
    PIPELINE_ID: The pipeline for which to get or manage permissions.`

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

		_ = args[0] // TODO: field not in sdk-go request: pipeline_id

		getPermissionsReq.RequestObjectType = new("pipelines")

		sdkReq := &getPermissionsReq
		response, err := w.AccessManagement.GetObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkpipelines.GetUpdateRequest,
)

func newGetUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var getUpdateReq sdkpipelines.GetUpdateRequest

	cmd.Use = "get-update PIPELINE_ID UPDATE_ID"
	cmd.Short = `Get a pipeline update.`
	cmd.Long = `Get a pipeline update.

  Gets an update from an active pipeline.

  Arguments:
    PIPELINE_ID: The ID of the pipeline.
    UPDATE_ID: The ID of the update.`

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

		getUpdateReq.PipelineId = new(args[0])
		getUpdateReq.UpdateId = new(args[1])

		sdkReq := &getUpdateReq
		response, err := w.Pipelines.GetUpdate(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := pipelines.GetUpdateResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listPipelineEventsLimit int

	flags.StringPtrVar(cmd.Flags(), &listPipelineEventsReq.Filter, "filter", `Criteria to select a subset of results, expressed using a SQL-like syntax.`)
	flags.IntPtrVar(cmd.Flags(), &listPipelineEventsReq.MaxResults, "max-results", `Max number of entries to return in a single page.`)
	// TODO: array: order_by

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listPipelineEventsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listPipelineEventsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-pipeline-events PIPELINE_ID"
	cmd.Short = `List pipeline events.`
	cmd.Long = `List pipeline events.

  Retrieves events for a pipeline.

  Arguments:
    PIPELINE_ID: The pipeline to return events for.`

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

		listPipelineEventsReq.PipelineId = new(args[0])

		sdkReq, err := pipelines.ListPipelineEventsRequestToSDK(&listPipelineEventsReq)
		if err != nil {
			return err
		}
		if listPipelineEventsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listPipelineEventsLimit)
		}
		if listPipelineEventsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listPipelineEventsLimit)
		}
		response := w.Pipelines.EventsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, pipelines.PipelineEventFromSDK))
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

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listPipelinesLimit int

	flags.StringPtrVar(cmd.Flags(), &listPipelinesReq.Filter, "filter", `Select a subset of results based on the specified criteria.`)
	flags.IntPtrVar(cmd.Flags(), &listPipelinesReq.MaxResults, "max-results", `The maximum number of entries to return in a single page.`)
	// TODO: array: order_by

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listPipelinesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listPipelinesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-pipelines"
	cmd.Short = `List pipelines.`
	cmd.Long = `List pipelines.

  Lists pipelines defined in the Spark Declarative Pipelines system.`

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

		sdkReq, err := pipelines.ListPipelinesRequestToSDK(&listPipelinesReq)
		if err != nil {
			return err
		}
		if listPipelinesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listPipelinesLimit)
		}
		if listPipelinesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listPipelinesLimit)
		}
		response := w.Pipelines.ListIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, pipelines.PipelineStateInfoFromSDK))
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

	flags.IntPtrVar(cmd.Flags(), &listUpdatesReq.MaxResults, "max-results", `Max number of entries to return in a single page.`)
	flags.StringPtrVar(cmd.Flags(), &listUpdatesReq.PageToken, "page-token", `Page token returned by previous call.`)
	flags.StringPtrVar(cmd.Flags(), &listUpdatesReq.UntilUpdateId, "until-update-id", `If present, returns updates until and including this update_id.`)

	cmd.Use = "list-updates PIPELINE_ID"
	cmd.Short = `List pipeline updates.`
	cmd.Long = `List pipeline updates.

  List updates for an active pipeline.

  Arguments:
    PIPELINE_ID: The pipeline to return updates for.`

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

		listUpdatesReq.PipelineId = new(args[0])

		sdkReq, err := pipelines.ListUpdatesRequestToSDK(&listUpdatesReq)
		if err != nil {
			return err
		}
		response, err := w.Pipelines.ListUpdates(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := pipelines.ListUpdatesResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*accessmanagement.SetObjectPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var setPermissionsReq accessmanagement.SetObjectPermissionsRequest

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
			diags := setPermissionsJson.Unmarshal(&setPermissionsReq)
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
		_ = args[0] // TODO: field not in sdk-go request: pipeline_id

		setPermissionsReq.RequestObjectType = new("pipelines")

		sdkReq, err := accessmanagement.SetObjectPermissionsRequestToSDK(&setPermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.AccessManagement.SetObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*pipelines.StartUpdateRequest,
)

func newStartUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var startUpdateReq pipelines.StartUpdateRequest

	var startUpdateJson flags.JsonFlag

	cmd.Flags().Var(&startUpdateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.EnumVar(cmd.Flags(), &startUpdateReq.Cause, "cause", `Supported values: [
  API_CALL,
  INFRASTRUCTURE_MAINTENANCE,
  JOB_TASK,
  RETRY_ON_FAILURE,
  SCHEMA_CHANGE,
  SERVICE_UPGRADE,
  USER_ACTION,
]`, "StartUpdateCause", "API_CALL", "INFRASTRUCTURE_MAINTENANCE", "JOB_TASK", "RETRY_ON_FAILURE", "SCHEMA_CHANGE", "SERVICE_UPGRADE", "USER_ACTION")
	flags.BoolPtrVar(cmd.Flags(), &startUpdateReq.Development, "development", `Whether the update is started in the development mode.`)
	flags.BoolPtrVar(cmd.Flags(), &startUpdateReq.FullRefresh, "full-refresh", `If true, this update will reset all tables before running.`)
	// TODO: array: full_refresh_selection
	// TODO: map via StringToStringVar: parameters
	// TODO: array: refresh_selection
	// TODO: array: replace_where_overrides
	// TODO: array: reset_checkpoint_selection
	// TODO: complex arg: rewind_spec
	flags.BoolPtrVar(cmd.Flags(), &startUpdateReq.ValidateOnly, "validate-only", `If true, this update only validates the correctness of pipeline source code but does not materialize or publish any datasets.`)

	cmd.Use = "start-update PIPELINE_ID"
	cmd.Short = `Start a pipeline.`
	cmd.Long = `Start a pipeline.

  Starts a new update for the pipeline. If there is already an active update for
  the pipeline, the request will fail and the active update will remain running.`

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
			diags := startUpdateJson.Unmarshal(&startUpdateReq)
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
		startUpdateReq.PipelineId = new(args[0])

		sdkReq, err := pipelines.StartUpdateRequestToSDK(&startUpdateReq)
		if err != nil {
			return err
		}
		response, err := w.Pipelines.Start(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := pipelines.StartUpdateResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*pipelines.StopPipelineRequest,
)

func newStop() *cobra.Command {
	cmd := &cobra.Command{}
	var stopReq pipelines.StopPipelineRequest

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

		stopReq.PipelineId = new(args[0])

		sdkReq, err := pipelines.StopPipelineRequestToSDK(&stopReq)
		if err != nil {
			return err
		}
		wait, err := w.Pipelines.Stop(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if stopSkipWait {
			// Render the current state, as of right after the call.
			current, err := w.Pipelines.Get(ctx, sdkpipelines.GetPipelineRequest{PipelineId: new(wait.GetPipelineId())})
			if err != nil {
				return err
			}
			out, err := pipelines.GetPipelineResponseFromSDK(current)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for stop to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(stopTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := pipelines.GetPipelineResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*pipelines.EditPipelineRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq pipelines.EditPipelineRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &updateReq.AllowDuplicateNames, "allow-duplicate-names", `If false, deployment will fail if name has changed and conflicts the name of another pipeline.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.BudgetPolicyId, "budget-policy-id", `Budget policy of this pipeline.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Catalog, "catalog", `A catalog in Unity Catalog to publish data from this pipeline to.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Channel, "channel", `SDP Release Channel that specifies which version to use.`)
	// TODO: array: clusters
	// TODO: map via StringToStringVar: configuration
	flags.BoolPtrVar(cmd.Flags(), &updateReq.Continuous, "continuous", `Whether the pipeline is continuous or triggered.`)
	// TODO: complex arg: deployment
	flags.BoolPtrVar(cmd.Flags(), &updateReq.Development, "development", `Whether the pipeline is in Development mode.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Edition, "edition", `Pipeline product edition.`)
	// TODO: complex arg: environment
	// TODO: complex arg: event_log
	flags.Int64PtrVar(cmd.Flags(), &updateReq.ExpectedLastModified, "expected-last-modified", `If present, the last-modified time of the pipeline settings before the edit.`)
	// TODO: complex arg: filters
	// TODO: complex arg: gateway_definition
	flags.StringPtrVar(cmd.Flags(), &updateReq.Id, "id", `Unique identifier for this pipeline.`)
	// TODO: complex arg: ingestion_definition
	// TODO: array: libraries
	flags.StringPtrVar(cmd.Flags(), &updateReq.Name, "name", `Friendly identifier for this pipeline.`)
	// TODO: array: notifications
	// TODO: map via StringToStringVar: parameters
	flags.StringPtrVar(cmd.Flags(), &updateReq.ParentPath, "parent-path", `Path of the pipeline parent folder in workspace file tree.`)
	flags.BoolPtrVar(cmd.Flags(), &updateReq.Photon, "photon", `Whether Photon is enabled for this pipeline.`)
	// TODO: complex arg: restart_window
	flags.StringPtrVar(cmd.Flags(), &updateReq.RootPath, "root-path", `Root path for this pipeline.`)
	// TODO: complex arg: run_as
	flags.StringPtrVar(cmd.Flags(), &updateReq.Schema, "schema", `The default schema (database) where tables are read from or published to.`)
	flags.BoolPtrVar(cmd.Flags(), &updateReq.Serverless, "serverless", `Whether serverless compute is enabled for this pipeline.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.ServerlessComputeId, "serverless-compute-id", `Serverless compute ID specified by the user for serverless pipelines.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Storage, "storage", `DBFS root directory for storing checkpoints and tables.`)
	// TODO: map via StringToStringVar: tags
	flags.StringPtrVar(cmd.Flags(), &updateReq.Target, "target", `Target schema (database) to add tables in this pipeline to.`)
	// TODO: complex arg: trigger
	flags.StringPtrVar(cmd.Flags(), &updateReq.UsagePolicyId, "usage-policy-id", `Usage policy of this pipeline.`)

	cmd.Use = "update PIPELINE_ID"
	cmd.Short = `Edit a pipeline.`
	cmd.Long = `Edit a pipeline.

  Updates a pipeline with the supplied configuration.

  Arguments:
    PIPELINE_ID: Unique identifier for this pipeline.`

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
		}
		updateReq.PipelineId = new(args[0])

		sdkReq, err := pipelines.EditPipelineRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Pipelines.Edit(ctx, *sdkReq)
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
	*accessmanagement.UpdateObjectPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var updatePermissionsReq accessmanagement.UpdateObjectPermissionsRequest

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
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
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
		_ = args[0] // TODO: field not in sdk-go request: pipeline_id

		updatePermissionsReq.RequestObjectType = new("pipelines")

		sdkReq, err := accessmanagement.UpdateObjectPermissionsRequestToSDK(&updatePermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.AccessManagement.UpdateObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
