// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package mason

import (
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/mason"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mason",
		Short: `APIs for managing agent memory and durable session state.`,
		Long: `APIs for managing agent memory and durable session state. This interface is
  under active development and may change.`,
		GroupID: "agentbricks",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	// Add methods
	cmd.AddCommand(newAppendSessionItems())
	cmd.AddCommand(newClearSessionItems())
	cmd.AddCommand(newCreateMemory())
	cmd.AddCommand(newCreateMemoryStore())
	cmd.AddCommand(newCreateSession())
	cmd.AddCommand(newCreateSessionStore())
	cmd.AddCommand(newDeleteMemory())
	cmd.AddCommand(newDeleteMemoryStore())
	cmd.AddCommand(newDeleteSession())
	cmd.AddCommand(newDeleteSessionStore())
	cmd.AddCommand(newExtractMemories())
	cmd.AddCommand(newForkSession())
	cmd.AddCommand(newGetMemory())
	cmd.AddCommand(newGetMemoryStore())
	cmd.AddCommand(newGetSession())
	cmd.AddCommand(newGetSessionStore())
	cmd.AddCommand(newListMemories())
	cmd.AddCommand(newListMemoryStores())
	cmd.AddCommand(newListSessionItems())
	cmd.AddCommand(newListSessionStores())
	cmd.AddCommand(newListSessions())
	cmd.AddCommand(newPopSessionItem())
	cmd.AddCommand(newSearchMemories())
	cmd.AddCommand(newUpdateMemory())
	cmd.AddCommand(newUpdateMemoryStore())
	cmd.AddCommand(newUpdateSession())
	cmd.AddCommand(newUpdateSessionStore())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start append-session-items command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var appendSessionItemsOverrides []func(
	*cobra.Command,
	*mason.AppendSessionItemsRequest,
)

func newAppendSessionItems() *cobra.Command {
	cmd := &cobra.Command{}

	var appendSessionItemsReq mason.AppendSessionItemsRequest
	var appendSessionItemsJson flags.JsonFlag

	cmd.Flags().Var(&appendSessionItemsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "append-session-items PARENT"
	cmd.Short = `Append items to a session.`
	cmd.Long = `Append items to a session.

  Appends items to a session.

  Arguments:
    PARENT: Resource name of the containing session, in the form
      session-stores/{session_store_id}/sessions/{session_id}.`

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
			diags := appendSessionItemsJson.Unmarshal(&appendSessionItemsReq)
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
		appendSessionItemsReq.Parent = args[0]

		response, err := w.Mason.AppendSessionItems(ctx, appendSessionItemsReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range appendSessionItemsOverrides {
		fn(cmd, &appendSessionItemsReq)
	}

	return cmd
}

// start clear-session-items command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var clearSessionItemsOverrides []func(
	*cobra.Command,
	*mason.ClearSessionItemsRequest,
)

func newClearSessionItems() *cobra.Command {
	cmd := &cobra.Command{}

	var clearSessionItemsReq mason.ClearSessionItemsRequest

	cmd.Use = "clear-session-items PARENT"
	cmd.Short = `Clear items from a session.`
	cmd.Long = `Clear items from a session.

  Clears all items from a session.

  Arguments:
    PARENT: Resource name of the containing session, in the form
      session-stores/{session_store_id}/sessions/{session_id}.`

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

		clearSessionItemsReq.Parent = args[0]

		response, err := w.Mason.ClearSessionItems(ctx, clearSessionItemsReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range clearSessionItemsOverrides {
		fn(cmd, &clearSessionItemsReq)
	}

	return cmd
}

// start create-memory command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createMemoryOverrides []func(
	*cobra.Command,
	*mason.CreateManagedMemoryEntryRequest,
)

func newCreateMemory() *cobra.Command {
	cmd := &cobra.Command{}

	var createMemoryReq mason.CreateManagedMemoryEntryRequest
	createMemoryReq.ManagedMemoryEntry = mason.ManagedMemoryEntry{}
	var createMemoryJson flags.JsonFlag

	cmd.Flags().Var(&createMemoryJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createMemoryReq.ManagedMemoryEntryId, "managed-memory-entry-id", createMemoryReq.ManagedMemoryEntryId, `Optional caller-selected managed memory entry ID.`)
	cmd.Flags().StringVar(&createMemoryReq.ManagedMemoryEntry.Content, "content", createMemoryReq.ManagedMemoryEntry.Content, `Optional free-form memory content.`)
	cmd.Flags().StringVar(&createMemoryReq.ManagedMemoryEntry.Description, "description", createMemoryReq.ManagedMemoryEntry.Description, `Human-readable description of the memory entry.`)
	cmd.Flags().StringVar(&createMemoryReq.ManagedMemoryEntry.SessionId, "session-id", createMemoryReq.ManagedMemoryEntry.SessionId, `Optional identifier for the session associated with this memory entry.`)
	cmd.Flags().Var(&createMemoryReq.ManagedMemoryEntry.SourceType, "source-type", `Which writer created this entry. Supported values: [MANAGED_MEMORY_ENTRY_SOURCE_TYPE_AGENT, MANAGED_MEMORY_ENTRY_SOURCE_TYPE_DREAMER]`)

	cmd.Use = "create-memory PARENT ACTOR_ID PATH"
	cmd.Short = `Create a managed memory entry.`
	cmd.Long = `Create a managed memory entry.

  Creates a managed memory entry using exclusive-create semantics. Callers may
  choose the entry ID; the service generates one when it is omitted. Returns
  ALREADY_EXISTS when an entry with the same actor, session, and path already
  exists. Omitted session_id is its own uniqueness key: two omitted-session
  entries with the same actor and path conflict, but an omitted-session entry
  does not conflict with a session-scoped entry at the same actor and path.

  Arguments:
    PARENT: Managed memory store that will contain the entry, in the form
      memory-stores/{managed_memory_store_id}.
    ACTOR_ID: Customer-provided identifier for the actor whose memory this entry
      represents.
    PATH: Absolute, case-sensitive path identifying the entry within its actor and
      optional session. Paths must begin with / and must not contain empty,
      . or .. segments.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'actor_id', 'path' in your JSON input")
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
			diags := createMemoryJson.Unmarshal(&createMemoryReq.ManagedMemoryEntry)
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
		createMemoryReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			createMemoryReq.ManagedMemoryEntry.ActorId = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createMemoryReq.ManagedMemoryEntry.Path = args[2]
		}

		response, err := w.Mason.CreateMemory(ctx, createMemoryReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createMemoryOverrides {
		fn(cmd, &createMemoryReq)
	}

	return cmd
}

// start create-memory-store command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createMemoryStoreOverrides []func(
	*cobra.Command,
	*mason.CreateManagedMemoryStoreRequest,
)

func newCreateMemoryStore() *cobra.Command {
	cmd := &cobra.Command{}

	var createMemoryStoreReq mason.CreateManagedMemoryStoreRequest
	createMemoryStoreReq.ManagedMemoryStore = mason.ManagedMemoryStore{}
	var createMemoryStoreJson flags.JsonFlag

	cmd.Flags().Var(&createMemoryStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createMemoryStoreReq.ManagedMemoryStore.Description, "description", createMemoryStoreReq.ManagedMemoryStore.Description, `Human-readable description of the memory store.`)
	cmd.Flags().StringVar(&createMemoryStoreReq.ManagedMemoryStore.DisplayName, "display-name", createMemoryStoreReq.ManagedMemoryStore.DisplayName, `Deprecated compatibility alias for the caller-provided managed memory store ID.`)
	// TODO: complex arg: storage_backend

	cmd.Use = "create-memory-store MANAGED_MEMORY_STORE_ID"
	cmd.Short = `Create a managed memory store.`
	cmd.Long = `Create a managed memory store.

  Creates a managed memory store in the caller's workspace.

  Arguments:
    MANAGED_MEMORY_STORE_ID: Caller-provided, workspace-unique managed memory store ID. It must be 3-56
      characters, begin with a lowercase letter, contain only lowercase letters,
      digits, and hyphens, and end with a letter or digit.`

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
			diags := createMemoryStoreJson.Unmarshal(&createMemoryStoreReq.ManagedMemoryStore)
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
		createMemoryStoreReq.ManagedMemoryStoreId = args[0]

		response, err := w.Mason.CreateMemoryStore(ctx, createMemoryStoreReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createMemoryStoreOverrides {
		fn(cmd, &createMemoryStoreReq)
	}

	return cmd
}

// start create-session command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSessionOverrides []func(
	*cobra.Command,
	*mason.CreateSessionRequest,
)

func newCreateSession() *cobra.Command {
	cmd := &cobra.Command{}

	var createSessionReq mason.CreateSessionRequest
	createSessionReq.Session = mason.Session{}
	var createSessionJson flags.JsonFlag

	cmd.Flags().Var(&createSessionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createSessionReq.SessionId, "session-id", createSessionReq.SessionId, `Optional caller-selected session ID.`)
	// TODO: map via StringToStringVar: metadata
	cmd.Flags().StringVar(&createSessionReq.Session.ParentSessionId, "parent-session-id", createSessionReq.Session.ParentSessionId, `Immediate parent session ID.`)

	cmd.Use = "create-session PARENT ACTOR_ID"
	cmd.Short = `Create a session.`
	cmd.Long = `Create a session.

  Creates a session within a session store.

  Arguments:
    PARENT: Resource name of the containing session store, in the form
      session-stores/{session_store_id}.
    ACTOR_ID: Opaque caller-provided identifier for the application actor associated
      with the session.

      This is application data and has no Databricks authentication or
      authorization semantics. Use the same value as the Managed Memory Entry
      actor_id when storing memories associated with this actor. Every session
      must set it. A child session must use the same value as its parent.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'actor_id' in your JSON input")
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
			diags := createSessionJson.Unmarshal(&createSessionReq.Session)
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
		createSessionReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			createSessionReq.Session.ActorId = args[1]
		}

		response, err := w.Mason.CreateSession(ctx, createSessionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createSessionOverrides {
		fn(cmd, &createSessionReq)
	}

	return cmd
}

// start create-session-store command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSessionStoreOverrides []func(
	*cobra.Command,
	*mason.CreateSessionStoreRequest,
)

func newCreateSessionStore() *cobra.Command {
	cmd := &cobra.Command{}

	var createSessionStoreReq mason.CreateSessionStoreRequest
	createSessionStoreReq.SessionStore = mason.SessionStore{}
	var createSessionStoreJson flags.JsonFlag

	cmd.Flags().Var(&createSessionStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createSessionStoreReq.SessionStore.Description, "description", createSessionStoreReq.SessionStore.Description, `Human-readable description of the session store.`)
	// TODO: map via StringToStringVar: metadata

	cmd.Use = "create-session-store SESSION_STORE_ID"
	cmd.Short = `Create a session store.`
	cmd.Long = `Create a session store.

  Creates a session store.

  Arguments:
    SESSION_STORE_ID: Caller-provided, workspace-unique session store ID. It must be 3-55
      characters, begin with a lowercase letter, and contain only lowercase
      letters, digits, and hyphens.`

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
			diags := createSessionStoreJson.Unmarshal(&createSessionStoreReq.SessionStore)
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
		createSessionStoreReq.SessionStoreId = args[0]

		response, err := w.Mason.CreateSessionStore(ctx, createSessionStoreReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createSessionStoreOverrides {
		fn(cmd, &createSessionStoreReq)
	}

	return cmd
}

// start delete-memory command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteMemoryOverrides []func(
	*cobra.Command,
	*mason.DeleteManagedMemoryEntryRequest,
)

func newDeleteMemory() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteMemoryReq mason.DeleteManagedMemoryEntryRequest

	cmd.Use = "delete-memory NAME"
	cmd.Short = `Delete a managed memory entry by resource name.`
	cmd.Long = `Delete a managed memory entry by resource name.

  Deletes a managed memory entry by resource name. Returns NOT_FOUND when the
  entry does not exist in the caller's workspace.

  Arguments:
    NAME: Resource name in the form
      memory-stores/{managed_memory_store_id}/entries/{managed_memory_entry_id}.`

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

		deleteMemoryReq.Name = args[0]

		err = w.Mason.DeleteMemory(ctx, deleteMemoryReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteMemoryOverrides {
		fn(cmd, &deleteMemoryReq)
	}

	return cmd
}

// start delete-memory-store command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteMemoryStoreOverrides []func(
	*cobra.Command,
	*mason.DeleteManagedMemoryStoreRequest,
)

func newDeleteMemoryStore() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteMemoryStoreReq mason.DeleteManagedMemoryStoreRequest

	cmd.Use = "delete-memory-store NAME"
	cmd.Short = `Delete a managed memory store by resource name.`
	cmd.Long = `Delete a managed memory store by resource name.

  Deletes a managed memory store by resource name. Returns NOT_FOUND when the
  store does not exist in the caller's workspace.

  Arguments:
    NAME: Resource name in the form memory-stores/{managed_memory_store_id}.`

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

		deleteMemoryStoreReq.Name = args[0]

		err = w.Mason.DeleteMemoryStore(ctx, deleteMemoryStoreReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteMemoryStoreOverrides {
		fn(cmd, &deleteMemoryStoreReq)
	}

	return cmd
}

// start delete-session command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSessionOverrides []func(
	*cobra.Command,
	*mason.DeleteSessionRequest,
)

func newDeleteSession() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteSessionReq mason.DeleteSessionRequest

	cmd.Use = "delete-session NAME"
	cmd.Short = `Delete a session.`
	cmd.Long = `Delete a session.

  Deletes a session, its items, and any descendant sessions recursively.
  Independently retained memory is not deleted.

  Arguments:
    NAME: Resource name in the form
      session-stores/{session_store_id}/sessions/{session_id}.`

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

		deleteSessionReq.Name = args[0]

		err = w.Mason.DeleteSession(ctx, deleteSessionReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSessionOverrides {
		fn(cmd, &deleteSessionReq)
	}

	return cmd
}

// start delete-session-store command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSessionStoreOverrides []func(
	*cobra.Command,
	*mason.DeleteSessionStoreRequest,
)

func newDeleteSessionStore() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteSessionStoreReq mason.DeleteSessionStoreRequest

	cmd.Use = "delete-session-store NAME"
	cmd.Short = `Delete a session store.`
	cmd.Long = `Delete a session store.

  Deletes a session store, its sessions and items, and its service-managed
  storage. Memory entries retained by a separate Memory Store are not deleted.

  Arguments:
    NAME: Resource name in the form session-stores/{session_store_id}.`

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

		deleteSessionStoreReq.Name = args[0]

		err = w.Mason.DeleteSessionStore(ctx, deleteSessionStoreReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSessionStoreOverrides {
		fn(cmd, &deleteSessionStoreReq)
	}

	return cmd
}

// start extract-memories command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var extractMemoriesOverrides []func(
	*cobra.Command,
	*mason.ExtractMemoriesRequest,
)

func newExtractMemories() *cobra.Command {
	cmd := &cobra.Command{}

	var extractMemoriesReq mason.ExtractMemoriesRequest
	var extractMemoriesJson flags.JsonFlag

	cmd.Flags().Var(&extractMemoriesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&extractMemoriesReq.DryRun, "dry-run", extractMemoriesReq.DryRun, `When true, extract and return the entries without writing them to the memory store.`)
	cmd.Flags().StringVar(&extractMemoriesReq.Instructions, "instructions", extractMemoriesReq.Instructions, `Instructions steering what is extracted from the session.`)

	cmd.Use = "extract-memories SESSION_STORE SESSION_ID MEMORY_STORE"
	cmd.Short = `Extract memories from a session.`
	cmd.Long = `Extract memories from a session.

  Synchronously extracts memories from a single session into the given memory
  store, returning the entries that were written.

  Arguments:
    SESSION_STORE: Session store containing the session, in the form
      session-stores/{session_store_id}.
    SESSION_ID: Identifier of the session whose transcript is distilled into memories.
    MEMORY_STORE: Managed memory store the extracted entries are written to, in the form
      memory-stores/{managed_memory_store_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only SESSION_STORE, SESSION_ID as positional arguments. Provide 'memory_store' in your JSON input")
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
			diags := extractMemoriesJson.Unmarshal(&extractMemoriesReq)
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
		extractMemoriesReq.SessionStore = args[0]
		extractMemoriesReq.SessionId = args[1]
		if !cmd.Flags().Changed("json") {
			extractMemoriesReq.MemoryStore = args[2]
		}

		response, err := w.Mason.ExtractMemories(ctx, extractMemoriesReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range extractMemoriesOverrides {
		fn(cmd, &extractMemoriesReq)
	}

	return cmd
}

// start fork-session command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var forkSessionOverrides []func(
	*cobra.Command,
	*mason.ForkSessionRequest,
)

func newForkSession() *cobra.Command {
	cmd := &cobra.Command{}

	var forkSessionReq mason.ForkSessionRequest
	var forkSessionJson flags.JsonFlag

	cmd.Flags().Var(&forkSessionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: map via StringToStringVar: metadata
	cmd.Flags().StringVar(&forkSessionReq.SessionId, "session-id", forkSessionReq.SessionId, `Optional unique ID for the forked session.`)
	cmd.Flags().StringVar(&forkSessionReq.UpToItemId, "up-to-item-id", forkSessionReq.UpToItemId, `Optional last item ID to copy through, inclusively.`)

	cmd.Use = "fork-session PARENT SOURCE_SESSION_ID ACTOR_ID"
	cmd.Short = `Fork a session.`
	cmd.Long = `Fork a session.

  Forks a session into an independent top-level copy.

  Arguments:
    PARENT: Resource name of the containing session store, in the form
      session-stores/{session_store_id}.
    SOURCE_SESSION_ID: ID of the session to copy.
    ACTOR_ID: Opaque caller-provided identifier for the application actor associated
      with the forked session.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'source_session_id', 'actor_id' in your JSON input")
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
			diags := forkSessionJson.Unmarshal(&forkSessionReq)
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
		forkSessionReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			forkSessionReq.SourceSessionId = args[1]
		}
		if !cmd.Flags().Changed("json") {
			forkSessionReq.ActorId = args[2]
		}

		response, err := w.Mason.ForkSession(ctx, forkSessionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range forkSessionOverrides {
		fn(cmd, &forkSessionReq)
	}

	return cmd
}

// start get-memory command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMemoryOverrides []func(
	*cobra.Command,
	*mason.GetManagedMemoryEntryRequest,
)

func newGetMemory() *cobra.Command {
	cmd := &cobra.Command{}

	var getMemoryReq mason.GetManagedMemoryEntryRequest

	var readMaskParam string
	cmd.Flags().StringVar(&readMaskParam, "read-mask", readMaskParam, `Fields to return, using proto field names such as content (not contents). Wire name: 'read_mask'.`)

	cmd.Use = "get-memory NAME"
	cmd.Short = `Get a managed memory entry by resource name.`
	cmd.Long = `Get a managed memory entry by resource name.

  Retrieves a managed memory entry, including its content, by resource name.
  Returns NOT_FOUND when the entry does not exist in the caller's workspace.

  Arguments:
    NAME: Resource name in the form
      memory-stores/{managed_memory_store_id}/entries/{managed_memory_entry_id}.`

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

		getMemoryReq.Name = args[0]

		if readMaskParam != "" {
			readMaskArray := strings.Split(readMaskParam, ",")
			getMemoryReq.ReadMask = fieldmask.New(readMaskArray)
		}

		response, err := w.Mason.GetMemory(ctx, getMemoryReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getMemoryOverrides {
		fn(cmd, &getMemoryReq)
	}

	return cmd
}

// start get-memory-store command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMemoryStoreOverrides []func(
	*cobra.Command,
	*mason.GetManagedMemoryStoreRequest,
)

func newGetMemoryStore() *cobra.Command {
	cmd := &cobra.Command{}

	var getMemoryStoreReq mason.GetManagedMemoryStoreRequest

	cmd.Use = "get-memory-store NAME"
	cmd.Short = `Get a managed memory store by resource name.`
	cmd.Long = `Get a managed memory store by resource name.

  Retrieves a managed memory store by resource name. Returns NOT_FOUND when
  the store does not exist in the caller's workspace.

  Arguments:
    NAME: Resource name in the form memory-stores/{managed_memory_store_id}.`

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

		getMemoryStoreReq.Name = args[0]

		response, err := w.Mason.GetMemoryStore(ctx, getMemoryStoreReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getMemoryStoreOverrides {
		fn(cmd, &getMemoryStoreReq)
	}

	return cmd
}

// start get-session command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSessionOverrides []func(
	*cobra.Command,
	*mason.GetSessionRequest,
)

func newGetSession() *cobra.Command {
	cmd := &cobra.Command{}

	var getSessionReq mason.GetSessionRequest

	cmd.Use = "get-session NAME"
	cmd.Short = `Get a session.`
	cmd.Long = `Get a session.

  Gets a session by resource name.

  Arguments:
    NAME: Resource name in the form
      session-stores/{session_store_id}/sessions/{session_id}.`

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

		getSessionReq.Name = args[0]

		response, err := w.Mason.GetSession(ctx, getSessionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSessionOverrides {
		fn(cmd, &getSessionReq)
	}

	return cmd
}

// start get-session-store command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSessionStoreOverrides []func(
	*cobra.Command,
	*mason.GetSessionStoreRequest,
)

func newGetSessionStore() *cobra.Command {
	cmd := &cobra.Command{}

	var getSessionStoreReq mason.GetSessionStoreRequest

	cmd.Use = "get-session-store NAME"
	cmd.Short = `Get a session store.`
	cmd.Long = `Get a session store.

  Gets a session store by resource name.

  Arguments:
    NAME: Resource name in the form session-stores/{session_store_id}.`

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

		getSessionStoreReq.Name = args[0]

		response, err := w.Mason.GetSessionStore(ctx, getSessionStoreReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSessionStoreOverrides {
		fn(cmd, &getSessionStoreReq)
	}

	return cmd
}

// start list-memories command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listMemoriesOverrides []func(
	*cobra.Command,
	*mason.ListManagedMemoryEntriesRequest,
)

func newListMemories() *cobra.Command {
	cmd := &cobra.Command{}

	var listMemoriesReq mason.ListManagedMemoryEntriesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listMemoriesLimit int

	cmd.Flags().IntVar(&listMemoriesReq.PageSize, "page-size", listMemoriesReq.PageSize, `Maximum number of entries to return. Wire name: 'page_size'.`)
	cmd.Flags().StringVar(&listMemoriesReq.PathPrefix, "path-prefix", listMemoriesReq.PathPrefix, `Optional path prefix used to restrict entries within the actor partition. Wire name: 'path_prefix'.`)
	var readMaskParam string
	cmd.Flags().StringVar(&readMaskParam, "read-mask", readMaskParam, `Fields to return in each entry, using proto field names such as content (not contents). Wire name: 'read_mask'.`)
	cmd.Flags().StringVar(&listMemoriesReq.SessionId, "session-id", listMemoriesReq.SessionId, `Optional session identifier. Wire name: 'session_id'.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listMemoriesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listMemoriesReq.PageToken, "page-token", listMemoriesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-memories PARENT ACTOR_ID"
	cmd.Short = `List managed memory entries.`
	cmd.Long = `List managed memory entries.

  Lists managed memory entries for one actor. An exact path filters entries
  across sessions, ignoring session metadata. Otherwise, session_id and
  path_prefix restrict the actor partition. read_mask selects fields in each
  returned entry.

  Arguments:
    PARENT: Managed memory store whose entries are listed, in the form
      memory-stores/{managed_memory_store_id}.
    ACTOR_ID: Customer-provided identifier for the actor whose entries are listed.`

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

		listMemoriesReq.Parent = args[0]
		listMemoriesReq.ActorId = args[1]

		if readMaskParam != "" {
			readMaskArray := strings.Split(readMaskParam, ",")
			listMemoriesReq.ReadMask = fieldmask.New(readMaskArray)
		}

		response := w.Mason.ListMemories(ctx, listMemoriesReq)
		if listMemoriesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listMemoriesLimit)
		}
		if listMemoriesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listMemoriesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listMemoriesOverrides {
		fn(cmd, &listMemoriesReq)
	}

	return cmd
}

// start list-memory-stores command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listMemoryStoresOverrides []func(
	*cobra.Command,
	*mason.ListManagedMemoryStoresRequest,
)

func newListMemoryStores() *cobra.Command {
	cmd := &cobra.Command{}

	var listMemoryStoresReq mason.ListManagedMemoryStoresRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listMemoryStoresLimit int

	cmd.Flags().IntVar(&listMemoryStoresReq.PageSize, "page-size", listMemoryStoresReq.PageSize, `Maximum number of stores to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listMemoryStoresLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listMemoryStoresReq.PageToken, "page-token", listMemoryStoresReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-memory-stores"
	cmd.Short = `List managed memory stores.`
	cmd.Long = `List managed memory stores.

  Lists managed memory stores in the caller's workspace.`

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

		response := w.Mason.ListMemoryStores(ctx, listMemoryStoresReq)
		if listMemoryStoresLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listMemoryStoresLimit)
		}
		if listMemoryStoresLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listMemoryStoresLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listMemoryStoresOverrides {
		fn(cmd, &listMemoryStoresReq)
	}

	return cmd
}

// start list-session-items command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSessionItemsOverrides []func(
	*cobra.Command,
	*mason.ListSessionItemsRequest,
)

func newListSessionItems() *cobra.Command {
	cmd := &cobra.Command{}

	var listSessionItemsReq mason.ListSessionItemsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSessionItemsLimit int

	cmd.Flags().StringVar(&listSessionItemsReq.OrderBy, "order-by", listSessionItemsReq.OrderBy, `Sort order.`)
	cmd.Flags().IntVar(&listSessionItemsReq.PageSize, "page-size", listSessionItemsReq.PageSize, `Maximum number of items to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSessionItemsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listSessionItemsReq.PageToken, "page-token", listSessionItemsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-session-items PARENT"
	cmd.Short = `List session items.`
	cmd.Long = `List session items.

  Lists items in a session.

  Arguments:
    PARENT: Resource name of the containing session, in the form
      session-stores/{session_store_id}/sessions/{session_id}.`

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

		listSessionItemsReq.Parent = args[0]

		response := w.Mason.ListSessionItems(ctx, listSessionItemsReq)
		if listSessionItemsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSessionItemsLimit)
		}
		if listSessionItemsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSessionItemsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSessionItemsOverrides {
		fn(cmd, &listSessionItemsReq)
	}

	return cmd
}

// start list-session-stores command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSessionStoresOverrides []func(
	*cobra.Command,
	*mason.ListSessionStoresRequest,
)

func newListSessionStores() *cobra.Command {
	cmd := &cobra.Command{}

	var listSessionStoresReq mason.ListSessionStoresRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSessionStoresLimit int

	cmd.Flags().IntVar(&listSessionStoresReq.PageSize, "page-size", listSessionStoresReq.PageSize, `Maximum number of session stores to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSessionStoresLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listSessionStoresReq.PageToken, "page-token", listSessionStoresReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-session-stores"
	cmd.Short = `List session stores.`
	cmd.Long = `List session stores.

  Lists session stores.`

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

		response := w.Mason.ListSessionStores(ctx, listSessionStoresReq)
		if listSessionStoresLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSessionStoresLimit)
		}
		if listSessionStoresLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSessionStoresLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSessionStoresOverrides {
		fn(cmd, &listSessionStoresReq)
	}

	return cmd
}

// start list-sessions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSessionsOverrides []func(
	*cobra.Command,
	*mason.ListSessionsRequest,
)

func newListSessions() *cobra.Command {
	cmd := &cobra.Command{}

	var listSessionsReq mason.ListSessionsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSessionsLimit int

	cmd.Flags().StringVar(&listSessionsReq.Filter, "filter", listSessionsReq.Filter, `Filter expression.`)
	cmd.Flags().StringVar(&listSessionsReq.OrderBy, "order-by", listSessionsReq.OrderBy, `Sort order.`)
	cmd.Flags().IntVar(&listSessionsReq.PageSize, "page-size", listSessionsReq.PageSize, `Maximum number of sessions to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSessionsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listSessionsReq.PageToken, "page-token", listSessionsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-sessions PARENT"
	cmd.Short = `List sessions.`
	cmd.Long = `List sessions.

  Lists sessions within a session store.

  Arguments:
    PARENT: Resource name of the containing session store, in the form
      session-stores/{session_store_id}.`

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

		listSessionsReq.Parent = args[0]

		response := w.Mason.ListSessions(ctx, listSessionsReq)
		if listSessionsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSessionsLimit)
		}
		if listSessionsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSessionsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSessionsOverrides {
		fn(cmd, &listSessionsReq)
	}

	return cmd
}

// start pop-session-item command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var popSessionItemOverrides []func(
	*cobra.Command,
	*mason.PopSessionItemRequest,
)

func newPopSessionItem() *cobra.Command {
	cmd := &cobra.Command{}

	var popSessionItemReq mason.PopSessionItemRequest

	cmd.Use = "pop-session-item PARENT"
	cmd.Short = `Pop an item from a session.`
	cmd.Long = `Pop an item from a session.

  Pops the newest item from a session.

  Arguments:
    PARENT: Resource name of the containing session, in the form
      session-stores/{session_store_id}/sessions/{session_id}.`

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

		popSessionItemReq.Parent = args[0]

		response, err := w.Mason.PopSessionItem(ctx, popSessionItemReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range popSessionItemOverrides {
		fn(cmd, &popSessionItemReq)
	}

	return cmd
}

// start search-memories command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var searchMemoriesOverrides []func(
	*cobra.Command,
	*mason.SearchManagedMemoryEntriesRequest,
)

func newSearchMemories() *cobra.Command {
	cmd := &cobra.Command{}

	var searchMemoriesReq mason.SearchManagedMemoryEntriesRequest
	var searchMemoriesJson flags.JsonFlag
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var searchMemoriesLimit int

	cmd.Flags().Var(&searchMemoriesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&searchMemoriesReq.PageSize, "page-size", searchMemoriesReq.PageSize, `Maximum number of relevance-ranked entries to return. Wire name: 'page_size'.`)
	cmd.Flags().StringVar(&searchMemoriesReq.PathPrefix, "path-prefix", searchMemoriesReq.PathPrefix, `Optional absolute, case-sensitive path prefix used to restrict searched entries within the actor partition. Wire name: 'path_prefix'.`)
	var readMaskParam string
	cmd.Flags().StringVar(&readMaskParam, "read-mask", readMaskParam, `Fields to return in each matching entry, using proto field names such as content (not contents). Wire name: 'read_mask'.`)
	cmd.Flags().StringVar(&searchMemoriesReq.SessionId, "session-id", searchMemoriesReq.SessionId, `Optional session identifier. Wire name: 'session_id'.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&searchMemoriesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&searchMemoriesReq.PageToken, "page-token", searchMemoriesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "search-memories PARENT ACTOR_ID QUERY"
	cmd.Short = `Search managed memory entries.`
	cmd.Long = `Search managed memory entries.

  Searches managed memory entries by text query for one actor. Returns matching
  entries and scores ranked by relevance; read_mask selects fields in each
  returned entry.

  Arguments:
    PARENT: Managed memory store whose entries are searched, in the form
      memory-stores/{managed_memory_store_id}.
    ACTOR_ID: Customer-provided identifier for the actor whose entries are searched.
    QUERY: Free-form search query.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'actor_id', 'query' in your JSON input")
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
			diags := searchMemoriesJson.Unmarshal(&searchMemoriesReq)
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
		searchMemoriesReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			searchMemoriesReq.ActorId = args[1]
		}
		if !cmd.Flags().Changed("json") {
			searchMemoriesReq.Query = args[2]
		}

		if readMaskParam != "" {
			readMaskArray := strings.Split(readMaskParam, ",")
			searchMemoriesReq.ReadMask = fieldmask.New(readMaskArray)
		}

		response := w.Mason.SearchMemories(ctx, searchMemoriesReq)
		if searchMemoriesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", searchMemoriesLimit)
		}
		if searchMemoriesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, searchMemoriesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range searchMemoriesOverrides {
		fn(cmd, &searchMemoriesReq)
	}

	return cmd
}

// start update-memory command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateMemoryOverrides []func(
	*cobra.Command,
	*mason.UpdateManagedMemoryEntryRequest,
)

func newUpdateMemory() *cobra.Command {
	cmd := &cobra.Command{}

	var updateMemoryReq mason.UpdateManagedMemoryEntryRequest
	updateMemoryReq.ManagedMemoryEntry = mason.ManagedMemoryEntry{}
	var updateMemoryJson flags.JsonFlag

	cmd.Flags().Var(&updateMemoryJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateMemoryReq.ManagedMemoryEntry.Content, "content", updateMemoryReq.ManagedMemoryEntry.Content, `Optional free-form memory content.`)
	cmd.Flags().StringVar(&updateMemoryReq.ManagedMemoryEntry.Description, "description", updateMemoryReq.ManagedMemoryEntry.Description, `Human-readable description of the memory entry.`)
	cmd.Flags().StringVar(&updateMemoryReq.ManagedMemoryEntry.SessionId, "session-id", updateMemoryReq.ManagedMemoryEntry.SessionId, `Optional identifier for the session associated with this memory entry.`)
	cmd.Flags().Var(&updateMemoryReq.ManagedMemoryEntry.SourceType, "source-type", `Which writer created this entry. Supported values: [MANAGED_MEMORY_ENTRY_SOURCE_TYPE_AGENT, MANAGED_MEMORY_ENTRY_SOURCE_TYPE_DREAMER]`)

	cmd.Use = "update-memory NAME UPDATE_MASK ACTOR_ID PATH"
	cmd.Short = `Update a managed memory entry.`
	cmd.Long = `Update a managed memory entry.

  Updates selected mutable fields on a managed memory entry. Identity fields are
  immutable.

  Arguments:
    NAME: Resource name in the form
      memory-stores/{managed_memory_store_id}/entries/{managed_memory_entry_id}.
    UPDATE_MASK: Fields to update. Only content and description may be updated.
    ACTOR_ID: Customer-provided identifier for the actor whose memory this entry
      represents.
    PATH: Absolute, case-sensitive path identifying the entry within its actor and
      optional session. Paths must begin with / and must not contain empty,
      . or .. segments.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'actor_id', 'path' in your JSON input")
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
			diags := updateMemoryJson.Unmarshal(&updateMemoryReq.ManagedMemoryEntry)
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
		updateMemoryReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateMemoryReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			updateMemoryReq.ManagedMemoryEntry.ActorId = args[2]
		}
		if !cmd.Flags().Changed("json") {
			updateMemoryReq.ManagedMemoryEntry.Path = args[3]
		}

		response, err := w.Mason.UpdateMemory(ctx, updateMemoryReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateMemoryOverrides {
		fn(cmd, &updateMemoryReq)
	}

	return cmd
}

// start update-memory-store command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateMemoryStoreOverrides []func(
	*cobra.Command,
	*mason.UpdateManagedMemoryStoreRequest,
)

func newUpdateMemoryStore() *cobra.Command {
	cmd := &cobra.Command{}

	var updateMemoryStoreReq mason.UpdateManagedMemoryStoreRequest
	updateMemoryStoreReq.ManagedMemoryStore = mason.ManagedMemoryStore{}
	var updateMemoryStoreJson flags.JsonFlag

	cmd.Flags().Var(&updateMemoryStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateMemoryStoreReq.ManagedMemoryStore.Description, "description", updateMemoryStoreReq.ManagedMemoryStore.Description, `Human-readable description of the memory store.`)
	cmd.Flags().StringVar(&updateMemoryStoreReq.ManagedMemoryStore.DisplayName, "display-name", updateMemoryStoreReq.ManagedMemoryStore.DisplayName, `Deprecated compatibility alias for the caller-provided managed memory store ID.`)
	// TODO: complex arg: storage_backend

	cmd.Use = "update-memory-store NAME UPDATE_MASK"
	cmd.Short = `Update a managed memory store.`
	cmd.Long = `Update a managed memory store.

  Updates a managed memory store's description.

  Arguments:
    NAME: Resource name in the form memory-stores/{managed_memory_store_id}.
    UPDATE_MASK: Only description may be updated.`

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

		if cmd.Flags().Changed("json") {
			diags := updateMemoryStoreJson.Unmarshal(&updateMemoryStoreReq.ManagedMemoryStore)
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
		updateMemoryStoreReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateMemoryStoreReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.Mason.UpdateMemoryStore(ctx, updateMemoryStoreReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateMemoryStoreOverrides {
		fn(cmd, &updateMemoryStoreReq)
	}

	return cmd
}

// start update-session command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateSessionOverrides []func(
	*cobra.Command,
	*mason.UpdateSessionRequest,
)

func newUpdateSession() *cobra.Command {
	cmd := &cobra.Command{}

	var updateSessionReq mason.UpdateSessionRequest
	updateSessionReq.Session = mason.Session{}
	var updateSessionJson flags.JsonFlag

	cmd.Flags().Var(&updateSessionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: map via StringToStringVar: metadata
	cmd.Flags().StringVar(&updateSessionReq.Session.ParentSessionId, "parent-session-id", updateSessionReq.Session.ParentSessionId, `Immediate parent session ID.`)

	cmd.Use = "update-session NAME UPDATE_MASK ACTOR_ID"
	cmd.Short = `Update a session.`
	cmd.Long = `Update a session.

  Updates a session's mutable fields.

  Arguments:
    NAME: Resource name in the form
      session-stores/{session_store_id}/sessions/{session_id}.
    UPDATE_MASK: Fields to update. Only metadata is mutable; any other path returns
      INVALID_PARAMETER_VALUE.
    ACTOR_ID: Opaque caller-provided identifier for the application actor associated
      with the session.

      This is application data and has no Databricks authentication or
      authorization semantics. Use the same value as the Managed Memory Entry
      actor_id when storing memories associated with this actor. Every session
      must set it. A child session must use the same value as its parent.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'actor_id' in your JSON input")
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
			diags := updateSessionJson.Unmarshal(&updateSessionReq.Session)
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
		updateSessionReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateSessionReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			updateSessionReq.Session.ActorId = args[2]
		}

		response, err := w.Mason.UpdateSession(ctx, updateSessionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateSessionOverrides {
		fn(cmd, &updateSessionReq)
	}

	return cmd
}

// start update-session-store command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateSessionStoreOverrides []func(
	*cobra.Command,
	*mason.UpdateSessionStoreRequest,
)

func newUpdateSessionStore() *cobra.Command {
	cmd := &cobra.Command{}

	var updateSessionStoreReq mason.UpdateSessionStoreRequest
	updateSessionStoreReq.SessionStore = mason.SessionStore{}
	var updateSessionStoreJson flags.JsonFlag

	cmd.Flags().Var(&updateSessionStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateSessionStoreReq.SessionStore.Description, "description", updateSessionStoreReq.SessionStore.Description, `Human-readable description of the session store.`)
	// TODO: map via StringToStringVar: metadata

	cmd.Use = "update-session-store NAME UPDATE_MASK"
	cmd.Short = `Update a session store.`
	cmd.Long = `Update a session store.

  Updates a session store's description and metadata.

  Arguments:
    NAME: Resource name in the form session-stores/{session_store_id}.
    UPDATE_MASK: Fields to update. Only description and metadata are mutable; any other
      path returns INVALID_PARAMETER_VALUE.`

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

		if cmd.Flags().Changed("json") {
			diags := updateSessionStoreJson.Unmarshal(&updateSessionStoreReq.SessionStore)
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
		updateSessionStoreReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateSessionStoreReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.Mason.UpdateSessionStore(ctx, updateSessionStoreReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateSessionStoreOverrides {
		fn(cmd, &updateSessionStoreReq)
	}

	return cmd
}

// end service Mason
