// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package agent_kit

import (
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/agentkit"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-kit",
		Short: `*Beta* APIs for managing agent memory and durable session state.`,
		Long: `This command is in Beta and may change without notice.

APIs for managing agent memory and durable session state. This interface is
  under active development and may change.`,
		GroupID: "agentbricks",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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
	cmd.AddCommand(newGetMemoryStorePermission())
	cmd.AddCommand(newGetSession())
	cmd.AddCommand(newGetSessionStore())
	cmd.AddCommand(newGetSessionStorePermission())
	cmd.AddCommand(newGrantMemoryStorePermission())
	cmd.AddCommand(newGrantSessionStorePermission())
	cmd.AddCommand(newListMemories())
	cmd.AddCommand(newListMemoryStorePermissions())
	cmd.AddCommand(newListMemoryStores())
	cmd.AddCommand(newListSessionItems())
	cmd.AddCommand(newListSessionStorePermissions())
	cmd.AddCommand(newListSessionStores())
	cmd.AddCommand(newListSessions())
	cmd.AddCommand(newPopSessionItem())
	cmd.AddCommand(newRevokeMemoryStorePermission())
	cmd.AddCommand(newRevokeSessionStorePermission())
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
	*agentkit.AppendSessionItemsRequest,
)

func newAppendSessionItems() *cobra.Command {
	cmd := &cobra.Command{}

	var appendSessionItemsReq agentkit.AppendSessionItemsRequest
	var appendSessionItemsJson flags.JsonFlag

	cmd.Flags().Var(&appendSessionItemsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "append-session-items PARENT"
	cmd.Short = `*Beta* Append items to a session.`
	cmd.Long = `This command is in Beta and may change without notice.

Append items to a session.

  Appends items to a session.

  Arguments:
    PARENT: Resource name of the containing session, in the form
      session-stores/{session_store_id}/sessions/{session_id}.`

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

		response, err := w.AgentKit.AppendSessionItems(ctx, appendSessionItemsReq)
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
	*agentkit.ClearSessionItemsRequest,
)

func newClearSessionItems() *cobra.Command {
	cmd := &cobra.Command{}

	var clearSessionItemsReq agentkit.ClearSessionItemsRequest

	cmd.Use = "clear-session-items PARENT"
	cmd.Short = `*Beta* Clear items from a session.`
	cmd.Long = `This command is in Beta and may change without notice.

Clear items from a session.

  Clears all items from a session.

  Arguments:
    PARENT: Resource name of the containing session, in the form
      session-stores/{session_store_id}/sessions/{session_id}.`

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

		clearSessionItemsReq.Parent = args[0]

		response, err := w.AgentKit.ClearSessionItems(ctx, clearSessionItemsReq)
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
	*agentkit.CreateManagedMemoryEntryRequest,
)

func newCreateMemory() *cobra.Command {
	cmd := &cobra.Command{}

	var createMemoryReq agentkit.CreateManagedMemoryEntryRequest
	createMemoryReq.ManagedMemoryEntry = agentkit.ManagedMemoryEntry{}
	var createMemoryJson flags.JsonFlag

	cmd.Flags().Var(&createMemoryJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createMemoryReq.ManagedMemoryEntryId, "managed-memory-entry-id", createMemoryReq.ManagedMemoryEntryId, `Optional caller-selected memory entry ID.`)
	cmd.Flags().StringVar(&createMemoryReq.ManagedMemoryEntry.Content, "content", createMemoryReq.ManagedMemoryEntry.Content, `Optional free-form memory content.`)
	cmd.Flags().StringVar(&createMemoryReq.ManagedMemoryEntry.Description, "description", createMemoryReq.ManagedMemoryEntry.Description, `Human-readable description of the memory entry.`)
	cmd.Flags().StringVar(&createMemoryReq.ManagedMemoryEntry.Name, "name", createMemoryReq.ManagedMemoryEntry.Name, `Resource name in the form memory-stores/{managed_memory_store_id}/entries/{managed_memory_entry_id}.`)
	cmd.Flags().StringVar(&createMemoryReq.ManagedMemoryEntry.SessionId, "session-id", createMemoryReq.ManagedMemoryEntry.SessionId, `Optional identifier for the session associated with this memory entry.`)
	cmd.Flags().Var(&createMemoryReq.ManagedMemoryEntry.SourceType, "source-type", `Which writer created this entry. Supported values: [MANAGED_MEMORY_ENTRY_SOURCE_TYPE_AGENT, MANAGED_MEMORY_ENTRY_SOURCE_TYPE_DREAMER]`)

	cmd.Use = "create-memory PARENT ACTOR_ID PATH"
	cmd.Short = `*Beta* Create a memory entry.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a memory entry.

  Creates a memory entry using exclusive-create semantics. Callers may choose
  the entry ID; the service generates one when it is omitted. Returns
  ALREADY_EXISTS when an entry with the same actor, session, and path already
  exists. Omitted session_id is its own uniqueness key: two omitted-session
  entries with the same actor and path conflict, but an omitted-session entry
  does not conflict with a session-scoped entry at the same actor and path.

  Arguments:
    PARENT: Memory store that will contain the entry, in the form
      memory-stores/{managed_memory_store_id}.
    ACTOR_ID: Customer-provided identifier for the actor whose memory this entry
      represents.
    PATH: Absolute, case-sensitive path identifying the entry within its actor and
      optional session. Paths must begin with / and must not contain empty,
      . or .. segments.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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

		response, err := w.AgentKit.CreateMemory(ctx, createMemoryReq)
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
	*agentkit.CreateManagedMemoryStoreRequest,
)

func newCreateMemoryStore() *cobra.Command {
	cmd := &cobra.Command{}

	var createMemoryStoreReq agentkit.CreateManagedMemoryStoreRequest
	createMemoryStoreReq.ManagedMemoryStore = agentkit.ManagedMemoryStore{}
	var createMemoryStoreJson flags.JsonFlag

	cmd.Flags().Var(&createMemoryStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createMemoryStoreReq.ManagedMemoryStore.Description, "description", createMemoryStoreReq.ManagedMemoryStore.Description, `Human-readable description of the memory store.`)
	cmd.Flags().StringVar(&createMemoryStoreReq.ManagedMemoryStore.DisplayName, "display-name", createMemoryStoreReq.ManagedMemoryStore.DisplayName, `Deprecated compatibility alias for the caller-provided memory store ID.`)
	// TODO: complex arg: storage_backend

	cmd.Use = "create-memory-store MANAGED_MEMORY_STORE_ID"
	cmd.Short = `*Beta* Create a memory store.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a memory store.

  Creates a memory store in the caller's workspace.

  Arguments:
    MANAGED_MEMORY_STORE_ID: Caller-provided, workspace-unique memory store ID. It must be 3-56
      characters, begin with a lowercase letter, contain only lowercase letters,
      digits, and hyphens, and end with a letter or digit.`

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

		response, err := w.AgentKit.CreateMemoryStore(ctx, createMemoryStoreReq)
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
	*agentkit.CreateSessionRequest,
)

func newCreateSession() *cobra.Command {
	cmd := &cobra.Command{}

	var createSessionReq agentkit.CreateSessionRequest
	createSessionReq.Session = agentkit.Session{}
	var createSessionJson flags.JsonFlag

	cmd.Flags().Var(&createSessionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createSessionReq.SessionId, "session-id", createSessionReq.SessionId, `Optional caller-selected session ID.`)
	// TODO: map via StringToStringVar: metadata
	cmd.Flags().StringVar(&createSessionReq.Session.ParentSessionId, "parent-session-id", createSessionReq.Session.ParentSessionId, `Immediate parent session ID.`)

	cmd.Use = "create-session PARENT ACTOR_ID"
	cmd.Short = `*Beta* Create a session.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a session.

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
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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

		response, err := w.AgentKit.CreateSession(ctx, createSessionReq)
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
	*agentkit.CreateSessionStoreRequest,
)

func newCreateSessionStore() *cobra.Command {
	cmd := &cobra.Command{}

	var createSessionStoreReq agentkit.CreateSessionStoreRequest
	createSessionStoreReq.SessionStore = agentkit.SessionStore{}
	var createSessionStoreJson flags.JsonFlag

	cmd.Flags().Var(&createSessionStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createSessionStoreReq.SessionStore.Description, "description", createSessionStoreReq.SessionStore.Description, `Human-readable description of the session store.`)
	// TODO: map via StringToStringVar: metadata

	cmd.Use = "create-session-store SESSION_STORE_ID"
	cmd.Short = `*Beta* Create a session store.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a session store.

  Creates a session store.

  Arguments:
    SESSION_STORE_ID: Caller-provided, workspace-unique session store ID. It must be 3-55
      characters, begin with a lowercase letter, and contain only lowercase
      letters, digits, and hyphens.`

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

		response, err := w.AgentKit.CreateSessionStore(ctx, createSessionStoreReq)
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
	*agentkit.DeleteManagedMemoryEntryRequest,
)

func newDeleteMemory() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteMemoryReq agentkit.DeleteManagedMemoryEntryRequest

	cmd.Use = "delete-memory NAME"
	cmd.Short = `*Beta* Delete a memory entry by resource name.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a memory entry by resource name.

  Deletes a memory entry by resource name. Returns NOT_FOUND when the entry
  does not exist in the caller's workspace.

  Arguments:
    NAME: Resource name in the form
      memory-stores/{managed_memory_store_id}/entries/{managed_memory_entry_id}.`

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

		deleteMemoryReq.Name = args[0]

		err = w.AgentKit.DeleteMemory(ctx, deleteMemoryReq)
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
	*agentkit.DeleteManagedMemoryStoreRequest,
)

func newDeleteMemoryStore() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteMemoryStoreReq agentkit.DeleteManagedMemoryStoreRequest

	cmd.Use = "delete-memory-store NAME"
	cmd.Short = `*Beta* Delete a memory store by resource name.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a memory store by resource name.

  Deletes a memory store by resource name. Returns NOT_FOUND when the store
  does not exist in the caller's workspace.

  Arguments:
    NAME: Resource name in the form memory-stores/{managed_memory_store_id}.`

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

		deleteMemoryStoreReq.Name = args[0]

		err = w.AgentKit.DeleteMemoryStore(ctx, deleteMemoryStoreReq)
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
	*agentkit.DeleteSessionRequest,
)

func newDeleteSession() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteSessionReq agentkit.DeleteSessionRequest

	cmd.Use = "delete-session NAME"
	cmd.Short = `*Beta* Delete a session.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a session.

  Deletes a session, its items, and any descendant sessions recursively.
  Independently retained memory is not deleted.

  Arguments:
    NAME: Resource name in the form
      session-stores/{session_store_id}/sessions/{session_id}.`

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

		deleteSessionReq.Name = args[0]

		err = w.AgentKit.DeleteSession(ctx, deleteSessionReq)
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
	*agentkit.DeleteSessionStoreRequest,
)

func newDeleteSessionStore() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteSessionStoreReq agentkit.DeleteSessionStoreRequest

	cmd.Use = "delete-session-store NAME"
	cmd.Short = `*Beta* Delete a session store.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a session store.

  Deletes a session store, its sessions and items, and its service-managed
  storage. Memory entries retained by a separate Memory Store are not deleted.

  Arguments:
    NAME: Resource name in the form session-stores/{session_store_id}.`

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

		deleteSessionStoreReq.Name = args[0]

		err = w.AgentKit.DeleteSessionStore(ctx, deleteSessionStoreReq)
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
	*agentkit.ExtractMemoriesRequest,
)

func newExtractMemories() *cobra.Command {
	cmd := &cobra.Command{}

	var extractMemoriesReq agentkit.ExtractMemoriesRequest
	var extractMemoriesJson flags.JsonFlag

	cmd.Flags().Var(&extractMemoriesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&extractMemoriesReq.DryRun, "dry-run", extractMemoriesReq.DryRun, `When true, extract and return the entries without writing them to the memory store.`)
	cmd.Flags().StringVar(&extractMemoriesReq.Instructions, "instructions", extractMemoriesReq.Instructions, `Instructions steering what is extracted from the session.`)

	cmd.Use = "extract-memories SESSION_STORE SESSION_ID MEMORY_STORE"
	cmd.Short = `*Beta* Extract memories from a session.`
	cmd.Long = `This command is in Beta and may change without notice.

Extract memories from a session.

  Synchronously extracts memories from a single session into the given memory
  store, returning the entries that were written.

  Arguments:
    SESSION_STORE: Session store containing the session, in the form
      session-stores/{session_store_id}.
    SESSION_ID: Identifier of the session whose transcript is distilled into memories.
    MEMORY_STORE: Memory store the extracted entries are written to, in the form
      memory-stores/{managed_memory_store_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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

		response, err := w.AgentKit.ExtractMemories(ctx, extractMemoriesReq)
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
	*agentkit.ForkSessionRequest,
)

func newForkSession() *cobra.Command {
	cmd := &cobra.Command{}

	var forkSessionReq agentkit.ForkSessionRequest
	var forkSessionJson flags.JsonFlag

	cmd.Flags().Var(&forkSessionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: map via StringToStringVar: metadata
	cmd.Flags().StringVar(&forkSessionReq.SessionId, "session-id", forkSessionReq.SessionId, `Optional unique ID for the forked session.`)
	cmd.Flags().StringVar(&forkSessionReq.UpToItemId, "up-to-item-id", forkSessionReq.UpToItemId, `Optional last item ID to copy through, inclusively.`)

	cmd.Use = "fork-session PARENT SOURCE_SESSION_ID ACTOR_ID"
	cmd.Short = `*Beta* Fork a session.`
	cmd.Long = `This command is in Beta and may change without notice.

Fork a session.

  Forks a session into an independent top-level copy.

  Arguments:
    PARENT: Resource name of the containing session store, in the form
      session-stores/{session_store_id}.
    SOURCE_SESSION_ID: ID of the session to copy.
    ACTOR_ID: Opaque caller-provided identifier for the application actor associated
      with the forked session.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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

		response, err := w.AgentKit.ForkSession(ctx, forkSessionReq)
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
	*agentkit.GetManagedMemoryEntryRequest,
)

func newGetMemory() *cobra.Command {
	cmd := &cobra.Command{}

	var getMemoryReq agentkit.GetManagedMemoryEntryRequest

	cmd.Flags().Var(&getMemoryReq.View, "view", `Controls whether the response includes memory content. Supported values: [MANAGED_MEMORY_ENTRY_VIEW_BASIC, MANAGED_MEMORY_ENTRY_VIEW_FULL]`)

	cmd.Use = "get-memory NAME"
	cmd.Short = `*Beta* Get a memory entry by resource name.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a memory entry by resource name.

  Retrieves a memory entry by resource name. view controls whether the
  response includes memory content. Returns NOT_FOUND when the entry does not
  exist in the caller's workspace.

  Arguments:
    NAME: Resource name in the form
      memory-stores/{managed_memory_store_id}/entries/{managed_memory_entry_id}.`

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

		getMemoryReq.Name = args[0]

		response, err := w.AgentKit.GetMemory(ctx, getMemoryReq)
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
	*agentkit.GetManagedMemoryStoreRequest,
)

func newGetMemoryStore() *cobra.Command {
	cmd := &cobra.Command{}

	var getMemoryStoreReq agentkit.GetManagedMemoryStoreRequest

	cmd.Use = "get-memory-store NAME"
	cmd.Short = `*Beta* Get a memory store by resource name.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a memory store by resource name.

  Retrieves a memory store by resource name. Returns NOT_FOUND when the store
  does not exist in the caller's workspace.

  Arguments:
    NAME: Resource name in the form memory-stores/{managed_memory_store_id}.`

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

		getMemoryStoreReq.Name = args[0]

		response, err := w.AgentKit.GetMemoryStore(ctx, getMemoryStoreReq)
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

// start get-memory-store-permission command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMemoryStorePermissionOverrides []func(
	*cobra.Command,
	*agentkit.GetMemoryStorePermissionRequest,
)

func newGetMemoryStorePermission() *cobra.Command {
	cmd := &cobra.Command{}

	var getMemoryStorePermissionReq agentkit.GetMemoryStorePermissionRequest

	cmd.Use = "get-memory-store-permission NAME"
	cmd.Short = `Get memory store permission.`
	cmd.Long = `Get memory store permission.

  Retrieves the current caller's effective memory store permission. Returns
  PERMISSION_LEVEL_UNSPECIFIED when the caller has no access.

  Arguments:
    NAME: Resource name in the form memory-stores/{managed_memory_store_id}.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

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

		getMemoryStorePermissionReq.Name = args[0]

		response, err := w.AgentKit.GetMemoryStorePermission(ctx, getMemoryStorePermissionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getMemoryStorePermissionOverrides {
		fn(cmd, &getMemoryStorePermissionReq)
	}

	return cmd
}

// start get-session command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSessionOverrides []func(
	*cobra.Command,
	*agentkit.GetSessionRequest,
)

func newGetSession() *cobra.Command {
	cmd := &cobra.Command{}

	var getSessionReq agentkit.GetSessionRequest

	cmd.Use = "get-session NAME"
	cmd.Short = `*Beta* Get a session.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a session.

  Gets a session by resource name.

  Arguments:
    NAME: Resource name in the form
      session-stores/{session_store_id}/sessions/{session_id}.`

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

		getSessionReq.Name = args[0]

		response, err := w.AgentKit.GetSession(ctx, getSessionReq)
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
	*agentkit.GetSessionStoreRequest,
)

func newGetSessionStore() *cobra.Command {
	cmd := &cobra.Command{}

	var getSessionStoreReq agentkit.GetSessionStoreRequest

	cmd.Use = "get-session-store NAME"
	cmd.Short = `*Beta* Get a session store.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a session store.

  Gets a session store by resource name.

  Arguments:
    NAME: Resource name in the form session-stores/{session_store_id}.`

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

		getSessionStoreReq.Name = args[0]

		response, err := w.AgentKit.GetSessionStore(ctx, getSessionStoreReq)
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

// start get-session-store-permission command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSessionStorePermissionOverrides []func(
	*cobra.Command,
	*agentkit.GetSessionStorePermissionRequest,
)

func newGetSessionStorePermission() *cobra.Command {
	cmd := &cobra.Command{}

	var getSessionStorePermissionReq agentkit.GetSessionStorePermissionRequest

	cmd.Use = "get-session-store-permission NAME"
	cmd.Short = `Get session store permission.`
	cmd.Long = `Get session store permission.

  Retrieves the current caller's effective session store permission. Returns
  PERMISSION_LEVEL_UNSPECIFIED when the caller has no access.

  Arguments:
    NAME: Resource name in the form session-stores/{session_store_id}.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

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

		getSessionStorePermissionReq.Name = args[0]

		response, err := w.AgentKit.GetSessionStorePermission(ctx, getSessionStorePermissionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSessionStorePermissionOverrides {
		fn(cmd, &getSessionStorePermissionReq)
	}

	return cmd
}

// start grant-memory-store-permission command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var grantMemoryStorePermissionOverrides []func(
	*cobra.Command,
	*agentkit.GrantMemoryStorePermissionRequest,
)

func newGrantMemoryStorePermission() *cobra.Command {
	cmd := &cobra.Command{}

	var grantMemoryStorePermissionReq agentkit.GrantMemoryStorePermissionRequest
	var grantMemoryStorePermissionJson flags.JsonFlag

	cmd.Flags().Var(&grantMemoryStorePermissionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "grant-memory-store-permission NAME"
	cmd.Short = `Grant memory store permission.`
	cmd.Long = `Grant memory store permission.

  Grants a READ or WRITE permission on a memory store to a principal. Only the
  owner or a workspace admin may grant permissions.

  Arguments:
    NAME: Resource name in the form memory-stores/{managed_memory_store_id}.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

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
			diags := grantMemoryStorePermissionJson.Unmarshal(&grantMemoryStorePermissionReq)
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
		grantMemoryStorePermissionReq.Name = args[0]

		response, err := w.AgentKit.GrantMemoryStorePermission(ctx, grantMemoryStorePermissionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range grantMemoryStorePermissionOverrides {
		fn(cmd, &grantMemoryStorePermissionReq)
	}

	return cmd
}

// start grant-session-store-permission command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var grantSessionStorePermissionOverrides []func(
	*cobra.Command,
	*agentkit.GrantSessionStorePermissionRequest,
)

func newGrantSessionStorePermission() *cobra.Command {
	cmd := &cobra.Command{}

	var grantSessionStorePermissionReq agentkit.GrantSessionStorePermissionRequest
	var grantSessionStorePermissionJson flags.JsonFlag

	cmd.Flags().Var(&grantSessionStorePermissionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "grant-session-store-permission NAME"
	cmd.Short = `Grant session store permission.`
	cmd.Long = `Grant session store permission.

  Grants a READ or WRITE permission on a session store to a principal. Only the
  owner or a workspace admin may grant permissions.

  Arguments:
    NAME: Resource name in the form session-stores/{session_store_id}.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

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
			diags := grantSessionStorePermissionJson.Unmarshal(&grantSessionStorePermissionReq)
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
		grantSessionStorePermissionReq.Name = args[0]

		response, err := w.AgentKit.GrantSessionStorePermission(ctx, grantSessionStorePermissionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range grantSessionStorePermissionOverrides {
		fn(cmd, &grantSessionStorePermissionReq)
	}

	return cmd
}

// start list-memories command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listMemoriesOverrides []func(
	*cobra.Command,
	*agentkit.ListManagedMemoryEntriesRequest,
)

func newListMemories() *cobra.Command {
	cmd := &cobra.Command{}

	var listMemoriesReq agentkit.ListManagedMemoryEntriesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listMemoriesLimit int

	cmd.Flags().IntVar(&listMemoriesReq.PageSize, "page-size", listMemoriesReq.PageSize, `Maximum number of entries to return.`)
	cmd.Flags().StringVar(&listMemoriesReq.PathPrefix, "path-prefix", listMemoriesReq.PathPrefix, `Optional path prefix used to restrict entries within the actor partition.`)
	cmd.Flags().StringVar(&listMemoriesReq.SessionId, "session-id", listMemoriesReq.SessionId, `Optional session identifier.`)
	cmd.Flags().Var(&listMemoriesReq.View, "view", `Controls whether each returned entry includes memory content. Supported values: [MANAGED_MEMORY_ENTRY_VIEW_BASIC, MANAGED_MEMORY_ENTRY_VIEW_FULL]`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listMemoriesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listMemoriesReq.PageToken, "page-token", listMemoriesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-memories PARENT ACTOR_ID"
	cmd.Short = `*Beta* List memory entries.`
	cmd.Long = `This command is in Beta and may change without notice.

List memory entries.

  Lists memory entries for one actor. An exact path filters entries across
  sessions, ignoring session metadata. Otherwise, session_id and path_prefix
  restrict the actor partition. view controls whether each returned entry
  includes memory content.

  Arguments:
    PARENT: Memory store whose entries are listed, in the form
      memory-stores/{managed_memory_store_id}.
    ACTOR_ID: Customer-provided identifier for the actor whose entries are listed.`

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

		listMemoriesReq.Parent = args[0]
		listMemoriesReq.ActorId = args[1]

		response := w.AgentKit.ListMemories(ctx, listMemoriesReq)
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

// start list-memory-store-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listMemoryStorePermissionsOverrides []func(
	*cobra.Command,
	*agentkit.ListMemoryStorePermissionsRequest,
)

func newListMemoryStorePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var listMemoryStorePermissionsReq agentkit.ListMemoryStorePermissionsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listMemoryStorePermissionsLimit int

	cmd.Flags().IntVar(&listMemoryStorePermissionsReq.PageSize, "page-size", listMemoryStorePermissionsReq.PageSize, `Maximum number of permissions to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listMemoryStorePermissionsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listMemoryStorePermissionsReq.PageToken, "page-token", listMemoryStorePermissionsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-memory-store-permissions PARENT"
	cmd.Short = `List memory store permissions.`
	cmd.Long = `List memory store permissions.

  Lists explicit memory store grants. Only the owner or a workspace admin may
  list permissions.

  Arguments:
    PARENT: Resource name of the store, in the form
      memory-stores/{managed_memory_store_id}.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

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

		listMemoryStorePermissionsReq.Parent = args[0]

		response := w.AgentKit.ListMemoryStorePermissions(ctx, listMemoryStorePermissionsReq)
		if listMemoryStorePermissionsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listMemoryStorePermissionsLimit)
		}
		if listMemoryStorePermissionsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listMemoryStorePermissionsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listMemoryStorePermissionsOverrides {
		fn(cmd, &listMemoryStorePermissionsReq)
	}

	return cmd
}

// start list-memory-stores command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listMemoryStoresOverrides []func(
	*cobra.Command,
	*agentkit.ListManagedMemoryStoresRequest,
)

func newListMemoryStores() *cobra.Command {
	cmd := &cobra.Command{}

	var listMemoryStoresReq agentkit.ListManagedMemoryStoresRequest
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
	cmd.Short = `*Beta* List memory stores.`
	cmd.Long = `This command is in Beta and may change without notice.

List memory stores.

  Lists memory stores in the caller's workspace.`

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

		response := w.AgentKit.ListMemoryStores(ctx, listMemoryStoresReq)
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
	*agentkit.ListSessionItemsRequest,
)

func newListSessionItems() *cobra.Command {
	cmd := &cobra.Command{}

	var listSessionItemsReq agentkit.ListSessionItemsRequest
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
	cmd.Short = `*Beta* List session items.`
	cmd.Long = `This command is in Beta and may change without notice.

List session items.

  Lists items in a session.

  Arguments:
    PARENT: Resource name of the containing session, in the form
      session-stores/{session_store_id}/sessions/{session_id}.`

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

		listSessionItemsReq.Parent = args[0]

		response := w.AgentKit.ListSessionItems(ctx, listSessionItemsReq)
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

// start list-session-store-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSessionStorePermissionsOverrides []func(
	*cobra.Command,
	*agentkit.ListSessionStorePermissionsRequest,
)

func newListSessionStorePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var listSessionStorePermissionsReq agentkit.ListSessionStorePermissionsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSessionStorePermissionsLimit int

	cmd.Flags().IntVar(&listSessionStorePermissionsReq.PageSize, "page-size", listSessionStorePermissionsReq.PageSize, `Maximum number of permissions to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSessionStorePermissionsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listSessionStorePermissionsReq.PageToken, "page-token", listSessionStorePermissionsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-session-store-permissions PARENT"
	cmd.Short = `List session store permissions.`
	cmd.Long = `List session store permissions.

  Lists explicit session store grants. Only the owner or a workspace admin may
  list permissions.

  Arguments:
    PARENT: Resource name of the store, in the form
      session-stores/{session_store_id}.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

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

		listSessionStorePermissionsReq.Parent = args[0]

		response := w.AgentKit.ListSessionStorePermissions(ctx, listSessionStorePermissionsReq)
		if listSessionStorePermissionsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSessionStorePermissionsLimit)
		}
		if listSessionStorePermissionsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSessionStorePermissionsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSessionStorePermissionsOverrides {
		fn(cmd, &listSessionStorePermissionsReq)
	}

	return cmd
}

// start list-session-stores command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSessionStoresOverrides []func(
	*cobra.Command,
	*agentkit.ListSessionStoresRequest,
)

func newListSessionStores() *cobra.Command {
	cmd := &cobra.Command{}

	var listSessionStoresReq agentkit.ListSessionStoresRequest
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
	cmd.Short = `*Beta* List session stores.`
	cmd.Long = `This command is in Beta and may change without notice.

List session stores.

  Lists session stores.`

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

		response := w.AgentKit.ListSessionStores(ctx, listSessionStoresReq)
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
	*agentkit.ListSessionsRequest,
)

func newListSessions() *cobra.Command {
	cmd := &cobra.Command{}

	var listSessionsReq agentkit.ListSessionsRequest
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
	cmd.Short = `*Beta* List sessions.`
	cmd.Long = `This command is in Beta and may change without notice.

List sessions.

  Lists sessions within a session store.

  Arguments:
    PARENT: Resource name of the containing session store, in the form
      session-stores/{session_store_id}.`

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

		listSessionsReq.Parent = args[0]

		response := w.AgentKit.ListSessions(ctx, listSessionsReq)
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
	*agentkit.PopSessionItemRequest,
)

func newPopSessionItem() *cobra.Command {
	cmd := &cobra.Command{}

	var popSessionItemReq agentkit.PopSessionItemRequest

	cmd.Use = "pop-session-item PARENT"
	cmd.Short = `*Beta* Pop an item from a session.`
	cmd.Long = `This command is in Beta and may change without notice.

Pop an item from a session.

  Pops the newest item from a session.

  Arguments:
    PARENT: Resource name of the containing session, in the form
      session-stores/{session_store_id}/sessions/{session_id}.`

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

		popSessionItemReq.Parent = args[0]

		response, err := w.AgentKit.PopSessionItem(ctx, popSessionItemReq)
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

// start revoke-memory-store-permission command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var revokeMemoryStorePermissionOverrides []func(
	*cobra.Command,
	*agentkit.RevokeMemoryStorePermissionRequest,
)

func newRevokeMemoryStorePermission() *cobra.Command {
	cmd := &cobra.Command{}

	var revokeMemoryStorePermissionReq agentkit.RevokeMemoryStorePermissionRequest
	var revokeMemoryStorePermissionJson flags.JsonFlag

	cmd.Flags().Var(&revokeMemoryStorePermissionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "revoke-memory-store-permission NAME"
	cmd.Short = `Revoke memory store permission.`
	cmd.Long = `Revoke memory store permission.

  Removes a principal's explicit READ and WRITE grants on a memory store. Only
  the owner or a workspace admin may revoke permissions. Ownership and inherited
  access are unaffected.

  Arguments:
    NAME: Resource name in the form memory-stores/{managed_memory_store_id}.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

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
			diags := revokeMemoryStorePermissionJson.Unmarshal(&revokeMemoryStorePermissionReq)
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
		revokeMemoryStorePermissionReq.Name = args[0]

		response, err := w.AgentKit.RevokeMemoryStorePermission(ctx, revokeMemoryStorePermissionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range revokeMemoryStorePermissionOverrides {
		fn(cmd, &revokeMemoryStorePermissionReq)
	}

	return cmd
}

// start revoke-session-store-permission command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var revokeSessionStorePermissionOverrides []func(
	*cobra.Command,
	*agentkit.RevokeSessionStorePermissionRequest,
)

func newRevokeSessionStorePermission() *cobra.Command {
	cmd := &cobra.Command{}

	var revokeSessionStorePermissionReq agentkit.RevokeSessionStorePermissionRequest
	var revokeSessionStorePermissionJson flags.JsonFlag

	cmd.Flags().Var(&revokeSessionStorePermissionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "revoke-session-store-permission NAME"
	cmd.Short = `Revoke session store permission.`
	cmd.Long = `Revoke session store permission.

  Removes a principal's explicit READ and WRITE grants on a session store. Only
  the owner or a workspace admin may revoke permissions. Ownership and inherited
  access are unaffected.

  Arguments:
    NAME: Resource name in the form session-stores/{session_store_id}.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

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
			diags := revokeSessionStorePermissionJson.Unmarshal(&revokeSessionStorePermissionReq)
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
		revokeSessionStorePermissionReq.Name = args[0]

		response, err := w.AgentKit.RevokeSessionStorePermission(ctx, revokeSessionStorePermissionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range revokeSessionStorePermissionOverrides {
		fn(cmd, &revokeSessionStorePermissionReq)
	}

	return cmd
}

// start search-memories command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var searchMemoriesOverrides []func(
	*cobra.Command,
	*agentkit.SearchManagedMemoryEntriesRequest,
)

func newSearchMemories() *cobra.Command {
	cmd := &cobra.Command{}

	var searchMemoriesReq agentkit.SearchManagedMemoryEntriesRequest
	var searchMemoriesJson flags.JsonFlag
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var searchMemoriesLimit int

	cmd.Flags().Var(&searchMemoriesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&searchMemoriesReq.PageSize, "page-size", searchMemoriesReq.PageSize, `Maximum number of relevance-ranked entries to return.`)
	cmd.Flags().StringVar(&searchMemoriesReq.PathPrefix, "path-prefix", searchMemoriesReq.PathPrefix, `Optional absolute, case-sensitive path prefix used to restrict searched entries within the actor partition.`)
	cmd.Flags().StringVar(&searchMemoriesReq.SessionId, "session-id", searchMemoriesReq.SessionId, `Optional session identifier.`)
	cmd.Flags().Var(&searchMemoriesReq.View, "view", `Controls whether each returned entry includes memory content. Supported values: [MANAGED_MEMORY_ENTRY_VIEW_BASIC, MANAGED_MEMORY_ENTRY_VIEW_FULL]`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&searchMemoriesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&searchMemoriesReq.PageToken, "page-token", searchMemoriesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "search-memories PARENT ACTOR_ID QUERY"
	cmd.Short = `*Beta* Search memory entries.`
	cmd.Long = `This command is in Beta and may change without notice.

Search memory entries.

  Searches memory entries by text query for one actor. Returns matching entries
  and scores ranked by relevance; view controls whether each returned entry
  includes content.

  Arguments:
    PARENT: Memory store whose entries are searched, in the form
      memory-stores/{managed_memory_store_id}.
    ACTOR_ID: Customer-provided identifier for the actor whose entries are searched.
    QUERY: Free-form search query.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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

		response := w.AgentKit.SearchMemories(ctx, searchMemoriesReq)
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
	*agentkit.UpdateManagedMemoryEntryRequest,
)

func newUpdateMemory() *cobra.Command {
	cmd := &cobra.Command{}

	var updateMemoryReq agentkit.UpdateManagedMemoryEntryRequest
	updateMemoryReq.ManagedMemoryEntry = agentkit.ManagedMemoryEntry{}
	var updateMemoryJson flags.JsonFlag

	cmd.Flags().Var(&updateMemoryJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateMemoryReq.ManagedMemoryEntry.Content, "content", updateMemoryReq.ManagedMemoryEntry.Content, `Optional free-form memory content.`)
	cmd.Flags().StringVar(&updateMemoryReq.ManagedMemoryEntry.Description, "description", updateMemoryReq.ManagedMemoryEntry.Description, `Human-readable description of the memory entry.`)
	cmd.Flags().StringVar(&updateMemoryReq.ManagedMemoryEntry.Name, "name", updateMemoryReq.ManagedMemoryEntry.Name, `Resource name in the form memory-stores/{managed_memory_store_id}/entries/{managed_memory_entry_id}.`)
	cmd.Flags().StringVar(&updateMemoryReq.ManagedMemoryEntry.SessionId, "session-id", updateMemoryReq.ManagedMemoryEntry.SessionId, `Optional identifier for the session associated with this memory entry.`)
	cmd.Flags().Var(&updateMemoryReq.ManagedMemoryEntry.SourceType, "source-type", `Which writer created this entry. Supported values: [MANAGED_MEMORY_ENTRY_SOURCE_TYPE_AGENT, MANAGED_MEMORY_ENTRY_SOURCE_TYPE_DREAMER]`)

	cmd.Use = "update-memory NAME UPDATE_MASK ACTOR_ID PATH"
	cmd.Short = `*Beta* Update a memory entry.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a memory entry.

  Updates selected mutable fields on a memory entry. Identity fields are
  immutable.

  Arguments:
    NAME: Resource name in the form
      memory-stores/{managed_memory_store_id}/entries/{managed_memory_entry_id}.
      Required for updates; assigned by the service on creation.
    UPDATE_MASK: Fields to update. Only content and description may be updated.
    ACTOR_ID: Customer-provided identifier for the actor whose memory this entry
      represents.
    PATH: Absolute, case-sensitive path identifying the entry within its actor and
      optional session. Paths must begin with / and must not contain empty,
      . or .. segments.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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

		response, err := w.AgentKit.UpdateMemory(ctx, updateMemoryReq)
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
	*agentkit.UpdateManagedMemoryStoreRequest,
)

func newUpdateMemoryStore() *cobra.Command {
	cmd := &cobra.Command{}

	var updateMemoryStoreReq agentkit.UpdateManagedMemoryStoreRequest
	updateMemoryStoreReq.ManagedMemoryStore = agentkit.ManagedMemoryStore{}
	var updateMemoryStoreJson flags.JsonFlag

	cmd.Flags().Var(&updateMemoryStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateMemoryStoreReq.ManagedMemoryStore.Description, "description", updateMemoryStoreReq.ManagedMemoryStore.Description, `Human-readable description of the memory store.`)
	cmd.Flags().StringVar(&updateMemoryStoreReq.ManagedMemoryStore.DisplayName, "display-name", updateMemoryStoreReq.ManagedMemoryStore.DisplayName, `Deprecated compatibility alias for the caller-provided memory store ID.`)
	// TODO: complex arg: storage_backend

	cmd.Use = "update-memory-store NAME UPDATE_MASK"
	cmd.Short = `*Beta* Update a memory store.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a memory store.

  Updates a memory store's description and metadata.

  Arguments:
    NAME: Resource name in the form memory-stores/{managed_memory_store_id}.
    UPDATE_MASK: Only description and metadata may be updated. Selecting metadata
      replaces the entire map.`

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

		response, err := w.AgentKit.UpdateMemoryStore(ctx, updateMemoryStoreReq)
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
	*agentkit.UpdateSessionRequest,
)

func newUpdateSession() *cobra.Command {
	cmd := &cobra.Command{}

	var updateSessionReq agentkit.UpdateSessionRequest
	updateSessionReq.Session = agentkit.Session{}
	var updateSessionJson flags.JsonFlag

	cmd.Flags().Var(&updateSessionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: map via StringToStringVar: metadata
	cmd.Flags().StringVar(&updateSessionReq.Session.ParentSessionId, "parent-session-id", updateSessionReq.Session.ParentSessionId, `Immediate parent session ID.`)

	cmd.Use = "update-session NAME UPDATE_MASK ACTOR_ID"
	cmd.Short = `*Beta* Update a session.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a session.

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
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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

		response, err := w.AgentKit.UpdateSession(ctx, updateSessionReq)
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
	*agentkit.UpdateSessionStoreRequest,
)

func newUpdateSessionStore() *cobra.Command {
	cmd := &cobra.Command{}

	var updateSessionStoreReq agentkit.UpdateSessionStoreRequest
	updateSessionStoreReq.SessionStore = agentkit.SessionStore{}
	var updateSessionStoreJson flags.JsonFlag

	cmd.Flags().Var(&updateSessionStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateSessionStoreReq.SessionStore.Description, "description", updateSessionStoreReq.SessionStore.Description, `Human-readable description of the session store.`)
	// TODO: map via StringToStringVar: metadata

	cmd.Use = "update-session-store NAME UPDATE_MASK"
	cmd.Short = `*Beta* Update a session store.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a session store.

  Updates a session store's description and metadata.

  Arguments:
    NAME: Resource name in the form session-stores/{session_store_id}.
    UPDATE_MASK: Fields to update. Only description and metadata are mutable; any other
      path returns INVALID_PARAMETER_VALUE.`

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

		response, err := w.AgentKit.UpdateSessionStore(ctx, updateSessionStoreReq)
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

// end service AgentKit
