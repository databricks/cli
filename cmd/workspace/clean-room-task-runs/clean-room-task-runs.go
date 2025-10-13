// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clean_room_task_runs

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/cleanrooms"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clean-room-task-runs",
		Short:   `Clean room task runs are the executions of notebooks in a clean room.`,
		Long:    `Clean room task runs are the executions of notebooks in a clean room.`,
		GroupID: "cleanrooms",
		Annotations: map[string]string{
			"package": "cleanrooms",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newList())

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

	cmd.Flags().StringVar(&listReq.NotebookName, "notebook-name", listReq.NotebookName, `Notebook name.`)
	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `The maximum number of task runs to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list CLEAN_ROOM_NAME"
	cmd.Short = `List notebook task runs.`
	cmd.Long = `List notebook task runs.
  
  List all the historical notebook task runs in a clean room.

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listReq.CleanRoomName = args[0]

		response := w.CleanRoomTaskRuns.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
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

// end service CleanRoomTaskRuns
