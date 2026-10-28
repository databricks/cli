// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/sandbox"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sandbox",
		Short: `*Beta* Create, manage, and control the lifecycle of sandboxes -- isolated, pre-configured, low-latency Serverless compute environments for running code.`,
		Long: `This command is in Beta and may change without notice.

Create, manage, and control the lifecycle of sandboxes -- isolated,
  pre-configured, low-latency Serverless compute environments for running code.`,
		GroupID: "sandbox",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	// Add methods
	cmd.AddCommand(newCreateDirectory())
	cmd.AddCommand(newCreateSandbox())
	cmd.AddCommand(newDeletePath())
	cmd.AddCommand(newDeleteSandbox())
	cmd.AddCommand(newExecuteCommandSync())
	cmd.AddCommand(newGetMetadata())
	cmd.AddCommand(newGetSandbox())
	cmd.AddCommand(newListCommands())
	cmd.AddCommand(newListDirectoryContents())
	cmd.AddCommand(newListSandboxes())
	cmd.AddCommand(newRenamePath())
	cmd.AddCommand(newStartSandbox())
	cmd.AddCommand(newStopSandbox())
	cmd.AddCommand(newUpdateSandbox())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-directory command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDirectoryOverrides []func(
	*cobra.Command,
	*sandbox.CreateDirectoryRequest,
)

func newCreateDirectory() *cobra.Command {
	cmd := &cobra.Command{}

	var createDirectoryReq sandbox.CreateDirectoryRequest
	var createDirectoryJson flags.JsonFlag

	cmd.Flags().Var(&createDirectoryJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&createDirectoryReq.Parents, "parents", createDirectoryReq.Parents, `Defaults to true, which also succeeds when the directory already exists.`)

	cmd.Use = "create-directory PARENT PATH"
	cmd.Short = `.`
	cmd.Long = `.

  Creates a directory in a sandbox, including missing parents by default.

  Arguments:
    PARENT:
    PATH: Absolute UTF-8 path of the directory.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'path' in your JSON input")
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
			diags := createDirectoryJson.Unmarshal(&createDirectoryReq)
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
		createDirectoryReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			createDirectoryReq.Path = args[1]
		}

		response, err := w.Sandbox.CreateDirectory(ctx, createDirectoryReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDirectoryOverrides {
		fn(cmd, &createDirectoryReq)
	}

	return cmd
}

// start create-sandbox command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSandboxOverrides []func(
	*cobra.Command,
	*sandbox.CreateSandboxRequest,
)

func newCreateSandbox() *cobra.Command {
	cmd := &cobra.Command{}

	var createSandboxReq sandbox.CreateSandboxRequest
	createSandboxReq.Sandbox = sandbox.Sandbox{}
	var createSandboxJson flags.JsonFlag

	cmd.Flags().Var(&createSandboxJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createSandboxReq.Sandbox.DisplayName, "display-name", createSandboxReq.Sandbox.DisplayName, `Human-readable display label for the sandbox.`)
	// TODO: complex arg: metadata
	cmd.Flags().StringVar(&createSandboxReq.Sandbox.Name, "name", createSandboxReq.Sandbox.Name, `The sandbox resource name, in the form sandboxes/{sandbox_id}.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-sandbox SANDBOX_ID"
	cmd.Short = `*Beta* Create a sandbox.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a sandbox.

  Creates a new Sandbox.

  Arguments:
    SANDBOX_ID: Client-supplied ID that becomes the final path segment of the resource
      name.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createSandboxJson.Unmarshal(&createSandboxReq.Sandbox)
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
		createSandboxReq.SandboxId = args[0]

		response, err := w.Sandbox.CreateSandbox(ctx, createSandboxReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createSandboxOverrides {
		fn(cmd, &createSandboxReq)
	}

	return cmd
}

// start delete-path command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deletePathOverrides []func(
	*cobra.Command,
	*sandbox.DeletePathRequest,
)

func newDeletePath() *cobra.Command {
	cmd := &cobra.Command{}

	var deletePathReq sandbox.DeletePathRequest
	var deletePathJson flags.JsonFlag

	cmd.Flags().Var(&deletePathJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&deletePathReq.Recursive, "recursive", deletePathReq.Recursive, `Required for a non-empty directory.`)

	cmd.Use = "delete-path PARENT PATH"
	cmd.Short = `.`
	cmd.Long = `.

  Deletes a file, symlink, or empty directory, with an option for recursive
  deletion. Succeeds if the path is already absent.

  Arguments:
    PARENT:
    PATH: Absolute UTF-8 path of the entry to delete.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'path' in your JSON input")
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
			diags := deletePathJson.Unmarshal(&deletePathReq)
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
		deletePathReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			deletePathReq.Path = args[1]
		}

		response, err := w.Sandbox.DeletePath(ctx, deletePathReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deletePathOverrides {
		fn(cmd, &deletePathReq)
	}

	return cmd
}

// start delete-sandbox command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSandboxOverrides []func(
	*cobra.Command,
	*sandbox.DeleteSandboxRequest,
)

func newDeleteSandbox() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteSandboxReq sandbox.DeleteSandboxRequest

	cmd.Use = "delete-sandbox NAME"
	cmd.Short = `*Beta* Delete a sandbox.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a sandbox.

  Deletes a Sandbox.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteSandboxReq.Name = args[0]

		err = w.Sandbox.DeleteSandbox(ctx, deleteSandboxReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSandboxOverrides {
		fn(cmd, &deleteSandboxReq)
	}

	return cmd
}

// start execute-command-sync command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var executeCommandSyncOverrides []func(
	*cobra.Command,
	*sandbox.ExecuteCommandSyncRequest,
)

func newExecuteCommandSync() *cobra.Command {
	cmd := &cobra.Command{}

	var executeCommandSyncReq sandbox.ExecuteCommandSyncRequest
	var executeCommandSyncJson flags.JsonFlag

	cmd.Flags().Var(&executeCommandSyncJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: args
	// TODO: map via StringToStringVar: envs
	var executionTimeoutParam string
	cmd.Flags().StringVar(&executionTimeoutParam, "execution-timeout", executionTimeoutParam, `Maximum time to wait for the command to finish.`)

	cmd.Use = "execute-command-sync NAME CMD"
	cmd.Short = `*Beta* Run a command in a sandbox and wait for its result.`
	cmd.Long = `This command is in Beta and may change without notice.

Run a command in a sandbox and wait for its result.

  Runs a command in the sandbox and blocks until it exits, returning the
  captured stdout, stderr and exit code in a single response.

  Arguments:
    NAME: Resource name of the sandbox to run the command in, in the form
      sandboxes/{sandbox_id}. Bound from the URL path.
    CMD: Executable or command to run (e.g. /bin/echo, python3).`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME as positional arguments. Provide 'cmd' in your JSON input")
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
			diags := executeCommandSyncJson.Unmarshal(&executeCommandSyncReq)
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
		executeCommandSyncReq.Name = args[0]
		if !cmd.Flags().Changed("json") {
			executeCommandSyncReq.Cmd = args[1]
		}

		if executionTimeoutParam != "" {
			executionTimeoutBytes := []byte(fmt.Sprintf("\"%s\"", executionTimeoutParam))
			var executionTimeoutField duration.Duration
			err = json.Unmarshal(executionTimeoutBytes, &executionTimeoutField)
			if err != nil {
				return fmt.Errorf("invalid EXECUTION_TIMEOUT: %s", executionTimeoutParam)
			}
			executeCommandSyncReq.ExecutionTimeout = &executionTimeoutField
		}

		response, err := w.Sandbox.ExecuteCommandSync(ctx, executeCommandSyncReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range executeCommandSyncOverrides {
		fn(cmd, &executeCommandSyncReq)
	}

	return cmd
}

// start get-metadata command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMetadataOverrides []func(
	*cobra.Command,
	*sandbox.GetMetadataRequest,
)

func newGetMetadata() *cobra.Command {
	cmd := &cobra.Command{}

	var getMetadataReq sandbox.GetMetadataRequest

	cmd.Use = "get-metadata PARENT PATH"
	cmd.Short = `.`
	cmd.Long = `.

  Returns metadata for a path without following its final symlink.

  Arguments:
    PARENT:
    PATH: Absolute UTF-8 path of the entry to inspect.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getMetadataReq.Parent = args[0]
		getMetadataReq.Path = args[1]

		response, err := w.Sandbox.GetMetadata(ctx, getMetadataReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getMetadataOverrides {
		fn(cmd, &getMetadataReq)
	}

	return cmd
}

// start get-sandbox command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSandboxOverrides []func(
	*cobra.Command,
	*sandbox.GetSandboxRequest,
)

func newGetSandbox() *cobra.Command {
	cmd := &cobra.Command{}

	var getSandboxReq sandbox.GetSandboxRequest

	cmd.Use = "get-sandbox NAME"
	cmd.Short = `*Beta* Get a sandbox.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a sandbox.

  Retrieves a Sandbox by name.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getSandboxReq.Name = args[0]

		response, err := w.Sandbox.GetSandbox(ctx, getSandboxReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSandboxOverrides {
		fn(cmd, &getSandboxReq)
	}

	return cmd
}

// start list-commands command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listCommandsOverrides []func(
	*cobra.Command,
	*sandbox.ListCommandsRequest,
)

func newListCommands() *cobra.Command {
	cmd := &cobra.Command{}

	var listCommandsReq sandbox.ListCommandsRequest

	cmd.Flags().IntVar(&listCommandsReq.PageSize, "page-size", listCommandsReq.PageSize, `Maximum number of commands to return.`)
	cmd.Flags().StringVar(&listCommandsReq.PageToken, "page-token", listCommandsReq.PageToken, `Page token returned by a previous ListCommands call.`)

	cmd.Use = "list-commands PARENT"
	cmd.Short = `*Beta* List the tracked command executions in a sandbox.`
	cmd.Long = `This command is in Beta and may change without notice.

List the tracked command executions in a sandbox.

  Lists the tracked command executions (running and completed) in a sandbox.

  Arguments:
    PARENT: The sandbox whose commands to list, in the form sandboxes/{sandbox_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listCommandsReq.Parent = args[0]

		response, err := w.Sandbox.ListCommands(ctx, listCommandsReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listCommandsOverrides {
		fn(cmd, &listCommandsReq)
	}

	return cmd
}

// start list-directory-contents command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDirectoryContentsOverrides []func(
	*cobra.Command,
	*sandbox.ListDirectoryContentsRequest,
)

func newListDirectoryContents() *cobra.Command {
	cmd := &cobra.Command{}

	var listDirectoryContentsReq sandbox.ListDirectoryContentsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDirectoryContentsLimit int

	cmd.Flags().IntVar(&listDirectoryContentsReq.PageSize, "page-size", listDirectoryContentsReq.PageSize, `Zero uses the default of 1000; values above 1000 are clamped to 1000.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDirectoryContentsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listDirectoryContentsReq.PageToken, "page-token", listDirectoryContentsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-directory-contents PARENT PATH"
	cmd.Short = `.`
	cmd.Long = `.

  Lists a directory's immediate children in UTF-8 name order. Later pages may
  reflect concurrent changes.

  Arguments:
    PARENT:
    PATH: Absolute UTF-8 path of the directory.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listDirectoryContentsReq.Parent = args[0]
		listDirectoryContentsReq.Path = args[1]

		response := w.Sandbox.ListDirectoryContents(ctx, listDirectoryContentsReq)
		if listDirectoryContentsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDirectoryContentsLimit)
		}
		if listDirectoryContentsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listDirectoryContentsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDirectoryContentsOverrides {
		fn(cmd, &listDirectoryContentsReq)
	}

	return cmd
}

// start list-sandboxes command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSandboxesOverrides []func(
	*cobra.Command,
	*sandbox.ListSandboxesRequest,
)

func newListSandboxes() *cobra.Command {
	cmd := &cobra.Command{}

	var listSandboxesReq sandbox.ListSandboxesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSandboxesLimit int

	cmd.Flags().IntVar(&listSandboxesReq.PageSize, "page-size", listSandboxesReq.PageSize, ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSandboxesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listSandboxesReq.PageToken, "page-token", listSandboxesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-sandboxes"
	cmd.Short = `*Beta* List sandboxes.`
	cmd.Long = `This command is in Beta and may change without notice.

List sandboxes.

  Lists all Sandboxes.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Sandbox.ListSandboxes(ctx, listSandboxesReq)
		if listSandboxesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSandboxesLimit)
		}
		if listSandboxesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSandboxesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSandboxesOverrides {
		fn(cmd, &listSandboxesReq)
	}

	return cmd
}

// start rename-path command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var renamePathOverrides []func(
	*cobra.Command,
	*sandbox.RenamePathRequest,
)

func newRenamePath() *cobra.Command {
	cmd := &cobra.Command{}

	var renamePathReq sandbox.RenamePathRequest
	var renamePathJson flags.JsonFlag

	cmd.Flags().Var(&renamePathJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "rename-path PARENT SOURCE DESTINATION"
	cmd.Short = `.`
	cmd.Long = `.

  Renames a path within a sandbox. The destination must be absent and on the
  same filesystem.

  Arguments:
    PARENT: Sandbox resource name, for example sandboxes/my-sandbox.
    SOURCE: Absolute UTF-8 path to move.
    DESTINATION: Exact absolute UTF-8 destination; it must not already exist.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'source', 'destination' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := renamePathJson.Unmarshal(&renamePathReq)
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
		renamePathReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			renamePathReq.Source = args[1]
		}
		if !cmd.Flags().Changed("json") {
			renamePathReq.Destination = args[2]
		}

		response, err := w.Sandbox.RenamePath(ctx, renamePathReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range renamePathOverrides {
		fn(cmd, &renamePathReq)
	}

	return cmd
}

// start start-sandbox command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var startSandboxOverrides []func(
	*cobra.Command,
	*sandbox.StartSandboxRequest,
)

func newStartSandbox() *cobra.Command {
	cmd := &cobra.Command{}

	var startSandboxReq sandbox.StartSandboxRequest

	cmd.Use = "start-sandbox NAME"
	cmd.Short = `*Beta* Start a sandbox.`
	cmd.Long = `This command is in Beta and may change without notice.

Start a sandbox.

  Starts a previously stopped Sandbox under the same sandbox name. Returns
  NOT_FOUND if there is no stopped sandbox to start for the given name.

  Arguments:
    NAME: Resource name of the sandbox to start, in the form
      sandboxes/{sandbox_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		startSandboxReq.Name = args[0]

		response, err := w.Sandbox.StartSandbox(ctx, startSandboxReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range startSandboxOverrides {
		fn(cmd, &startSandboxReq)
	}

	return cmd
}

// start stop-sandbox command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var stopSandboxOverrides []func(
	*cobra.Command,
	*sandbox.StopSandboxRequest,
)

func newStopSandbox() *cobra.Command {
	cmd := &cobra.Command{}

	var stopSandboxReq sandbox.StopSandboxRequest

	cmd.Use = "stop-sandbox NAME"
	cmd.Short = `*Beta* Stop a sandbox.`
	cmd.Long = `This command is in Beta and may change without notice.

Stop a sandbox.

  Stops a running Sandbox, preserving it so it can later be restarted with a
  Start request.

  Arguments:
    NAME: Resource name of the sandbox to stop, in the form
      sandboxes/{sandbox_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		stopSandboxReq.Name = args[0]

		response, err := w.Sandbox.StopSandbox(ctx, stopSandboxReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range stopSandboxOverrides {
		fn(cmd, &stopSandboxReq)
	}

	return cmd
}

// start update-sandbox command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateSandboxOverrides []func(
	*cobra.Command,
	*sandbox.UpdateSandboxRequest,
)

func newUpdateSandbox() *cobra.Command {
	cmd := &cobra.Command{}

	var updateSandboxReq sandbox.UpdateSandboxRequest
	updateSandboxReq.Sandbox = sandbox.Sandbox{}
	var updateSandboxJson flags.JsonFlag

	cmd.Flags().Var(&updateSandboxJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateSandboxReq.Sandbox.DisplayName, "display-name", updateSandboxReq.Sandbox.DisplayName, `Human-readable display label for the sandbox.`)
	// TODO: complex arg: metadata
	cmd.Flags().StringVar(&updateSandboxReq.Sandbox.Name, "name", updateSandboxReq.Sandbox.Name, `The sandbox resource name, in the form sandboxes/{sandbox_id}.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "update-sandbox NAME UPDATE_MASK"
	cmd.Short = `*Beta* Update a sandbox.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a sandbox.

  Updates mutable fields on an existing Sandbox. Allowlisted update_mask paths
  today: display_name, spec.compute.inactivity_timeout. Returns
  INVALID_PARAMETER_VALUE for empty masks or unknown paths; NOT_FOUND if the
  sandbox does not exist.

  Arguments:
    NAME: Resource name of the sandbox to update, in the form
      sandboxes/{sandbox_id}.
    UPDATE_MASK: Field paths to update. Must be a non-empty subset of: - display_name -
      spec.compute.inactivity_timeout Any other path returns
      INVALID_PARAMETER_VALUE.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateSandboxJson.Unmarshal(&updateSandboxReq.Sandbox)
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
		updateSandboxReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateSandboxReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.Sandbox.UpdateSandbox(ctx, updateSandboxReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateSandboxOverrides {
		fn(cmd, &updateSandboxReq)
	}

	return cmd
}

// end service Sandbox
