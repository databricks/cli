// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package agent_kit

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	agentkit "github.com/databricks/cli/libs/sdktypes/agentkit/v1"
	sdkagentkit "github.com/databricks/sdk-go/agentkit/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-kit",
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
	*agentkit.AppendSessionItemsRequest,
)

func newAppendSessionItems() *cobra.Command {
	cmd := &cobra.Command{}
	var appendSessionItemsReq agentkit.AppendSessionItemsRequest

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
		_ = w

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
		appendSessionItemsReq.Parent = new(args[0])

		sdkReq, err := agentkit.AppendSessionItemsRequestToSDK(&appendSessionItemsReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.AppendSessionItems(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.AppendSessionItemsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
		_ = w

		clearSessionItemsReq.Parent = new(args[0])

		sdkReq, err := agentkit.ClearSessionItemsRequestToSDK(&clearSessionItemsReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.ClearSessionItems(ctx, *sdkReq)
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
	createMemoryReq.ManagedMemoryEntry = &agentkit.ManagedMemoryEntry{}

	var createMemoryJson flags.JsonFlag

	cmd.Flags().Var(&createMemoryJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createMemoryReq.ManagedMemoryEntryId, "managed-memory-entry-id", `Optional caller-selected managed memory entry ID.`)
	flags.StringPtrVar(cmd.Flags(), &createMemoryReq.ManagedMemoryEntry.Content, "content", `Optional free-form memory content.`)
	flags.StringPtrVar(cmd.Flags(), &createMemoryReq.ManagedMemoryEntry.Description, "description", `Human-readable description of the memory entry.`)
	flags.StringPtrVar(cmd.Flags(), &createMemoryReq.ManagedMemoryEntry.SessionId, "session-id", `Optional identifier for the session associated with this memory entry.`)
	flags.EnumVar(cmd.Flags(), &createMemoryReq.ManagedMemoryEntry.SourceType, "source-type", `Which writer created this entry. Supported values: [MANAGED_MEMORY_ENTRY_SOURCE_TYPE_AGENT, MANAGED_MEMORY_ENTRY_SOURCE_TYPE_DREAMER]`, "ManagedMemoryEntrySourceType", "MANAGED_MEMORY_ENTRY_SOURCE_TYPE_AGENT", "MANAGED_MEMORY_ENTRY_SOURCE_TYPE_DREAMER")

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
		_ = w

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
		createMemoryReq.Parent = new(args[0])
		if !cmd.Flags().Changed("json") {
			createMemoryReq.ManagedMemoryEntry.ActorId = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createMemoryReq.ManagedMemoryEntry.Path = new(args[2])
		}

		sdkReq, err := agentkit.CreateManagedMemoryEntryRequestToSDK(&createMemoryReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.CreateMemory(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.ManagedMemoryEntryFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	createMemoryStoreReq.ManagedMemoryStore = &agentkit.ManagedMemoryStore{}

	var createMemoryStoreJson flags.JsonFlag

	cmd.Flags().Var(&createMemoryStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createMemoryStoreReq.ManagedMemoryStore.Description, "description", `Human-readable description of the memory store.`)
	flags.StringPtrVar(cmd.Flags(), &createMemoryStoreReq.ManagedMemoryStore.DisplayName, "display-name", `Deprecated compatibility alias for the caller-provided managed memory store ID.`)
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
		_ = w

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
		createMemoryStoreReq.ManagedMemoryStoreId = new(args[0])

		sdkReq, err := agentkit.CreateManagedMemoryStoreRequestToSDK(&createMemoryStoreReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.CreateMemoryStore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.ManagedMemoryStoreFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	createSessionReq.Session = &agentkit.Session{}

	var createSessionJson flags.JsonFlag

	cmd.Flags().Var(&createSessionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createSessionReq.SessionId, "session-id", `Optional caller-selected session ID.`)
	// TODO: map via StringToStringVar: metadata
	flags.StringPtrVar(cmd.Flags(), &createSessionReq.Session.ParentSessionId, "parent-session-id", `Immediate parent session ID.`)

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
		_ = w

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
		createSessionReq.Parent = new(args[0])
		if !cmd.Flags().Changed("json") {
			createSessionReq.Session.ActorId = new(args[1])
		}

		sdkReq, err := agentkit.CreateSessionRequestToSDK(&createSessionReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.CreateSession(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.SessionFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	createSessionStoreReq.SessionStore = &agentkit.SessionStore{}

	var createSessionStoreJson flags.JsonFlag

	cmd.Flags().Var(&createSessionStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createSessionStoreReq.SessionStore.Description, "description", `Human-readable description of the session store.`)
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
		_ = w

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
		createSessionStoreReq.SessionStoreId = new(args[0])

		sdkReq, err := agentkit.CreateSessionStoreRequestToSDK(&createSessionStoreReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.CreateSessionStore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.SessionStoreFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkagentkit.DeleteManagedMemoryEntryRequest,
)

func newDeleteMemory() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteMemoryReq sdkagentkit.DeleteManagedMemoryEntryRequest

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
		_ = w

		deleteMemoryReq.Name = new(args[0])

		sdkReq := &deleteMemoryReq
		err = w.AgentKit.DeleteMemory(ctx, *sdkReq)
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
	*sdkagentkit.DeleteManagedMemoryStoreRequest,
)

func newDeleteMemoryStore() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteMemoryStoreReq sdkagentkit.DeleteManagedMemoryStoreRequest

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
		_ = w

		deleteMemoryStoreReq.Name = new(args[0])

		sdkReq := &deleteMemoryStoreReq
		err = w.AgentKit.DeleteMemoryStore(ctx, *sdkReq)
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
	*sdkagentkit.DeleteSessionRequest,
)

func newDeleteSession() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteSessionReq sdkagentkit.DeleteSessionRequest

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
		_ = w

		deleteSessionReq.Name = new(args[0])

		sdkReq := &deleteSessionReq
		err = w.AgentKit.DeleteSession(ctx, *sdkReq)
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
	*sdkagentkit.DeleteSessionStoreRequest,
)

func newDeleteSessionStore() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteSessionStoreReq sdkagentkit.DeleteSessionStoreRequest

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
		_ = w

		deleteSessionStoreReq.Name = new(args[0])

		sdkReq := &deleteSessionStoreReq
		err = w.AgentKit.DeleteSessionStore(ctx, *sdkReq)
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

	flags.BoolPtrVar(cmd.Flags(), &extractMemoriesReq.DryRun, "dry-run", `When true, extract and return the entries without writing them to the memory store.`)
	flags.StringPtrVar(cmd.Flags(), &extractMemoriesReq.Instructions, "instructions", `Instructions steering what is extracted from the session.`)

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
		_ = w

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
		extractMemoriesReq.SessionStore = new(args[0])
		extractMemoriesReq.SessionId = new(args[1])
		if !cmd.Flags().Changed("json") {
			extractMemoriesReq.MemoryStore = new(args[2])
		}

		sdkReq, err := agentkit.ExtractMemoriesRequestToSDK(&extractMemoriesReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.ExtractMemories(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.ExtractMemoriesResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	flags.StringPtrVar(cmd.Flags(), &forkSessionReq.SessionId, "session-id", `Optional unique ID for the forked session.`)
	flags.StringPtrVar(cmd.Flags(), &forkSessionReq.UpToItemId, "up-to-item-id", `Optional last item ID to copy through, inclusively.`)

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
		_ = w

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
		forkSessionReq.Parent = new(args[0])
		if !cmd.Flags().Changed("json") {
			forkSessionReq.SourceSessionId = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			forkSessionReq.ActorId = new(args[2])
		}

		sdkReq, err := agentkit.ForkSessionRequestToSDK(&forkSessionReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.ForkSession(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.ForkSessionResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

	var readMaskParam string
	cmd.Flags().StringVar(&readMaskParam, "read-mask", readMaskParam, `Deprecated. Wire name: 'read_mask'.`)

	cmd.Use = "get-memory NAME"
	cmd.Short = `Get a managed memory entry by resource name.`
	cmd.Long = `Get a managed memory entry by resource name.

  Retrieves a managed memory entry by resource name. view controls whether the
  response includes memory content. Returns NOT_FOUND when the entry does not
  exist in the caller's workspace.

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
		_ = w

		getMemoryReq.Name = new(args[0])

		if readMaskParam != "" {
			_ = readMaskParam
		}

		sdkReq, err := agentkit.GetManagedMemoryEntryRequestToSDK(&getMemoryReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.GetMemory(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.ManagedMemoryEntryFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkagentkit.GetManagedMemoryStoreRequest,
)

func newGetMemoryStore() *cobra.Command {
	cmd := &cobra.Command{}
	var getMemoryStoreReq sdkagentkit.GetManagedMemoryStoreRequest

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
		_ = w

		getMemoryStoreReq.Name = new(args[0])

		sdkReq := &getMemoryStoreReq
		response, err := w.AgentKit.GetMemoryStore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.ManagedMemoryStoreFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkagentkit.GetSessionRequest,
)

func newGetSession() *cobra.Command {
	cmd := &cobra.Command{}
	var getSessionReq sdkagentkit.GetSessionRequest

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
		_ = w

		getSessionReq.Name = new(args[0])

		sdkReq := &getSessionReq
		response, err := w.AgentKit.GetSession(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.SessionFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkagentkit.GetSessionStoreRequest,
)

func newGetSessionStore() *cobra.Command {
	cmd := &cobra.Command{}
	var getSessionStoreReq sdkagentkit.GetSessionStoreRequest

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
		_ = w

		getSessionStoreReq.Name = new(args[0])

		sdkReq := &getSessionStoreReq
		response, err := w.AgentKit.GetSessionStore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.SessionStoreFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*agentkit.ListManagedMemoryEntriesRequest,
)

func newListMemories() *cobra.Command {
	cmd := &cobra.Command{}
	var listMemoriesReq agentkit.ListManagedMemoryEntriesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listMemoriesLimit int

	flags.IntPtrVar(cmd.Flags(), &listMemoriesReq.PageSize, "page-size", `Maximum number of entries to return. Wire name: 'page_size'.`)
	flags.StringPtrVar(cmd.Flags(), &listMemoriesReq.PathPrefix, "path-prefix", `Optional path prefix used to restrict entries within the actor partition. Wire name: 'path_prefix'.`)
	var readMaskParam string
	cmd.Flags().StringVar(&readMaskParam, "read-mask", readMaskParam, `Deprecated. Wire name: 'read_mask'.`)
	flags.StringPtrVar(cmd.Flags(), &listMemoriesReq.SessionId, "session-id", `Optional session identifier. Wire name: 'session_id'.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listMemoriesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listMemoriesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-memories PARENT ACTOR_ID"
	cmd.Short = `List managed memory entries.`
	cmd.Long = `List managed memory entries.

  Lists managed memory entries for one actor. An exact path filters entries
  across sessions, ignoring session metadata. Otherwise, session_id and
  path_prefix restrict the actor partition. view controls whether each
  returned entry includes memory content.

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
		_ = w

		listMemoriesReq.Parent = new(args[0])
		listMemoriesReq.ActorId = new(args[1])

		if readMaskParam != "" {
			_ = readMaskParam
		}

		sdkReq, err := agentkit.ListManagedMemoryEntriesRequestToSDK(&listMemoriesReq)
		if err != nil {
			return err
		}
		if listMemoriesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listMemoriesLimit)
		}
		if listMemoriesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listMemoriesLimit)
		}
		response := w.AgentKit.ListMemoriesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, agentkit.ManagedMemoryEntryFromSDK))
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
	*agentkit.ListManagedMemoryStoresRequest,
)

func newListMemoryStores() *cobra.Command {
	cmd := &cobra.Command{}
	var listMemoryStoresReq agentkit.ListManagedMemoryStoresRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listMemoryStoresLimit int

	flags.IntPtrVar(cmd.Flags(), &listMemoryStoresReq.PageSize, "page-size", `Maximum number of stores to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listMemoryStoresLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listMemoryStoresReq.PageToken, "page-token", `Pagination token.`)
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
		_ = w

		sdkReq, err := agentkit.ListManagedMemoryStoresRequestToSDK(&listMemoryStoresReq)
		if err != nil {
			return err
		}
		if listMemoryStoresLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listMemoryStoresLimit)
		}
		if listMemoryStoresLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listMemoryStoresLimit)
		}
		response := w.AgentKit.ListMemoryStoresIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, agentkit.ManagedMemoryStoreFromSDK))
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

	flags.StringPtrVar(cmd.Flags(), &listSessionItemsReq.OrderBy, "order-by", `Sort order.`)
	flags.IntPtrVar(cmd.Flags(), &listSessionItemsReq.PageSize, "page-size", `Maximum number of items to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSessionItemsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listSessionItemsReq.PageToken, "page-token", `Pagination token.`)
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
		_ = w

		listSessionItemsReq.Parent = new(args[0])

		sdkReq, err := agentkit.ListSessionItemsRequestToSDK(&listSessionItemsReq)
		if err != nil {
			return err
		}
		if listSessionItemsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSessionItemsLimit)
		}
		if listSessionItemsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSessionItemsLimit)
		}
		response := w.AgentKit.ListSessionItemsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, agentkit.SessionItemFromSDK))
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
	*agentkit.ListSessionStoresRequest,
)

func newListSessionStores() *cobra.Command {
	cmd := &cobra.Command{}
	var listSessionStoresReq agentkit.ListSessionStoresRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSessionStoresLimit int

	flags.IntPtrVar(cmd.Flags(), &listSessionStoresReq.PageSize, "page-size", `Maximum number of session stores to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSessionStoresLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listSessionStoresReq.PageToken, "page-token", `Pagination token.`)
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
		_ = w

		sdkReq, err := agentkit.ListSessionStoresRequestToSDK(&listSessionStoresReq)
		if err != nil {
			return err
		}
		if listSessionStoresLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSessionStoresLimit)
		}
		if listSessionStoresLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSessionStoresLimit)
		}
		response := w.AgentKit.ListSessionStoresIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, agentkit.SessionStoreFromSDK))
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

	flags.StringPtrVar(cmd.Flags(), &listSessionsReq.Filter, "filter", `Filter expression.`)
	flags.StringPtrVar(cmd.Flags(), &listSessionsReq.OrderBy, "order-by", `Sort order.`)
	flags.IntPtrVar(cmd.Flags(), &listSessionsReq.PageSize, "page-size", `Maximum number of sessions to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSessionsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listSessionsReq.PageToken, "page-token", `Pagination token.`)
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
		_ = w

		listSessionsReq.Parent = new(args[0])

		sdkReq, err := agentkit.ListSessionsRequestToSDK(&listSessionsReq)
		if err != nil {
			return err
		}
		if listSessionsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSessionsLimit)
		}
		if listSessionsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSessionsLimit)
		}
		response := w.AgentKit.ListSessionsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, agentkit.SessionFromSDK))
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
		_ = w

		popSessionItemReq.Parent = new(args[0])

		sdkReq, err := agentkit.PopSessionItemRequestToSDK(&popSessionItemReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.PopSessionItem(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.PopSessionItemResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

	flags.IntPtrVar(cmd.Flags(), &searchMemoriesReq.PageSize, "page-size", `Maximum number of relevance-ranked entries to return. Wire name: 'page_size'.`)
	flags.StringPtrVar(cmd.Flags(), &searchMemoriesReq.PathPrefix, "path-prefix", `Optional absolute, case-sensitive path prefix used to restrict searched entries within the actor partition. Wire name: 'path_prefix'.`)
	var readMaskParam string
	cmd.Flags().StringVar(&readMaskParam, "read-mask", readMaskParam, `Deprecated. Wire name: 'read_mask'.`)
	flags.StringPtrVar(cmd.Flags(), &searchMemoriesReq.SessionId, "session-id", `Optional session identifier. Wire name: 'session_id'.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&searchMemoriesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &searchMemoriesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "search-memories PARENT ACTOR_ID QUERY"
	cmd.Short = `Search managed memory entries.`
	cmd.Long = `Search managed memory entries.

  Searches managed memory entries by text query for one actor. Returns matching
  entries and scores ranked by relevance; view controls whether each returned
  entry includes content.

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
		_ = w

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
		searchMemoriesReq.Parent = new(args[0])
		if !cmd.Flags().Changed("json") {
			searchMemoriesReq.ActorId = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			searchMemoriesReq.Query = new(args[2])
		}

		if readMaskParam != "" {
			_ = readMaskParam
		}

		sdkReq, err := agentkit.SearchManagedMemoryEntriesRequestToSDK(&searchMemoriesReq)
		if err != nil {
			return err
		}
		if searchMemoriesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", searchMemoriesLimit)
		}
		if searchMemoriesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, searchMemoriesLimit)
		}
		response := w.AgentKit.SearchMemoriesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, agentkit.ManagedMemoryEntrySearchResultFromSDK))
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
	updateMemoryReq.ManagedMemoryEntry = &agentkit.ManagedMemoryEntry{}

	var updateMemoryJson flags.JsonFlag

	cmd.Flags().Var(&updateMemoryJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateMemoryReq.ManagedMemoryEntry.Content, "content", `Optional free-form memory content.`)
	flags.StringPtrVar(cmd.Flags(), &updateMemoryReq.ManagedMemoryEntry.Description, "description", `Human-readable description of the memory entry.`)
	flags.StringPtrVar(cmd.Flags(), &updateMemoryReq.ManagedMemoryEntry.SessionId, "session-id", `Optional identifier for the session associated with this memory entry.`)
	flags.EnumVar(cmd.Flags(), &updateMemoryReq.ManagedMemoryEntry.SourceType, "source-type", `Which writer created this entry. Supported values: [MANAGED_MEMORY_ENTRY_SOURCE_TYPE_AGENT, MANAGED_MEMORY_ENTRY_SOURCE_TYPE_DREAMER]`, "ManagedMemoryEntrySourceType", "MANAGED_MEMORY_ENTRY_SOURCE_TYPE_AGENT", "MANAGED_MEMORY_ENTRY_SOURCE_TYPE_DREAMER")

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
		_ = w

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
		updateMemoryReq.ManagedMemoryEntry.Name = new(args[0])
		if args[1] != "" {
			updateMemoryReq.UpdateMask = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			updateMemoryReq.ManagedMemoryEntry.ActorId = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			updateMemoryReq.ManagedMemoryEntry.Path = new(args[3])
		}

		sdkReq, err := agentkit.UpdateManagedMemoryEntryRequestToSDK(&updateMemoryReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.UpdateMemory(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.ManagedMemoryEntryFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	updateMemoryStoreReq.ManagedMemoryStore = &agentkit.ManagedMemoryStore{}

	var updateMemoryStoreJson flags.JsonFlag

	cmd.Flags().Var(&updateMemoryStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateMemoryStoreReq.ManagedMemoryStore.Description, "description", `Human-readable description of the memory store.`)
	flags.StringPtrVar(cmd.Flags(), &updateMemoryStoreReq.ManagedMemoryStore.DisplayName, "display-name", `Deprecated compatibility alias for the caller-provided managed memory store ID.`)
	// TODO: complex arg: storage_backend

	cmd.Use = "update-memory-store NAME UPDATE_MASK"
	cmd.Short = `Update a managed memory store.`
	cmd.Long = `Update a managed memory store.

  Updates a managed memory store's description and metadata.

  Arguments:
    NAME: Resource name in the form memory-stores/{managed_memory_store_id}.
    UPDATE_MASK: Only description and metadata may be updated. Selecting metadata
      replaces the entire map.`

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
		_ = w

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
		updateMemoryStoreReq.ManagedMemoryStore.Name = new(args[0])
		if args[1] != "" {
			updateMemoryStoreReq.UpdateMask = new(args[1])
		}

		sdkReq, err := agentkit.UpdateManagedMemoryStoreRequestToSDK(&updateMemoryStoreReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.UpdateMemoryStore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.ManagedMemoryStoreFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	updateSessionReq.Session = &agentkit.Session{}

	var updateSessionJson flags.JsonFlag

	cmd.Flags().Var(&updateSessionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: map via StringToStringVar: metadata
	flags.StringPtrVar(cmd.Flags(), &updateSessionReq.Session.ParentSessionId, "parent-session-id", `Immediate parent session ID.`)

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
		_ = w

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
		updateSessionReq.Session.Name = new(args[0])
		if args[1] != "" {
			updateSessionReq.UpdateMask = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			updateSessionReq.Session.ActorId = new(args[2])
		}

		sdkReq, err := agentkit.UpdateSessionRequestToSDK(&updateSessionReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.UpdateSession(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.SessionFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	updateSessionStoreReq.SessionStore = &agentkit.SessionStore{}

	var updateSessionStoreJson flags.JsonFlag

	cmd.Flags().Var(&updateSessionStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateSessionStoreReq.SessionStore.Description, "description", `Human-readable description of the session store.`)
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
		_ = w

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
		updateSessionStoreReq.SessionStore.Name = new(args[0])
		if args[1] != "" {
			updateSessionStoreReq.UpdateMask = new(args[1])
		}

		sdkReq, err := agentkit.UpdateSessionStoreRequestToSDK(&updateSessionStoreReq)
		if err != nil {
			return err
		}
		response, err := w.AgentKit.UpdateSessionStore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := agentkit.SessionStoreFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
