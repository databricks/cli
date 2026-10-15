// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clean_room_task_runs

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	cleanrooms "github.com/databricks/cli/libs/sdktypes/cleanrooms/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean-room-task-runs",
		Short: `Clean room task runs are the executions of notebooks and JAR analyses in a clean room.`,
		Long: `Clean room task runs are the executions of notebooks and JAR analyses in a
  clean room.`,
		GroupID: "cleanrooms",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newList())
	cmd.AddCommand(newListCleanRoomTaskRunsHandler())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*cleanrooms.ListCleanRoomNotebookTaskRunsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq cleanrooms.ListCleanRoomNotebookTaskRunsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.StringPtrVar(cmd.Flags(), &listReq.NotebookName, "notebook-name", `Notebook name.`)
	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", `The maximum number of task runs to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list CLEAN_ROOM_NAME"
	cmd.Short = `List notebook task runs.`
	cmd.Long = `List notebook task runs.

  List all the historical notebook task runs in a clean room.

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.`

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

		listReq.CleanRoomName = new(args[0])

		sdkReq, err := cleanrooms.ListCleanRoomNotebookTaskRunsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.CleanRooms.ListCleanRoomNotebookTaskRunsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, cleanrooms.CleanRoomNotebookTaskRunFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// start list-clean-room-task-runs-handler command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listCleanRoomTaskRunsHandlerOverrides []func(
	*cobra.Command,
	*cleanrooms.ListCleanRoomTaskRunsRequest,
)

func newListCleanRoomTaskRunsHandler() *cobra.Command {
	cmd := &cobra.Command{}
	var listCleanRoomTaskRunsHandlerReq cleanrooms.ListCleanRoomTaskRunsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listCleanRoomTaskRunsHandlerLimit int

	flags.StringPtrVar(cmd.Flags(), &listCleanRoomTaskRunsHandlerReq.Name, "name", `Executable name.`)
	flags.IntPtrVar(cmd.Flags(), &listCleanRoomTaskRunsHandlerReq.PageSize, "page-size", `The maximum number of task runs to return.`)
	flags.EnumVar(cmd.Flags(), &listCleanRoomTaskRunsHandlerReq.TaskType, "task-type", `Filter by the type of Clean Room task. Supported values: [JAR, NOTEBOOK]`, "CleanRoomTaskType", "JAR", "NOTEBOOK")

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listCleanRoomTaskRunsHandlerLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listCleanRoomTaskRunsHandlerReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-clean-room-task-runs-handler CLEAN_ROOM_NAME"
	cmd.Short = `List task runs.`
	cmd.Long = `List task runs.

  List all the historical task runs in a clean room.

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.`

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

		listCleanRoomTaskRunsHandlerReq.CleanRoomName = new(args[0])

		sdkReq, err := cleanrooms.ListCleanRoomTaskRunsRequestToSDK(&listCleanRoomTaskRunsHandlerReq)
		if err != nil {
			return err
		}
		if listCleanRoomTaskRunsHandlerLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listCleanRoomTaskRunsHandlerLimit)
		}
		if listCleanRoomTaskRunsHandlerLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listCleanRoomTaskRunsHandlerLimit)
		}
		response := w.CleanRooms.ListCleanRoomTaskRunsHandlerIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, cleanrooms.CleanRoomTaskRunFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listCleanRoomTaskRunsHandlerOverrides {
		fn(cmd, &listCleanRoomTaskRunsHandlerReq)
	}

	return cmd
}

// end service CleanRoomTaskRuns
