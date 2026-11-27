// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package genie_migrate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/experimental/api"
	"github.com/databricks/databricks-sdk-go/service/geniemigrate"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

// Spec declares the command for this service.
var Spec = root.CommandSpec{
	Use:     "genie-migrate",
	Short:   `Use the Genie Migrate API to create and manage migration projects and the source files to convert.`,
	GroupID: "migration",

	// This service is being previewed; hide from help output.
	Hidden: true,
	Build:  buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Use the Genie Migrate API to create and manage migration projects and the
  source files to convert.`
	cmd.RunE = root.ReportUnknownSubcommand

	cmd.Annotations = map[string]string{
		"launch_stage":         "PRIVATE_PREVIEW",
		"launch_stage_display": "Private Preview",
	}

	// Add methods
	cmd.AddCommand(newAddFiles())
	cmd.AddCommand(newCancelOperation())
	cmd.AddCommand(newConvertProjectFiles())
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetFile())
	cmd.AddCommand(newGetOperation())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListFiles())
	cmd.AddCommand(newRemoveFiles())
	cmd.AddCommand(newReviewProjectFiles())
	cmd.AddCommand(newUnreviewProjectFiles())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}
}

// start add-files command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var addFilesOverrides []func(
	*cobra.Command,
	*geniemigrate.AddProjectFilesRequest,
)

func newAddFiles() *cobra.Command {
	cmd := &cobra.Command{}

	var addFilesReq geniemigrate.AddProjectFilesRequest
	var addFilesJson flags.JsonFlag

	cmd.Flags().Var(&addFilesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "add-files NAME"
	cmd.Short = `Add files to a migration project.`
	cmd.Long = `Add files to a migration project.

  Adds source files to a migration project. Files that are already in the
  project are skipped and keep their conversion results.

  Arguments:
    NAME: The resource name of the project, in the format projects/{project_id}.
      For example, projects/123.`

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

		if cmd.Flags().Changed("json") {
			diags := addFilesJson.Unmarshal(&addFilesReq)
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
		addFilesReq.Name = args[0]

		response, err := w.GenieMigrate.AddFiles(ctx, addFilesReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range addFilesOverrides {
		fn(cmd, &addFilesReq)
	}

	return cmd
}

// start cancel-operation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cancelOperationOverrides []func(
	*cobra.Command,
	*geniemigrate.CancelOperationRequest,
)

func newCancelOperation() *cobra.Command {
	cmd := &cobra.Command{}

	var cancelOperationReq geniemigrate.CancelOperationRequest

	cmd.Use = "cancel-operation NAME"
	cmd.Short = `Cancel an operation.`
	cmd.Long = `Cancel an operation.

  Requests cancellation of a long-running operation. Requires CAN_EDIT
  permission on the project. Cancellation is best effort: the operation can
  complete before cancellation takes effect. Call GetOperation to see its
  final state.

  Arguments:
    NAME: The name of the operation resource to be cancelled.`

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

		cancelOperationReq.Name = args[0]

		err = w.GenieMigrate.CancelOperation(ctx, cancelOperationReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range cancelOperationOverrides {
		fn(cmd, &cancelOperationReq)
	}

	return cmd
}

// start convert-project-files command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var convertProjectFilesOverrides []func(
	*cobra.Command,
	*geniemigrate.ConvertProjectFilesRequest,
)

func newConvertProjectFiles() *cobra.Command {
	cmd := &cobra.Command{}

	var convertProjectFilesReq geniemigrate.ConvertProjectFilesRequest
	var convertProjectFilesJson flags.JsonFlag

	var convertProjectFilesSkipWait bool
	var convertProjectFilesTimeout time.Duration

	cmd.Flags().BoolVar(&convertProjectFilesSkipWait, "no-wait", convertProjectFilesSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&convertProjectFilesTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&convertProjectFilesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: names
	// TODO: array: skill_paths
	cmd.Flags().BoolVar(&convertProjectFilesReq.StartOver, "start-over", convertProjectFilesReq.StartOver, `Whether to convert each selected file from scratch instead of continuing from its previous results (changes, to-do items, blockers, and notes).`)

	cmd.Use = "convert-project-files PARENT"
	cmd.Short = `Convert a migration project's files.`
	cmd.Long = `Convert a migration project's files.

  Converts files of a migration project in a background job. Returns an
  operation; poll it with GetOperation. The operation's metadata is the
  current ConversionRun, with the run's state and the status of each file.
  When the run completes, the operation's response is a
  ConvertProjectFilesResponse. A failed or cancelled run sets the operation's
  error instead. Files that fail to convert don't fail the run.

  To stop a conversion, call CancelOperation. Cancellation is best effort, and
  the conversion can complete first. GetOperation reports the final outcome.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: The resource name of the project whose files to convert, in the format
      projects/{project_id}. For example, projects/123.`

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

		if cmd.Flags().Changed("json") {
			diags := convertProjectFilesJson.Unmarshal(&convertProjectFilesReq)
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
		convertProjectFilesReq.Parent = args[0]

		// Determine which mode to execute based on flags.
		switch {
		case convertProjectFilesSkipWait:
			wait, err := w.GenieMigrate.ConvertProjectFiles(ctx, convertProjectFilesReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.GenieMigrate.GetOperation(ctx, geniemigrate.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.GenieMigrate.ConvertProjectFiles(ctx, convertProjectFilesReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for convert-project-files to complete...")

			// Wait for completion.
			opts := api.WithTimeout(convertProjectFilesTimeout)
			response, err := wait.Wait(ctx, opts)
			if err != nil {
				return err
			}
			sp.Close()
			return cmdio.Render(ctx, response)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range convertProjectFilesOverrides {
		fn(cmd, &convertProjectFilesReq)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*geniemigrate.CreateProjectRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq geniemigrate.CreateProjectRequest
	createReq.Project = geniemigrate.Project{}
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Project.Folder, "folder", createReq.Project.Folder, `The workspace folder that contains the project, in the format folders/{folder_id}.`)
	cmd.Flags().StringVar(&createReq.Project.Name, "name", createReq.Project.Name, `The resource name of the project, in the format projects/{project_id}.`)

	cmd.Use = "create PROJECT_CONFIG DISPLAY_NAME"
	cmd.Short = `Create a migration project.`
	cmd.Long = `Create a migration project.

  Creates a migration project in a workspace folder. The server assigns the
  project ID. Add source files to the project with AddProjectFiles.

  Arguments:
    PROJECT_CONFIG: The conversion settings of the project. Required on create. Returned by
      CreateProject, GetProject, and UpdateProject, but not by
      ListProjects.
    DISPLAY_NAME: The display name of the project. Required on create, and can't be blank.
      Changing it renames the project.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'project_config', 'display_name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq.Project)
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
			_, err = fmt.Sscan(args[0], &createReq.Project.ProjectConfig)
			if err != nil {
				return fmt.Errorf("invalid PROJECT_CONFIG: %s", args[0])
			}

		}
		if !cmd.Flags().Changed("json") {
			createReq.Project.DisplayName = args[1]
		}

		response, err := w.GenieMigrate.Create(ctx, createReq)
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
	*geniemigrate.DeleteProjectRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq geniemigrate.DeleteProjectRequest

	cmd.Use = "delete NAME"
	cmd.Short = `Move a migration project to the trash.`
	cmd.Long = `Move a migration project to the trash.

  Moves a migration project to the workspace trash. The project keeps its ID
  until it's removed from the trash.

  Arguments:
    NAME: The resource name of the project, in the format projects/{project_id}.
      For example, projects/123.`

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

		deleteReq.Name = args[0]

		err = w.GenieMigrate.Delete(ctx, deleteReq)
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
	*geniemigrate.GetProjectRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq geniemigrate.GetProjectRequest

	cmd.Use = "get NAME"
	cmd.Short = `Get a migration project.`
	cmd.Long = `Get a migration project.

  Gets a migration project, including its conversion settings and whether
  they're locked. To list the project's source files, call ListProjectFiles.
  To track a conversion, call GetOperation.

  Arguments:
    NAME: The resource name of the project, in the format projects/{project_id}.
      For example, projects/123.`

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

		getReq.Name = args[0]

		response, err := w.GenieMigrate.Get(ctx, getReq)
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

// start get-file command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getFileOverrides []func(
	*cobra.Command,
	*geniemigrate.GetProjectFileRequest,
)

func newGetFile() *cobra.Command {
	cmd := &cobra.Command{}

	var getFileReq geniemigrate.GetProjectFileRequest

	cmd.Use = "get-file NAME"
	cmd.Short = `Get a file in a migration project.`
	cmd.Long = `Get a file in a migration project.

  Gets a file of a migration project, including the conversion details stored
  with its converted file, when available. Requires CAN_READ permission on the
  project and, once the file is converted, on the converted file.

  Arguments:
    NAME: The resource name of the file, in the format
      projects/{project_id}/files/{file_id}. For example,
      projects/123/files/789.`

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

		getFileReq.Name = args[0]

		response, err := w.GenieMigrate.GetFile(ctx, getFileReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getFileOverrides {
		fn(cmd, &getFileReq)
	}

	return cmd
}

// start get-operation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOperationOverrides []func(
	*cobra.Command,
	*geniemigrate.GetOperationRequest,
)

func newGetOperation() *cobra.Command {
	cmd := &cobra.Command{}

	var getOperationReq geniemigrate.GetOperationRequest

	cmd.Use = "get-operation NAME"
	cmd.Short = `Get an operation.`
	cmd.Long = `Get an operation.

  Gets the latest state of a long-running operation, such as a conversion
  started by ConvertProjectFiles. Operation names have the format
  projects/{project_id}/operations/{operation_id}. Use the full name that the
  service returned, and treat the operation ID as opaque. Requires CAN_READ
  permission on the project.

  Arguments:
    NAME: The name of the operation resource.`

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

		getOperationReq.Name = args[0]

		response, err := w.GenieMigrate.GetOperation(ctx, getOperationReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOperationOverrides {
		fn(cmd, &getOperationReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*geniemigrate.ListProjectsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq geniemigrate.ListProjectsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	cmd.Flags().StringVar(&listReq.Filter, "filter", listReq.Filter, `The folder to list, as folder = "folders/{folder_id}".`)
	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `The maximum number of projects to return, at most 500.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List migration projects.`
	cmd.Long = `List migration projects.

  Lists the migration projects in a workspace folder. Select the folder with
  filter; without a filter, lists the caller's home folder. Listing every
  project in the workspace isn't supported.

  Returns project summaries without conversion settings; call GetProject for
  the full project. A page can be empty even when more pages follow, so keep
  requesting pages until next_page_token is empty.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.GenieMigrate.List(ctx, listReq)
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

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

// start list-files command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listFilesOverrides []func(
	*cobra.Command,
	*geniemigrate.ListProjectFilesRequest,
)

func newListFiles() *cobra.Command {
	cmd := &cobra.Command{}

	var listFilesReq geniemigrate.ListProjectFilesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listFilesLimit int

	cmd.Flags().IntVar(&listFilesReq.PageSize, "page-size", listFilesReq.PageSize, `The maximum number of files to return, at most 500.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listFilesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listFilesReq.PageToken, "page-token", listFilesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-files PARENT"
	cmd.Short = `List files in a migration project.`
	cmd.Long = `List files in a migration project.

  Lists the files of a migration project with their latest conversion statuses.
  To read a file's changes, to-do items, blockers, and notes, call
  GetProjectFile. A page can have fewer files than requested, or none, even
  when more pages follow; keep requesting pages until next_page_token is
  empty.

  Arguments:
    PARENT: The resource name of the project, in the format projects/{project_id}.
      For example, projects/123.`

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

		listFilesReq.Parent = args[0]

		response := w.GenieMigrate.ListFiles(ctx, listFilesReq)
		if listFilesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listFilesLimit)
		}
		if listFilesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listFilesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listFilesOverrides {
		fn(cmd, &listFilesReq)
	}

	return cmd
}

// start remove-files command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var removeFilesOverrides []func(
	*cobra.Command,
	*geniemigrate.RemoveProjectFilesRequest,
)

func newRemoveFiles() *cobra.Command {
	cmd := &cobra.Command{}

	var removeFilesReq geniemigrate.RemoveProjectFilesRequest
	var removeFilesJson flags.JsonFlag

	cmd.Flags().Var(&removeFilesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: names

	cmd.Use = "remove-files NAME"
	cmd.Short = `Remove files from a migration project.`
	cmd.Long = `Remove files from a migration project.

  Removes files from a migration project by their resource names. The source
  files and their converted output stay in the workspace.

  Arguments:
    NAME: The resource name of the project, in the format projects/{project_id}.
      For example, projects/123.`

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

		if cmd.Flags().Changed("json") {
			diags := removeFilesJson.Unmarshal(&removeFilesReq)
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
		removeFilesReq.Name = args[0]

		response, err := w.GenieMigrate.RemoveFiles(ctx, removeFilesReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range removeFilesOverrides {
		fn(cmd, &removeFilesReq)
	}

	return cmd
}

// start review-project-files command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var reviewProjectFilesOverrides []func(
	*cobra.Command,
	*geniemigrate.ReviewProjectFilesRequest,
)

func newReviewProjectFiles() *cobra.Command {
	cmd := &cobra.Command{}

	var reviewProjectFilesReq geniemigrate.ReviewProjectFilesRequest
	var reviewProjectFilesJson flags.JsonFlag

	cmd.Flags().Var(&reviewProjectFilesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "review-project-files PARENT"
	cmd.Short = `Review files in a migration project.`
	cmd.Long = `Review files in a migration project.

  Marks files of a migration project as reviewed, accepting their converted
  output. Each file's status becomes FILE_STATUS_REVIEWED, and reviewed files
  aren't converted again. Files that are already reviewed don't change. Returns
  the files in request order.

  Arguments:
    PARENT: The resource name of the project, in the format projects/{project_id}.
      For example, projects/123. Every file in names must belong to this
      project.`

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

		if cmd.Flags().Changed("json") {
			diags := reviewProjectFilesJson.Unmarshal(&reviewProjectFilesReq)
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
		reviewProjectFilesReq.Parent = args[0]

		response, err := w.GenieMigrate.ReviewProjectFiles(ctx, reviewProjectFilesReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range reviewProjectFilesOverrides {
		fn(cmd, &reviewProjectFilesReq)
	}

	return cmd
}

// start unreview-project-files command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var unreviewProjectFilesOverrides []func(
	*cobra.Command,
	*geniemigrate.UnreviewProjectFilesRequest,
)

func newUnreviewProjectFiles() *cobra.Command {
	cmd := &cobra.Command{}

	var unreviewProjectFilesReq geniemigrate.UnreviewProjectFilesRequest
	var unreviewProjectFilesJson flags.JsonFlag

	cmd.Flags().Var(&unreviewProjectFilesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "unreview-project-files PARENT"
	cmd.Short = `Unreview files in a migration project.`
	cmd.Long = `Unreview files in a migration project.

  Removes the reviewed status from files of a migration project. Each file
  returns to the status it had before it was reviewed. Files that aren't
  reviewed don't change. Returns the files in request order.

  Arguments:
    PARENT: The resource name of the project, in the format projects/{project_id}.
      For example, projects/123. Every file in names must belong to this
      project.`

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

		if cmd.Flags().Changed("json") {
			diags := unreviewProjectFilesJson.Unmarshal(&unreviewProjectFilesReq)
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
		unreviewProjectFilesReq.Parent = args[0]

		response, err := w.GenieMigrate.UnreviewProjectFiles(ctx, unreviewProjectFilesReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range unreviewProjectFilesOverrides {
		fn(cmd, &unreviewProjectFilesReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*geniemigrate.UpdateProjectRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq geniemigrate.UpdateProjectRequest
	updateReq.Project = geniemigrate.Project{}
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Project.Folder, "folder", updateReq.Project.Folder, `The workspace folder that contains the project, in the format folders/{folder_id}.`)
	cmd.Flags().StringVar(&updateReq.Project.Name, "name", updateReq.Project.Name, `The resource name of the project, in the format projects/{project_id}.`)

	cmd.Use = "update NAME UPDATE_MASK PROJECT_CONFIG DISPLAY_NAME"
	cmd.Short = `Update a migration project.`
	cmd.Long = `Update a migration project.

  Updates a migration project. Only the fields listed in update_mask change.
  An empty mask, or a mask with an unsupported field, returns
  INVALID_PARAMETER_VALUE. When changes_locked is true, the source dialect
  and target framework can't be changed.

  Arguments:
    NAME: The resource name of the project, in the format projects/{project_id}.
      For example, projects/123. The server assigns it when the project is
      created.
    UPDATE_MASK: The fields to update. Supported fields are display_name,
      project_config.source_dialect, project_config.target_framework, and
      project_config.lineage_location. project_config and * update all
      three project_config fields and can't be combined with display_name.
    PROJECT_CONFIG: The conversion settings of the project. Required on create. Returned by
      CreateProject, GetProject, and UpdateProject, but not by
      ListProjects.
    DISPLAY_NAME: The display name of the project. Required on create, and can't be blank.
      Changing it renames the project.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'project_config', 'display_name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq.Project)
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
		updateReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &updateReq.Project.ProjectConfig)
			if err != nil {
				return fmt.Errorf("invalid PROJECT_CONFIG: %s", args[2])
			}

		}
		if !cmd.Flags().Changed("json") {
			updateReq.Project.DisplayName = args[3]
		}

		response, err := w.GenieMigrate.Update(ctx, updateReq)
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

// end service GenieMigrate
