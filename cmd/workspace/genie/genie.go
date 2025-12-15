// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package genie

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
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
		Use:   "genie",
		Short: `Genie provides a no-code experience for business users, powered by AI/BI.`,
		Long: `Genie provides a no-code experience for business users, powered by AI/BI.
  Analysts set up spaces that business users can use to ask questions using
  natural language. Genie uses data registered to Unity Catalog and requires at
  least CAN USE permission on a Pro or Serverless SQL warehouse. Also,
  Databricks Assistant must be enabled.`,
		GroupID: "dashboards",
		RunE:    root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateMessage())
	cmd.AddCommand(newCreateSpace())
	cmd.AddCommand(newDeleteConversation())
	cmd.AddCommand(newDeleteConversationMessage())
	cmd.AddCommand(newExecuteMessageAttachmentQuery())
	cmd.AddCommand(newExecuteMessageQuery())
	cmd.AddCommand(newGetMessage())
	cmd.AddCommand(newGetMessageAttachmentQueryResult())
	cmd.AddCommand(newGetMessageQueryResult())
	cmd.AddCommand(newGetMessageQueryResultByAttachment())
	cmd.AddCommand(newGetSpace())
	cmd.AddCommand(newListConversationMessages())
	cmd.AddCommand(newListConversations())
	cmd.AddCommand(newListSpaces())
	cmd.AddCommand(newSendMessageFeedback())
	cmd.AddCommand(newStartConversation())
	cmd.AddCommand(newTrashSpace())
	cmd.AddCommand(newUpdateSpace())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-message command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createMessageOverrides []func(
	*cobra.Command,
	*dashboards.GenieCreateConversationMessageRequest,
)

func newCreateMessage() *cobra.Command {
	cmd := &cobra.Command{}

	var createMessageReq dashboards.GenieCreateConversationMessageRequest
	var createMessageJson flags.JsonFlag

	var createMessageSkipWait bool
	var createMessageTimeout time.Duration

	cmd.Flags().BoolVar(&createMessageSkipWait, "no-wait", createMessageSkipWait, `do not wait to reach COMPLETED state`)
	cmd.Flags().DurationVar(&createMessageTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach COMPLETED state`)

	cmd.Flags().Var(&createMessageJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create-message SPACE_ID CONVERSATION_ID CONTENT"
	cmd.Short = `Create conversation message.`
	cmd.Long = `Create conversation message.

  Create new message in a [conversation](:method:genie/startconversation). The
  AI response uses all previously created messages in the conversation to
  respond.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the conversation is started.
    CONVERSATION_ID: The ID associated with the conversation.
    CONTENT: User message content.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only SPACE_ID, CONVERSATION_ID as positional arguments. Provide 'content' in your JSON input")
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
			diags := createMessageJson.Unmarshal(&createMessageReq)
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
		createMessageReq.SpaceId = args[0]
		createMessageReq.ConversationId = args[1]
		if !cmd.Flags().Changed("json") {
			createMessageReq.Content = args[2]
		}

		wait, err := w.Genie.CreateMessage(ctx, createMessageReq)
		if err != nil {
			return err
		}
		if createMessageSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *dashboards.GenieMessage) {
			status := i.Status
			statusMessage := fmt.Sprintf("current status: %s", status)
			spinner <- statusMessage
		}).GetWithTimeout(createMessageTimeout)
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
	for _, fn := range createMessageOverrides {
		fn(cmd, &createMessageReq)
	}

	return cmd
}

// start create-space command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSpaceOverrides []func(
	*cobra.Command,
	*dashboards.GenieCreateSpaceRequest,
)

func newCreateSpace() *cobra.Command {
	cmd := &cobra.Command{}

	var createSpaceReq dashboards.GenieCreateSpaceRequest
	var createSpaceJson flags.JsonFlag

	cmd.Flags().Var(&createSpaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createSpaceReq.Description, "description", createSpaceReq.Description, `Optional description.`)
	cmd.Flags().StringVar(&createSpaceReq.ParentPath, "parent-path", createSpaceReq.ParentPath, `Parent folder path where the space will be registered.`)
	cmd.Flags().StringVar(&createSpaceReq.Title, "title", createSpaceReq.Title, `Optional title override.`)

	cmd.Use = "create-space WAREHOUSE_ID SERIALIZED_SPACE"
	cmd.Short = `Create Genie Space.`
	cmd.Long = `Create Genie Space.

  Creates a Genie space from a serialized payload.

  Arguments:
    WAREHOUSE_ID: Warehouse to associate with the new space
    SERIALIZED_SPACE: The contents of the Genie Space in serialized string form. Use the [Get
      Genie Space](:method:genie/getspace) API to retrieve an example response,
      which includes the serialized_space field. This field provides the
      structure of the JSON string that represents the space's layout and
      components.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'warehouse_id', 'serialized_space' in your JSON input")
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
			diags := createSpaceJson.Unmarshal(&createSpaceReq)
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
		if !cmd.Flags().Changed("json") {
			createSpaceReq.WarehouseId = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createSpaceReq.SerializedSpace = args[1]
		}

		response, err := w.Genie.CreateSpace(ctx, createSpaceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createSpaceOverrides {
		fn(cmd, &createSpaceReq)
	}

	return cmd
}

// start delete-conversation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteConversationOverrides []func(
	*cobra.Command,
	*dashboards.GenieDeleteConversationRequest,
)

func newDeleteConversation() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteConversationReq dashboards.GenieDeleteConversationRequest

	cmd.Use = "delete-conversation SPACE_ID CONVERSATION_ID"
	cmd.Short = `Delete conversation.`
	cmd.Long = `Delete conversation.

  Delete a conversation.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the conversation is located.
    CONVERSATION_ID: The ID of the conversation to delete.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteConversationReq.SpaceId = args[0]
		deleteConversationReq.ConversationId = args[1]

		err = w.Genie.DeleteConversation(ctx, deleteConversationReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteConversationOverrides {
		fn(cmd, &deleteConversationReq)
	}

	return cmd
}

// start delete-conversation-message command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteConversationMessageOverrides []func(
	*cobra.Command,
	*dashboards.GenieDeleteConversationMessageRequest,
)

func newDeleteConversationMessage() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteConversationMessageReq dashboards.GenieDeleteConversationMessageRequest

	cmd.Use = "delete-conversation-message SPACE_ID CONVERSATION_ID MESSAGE_ID"
	cmd.Short = `Delete conversation message.`
	cmd.Long = `Delete conversation message.

  Delete a conversation message.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the message is located.
    CONVERSATION_ID: The ID associated with the conversation.
    MESSAGE_ID: The ID associated with the message to delete.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteConversationMessageReq.SpaceId = args[0]
		deleteConversationMessageReq.ConversationId = args[1]
		deleteConversationMessageReq.MessageId = args[2]

		err = w.Genie.DeleteConversationMessage(ctx, deleteConversationMessageReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteConversationMessageOverrides {
		fn(cmd, &deleteConversationMessageReq)
	}

	return cmd
}

// start execute-message-attachment-query command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var executeMessageAttachmentQueryOverrides []func(
	*cobra.Command,
	*dashboards.GenieExecuteMessageAttachmentQueryRequest,
)

func newExecuteMessageAttachmentQuery() *cobra.Command {
	cmd := &cobra.Command{}

	var executeMessageAttachmentQueryReq dashboards.GenieExecuteMessageAttachmentQueryRequest

	cmd.Use = "execute-message-attachment-query SPACE_ID CONVERSATION_ID MESSAGE_ID ATTACHMENT_ID"
	cmd.Short = `Execute message attachment SQL query.`
	cmd.Long = `Execute message attachment SQL query.

  Execute the SQL for a message query attachment. Use this API when the query
  attachment has expired and needs to be re-executed.

  Arguments:
    SPACE_ID: Genie space ID
    CONVERSATION_ID: Conversation ID
    MESSAGE_ID: Message ID
    ATTACHMENT_ID: Attachment ID`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		executeMessageAttachmentQueryReq.SpaceId = args[0]
		executeMessageAttachmentQueryReq.ConversationId = args[1]
		executeMessageAttachmentQueryReq.MessageId = args[2]
		executeMessageAttachmentQueryReq.AttachmentId = args[3]

		response, err := w.Genie.ExecuteMessageAttachmentQuery(ctx, executeMessageAttachmentQueryReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range executeMessageAttachmentQueryOverrides {
		fn(cmd, &executeMessageAttachmentQueryReq)
	}

	return cmd
}

// start execute-message-query command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var executeMessageQueryOverrides []func(
	*cobra.Command,
	*dashboards.GenieExecuteMessageQueryRequest,
)

func newExecuteMessageQuery() *cobra.Command {
	cmd := &cobra.Command{}

	var executeMessageQueryReq dashboards.GenieExecuteMessageQueryRequest

	cmd.Use = "execute-message-query SPACE_ID CONVERSATION_ID MESSAGE_ID"
	cmd.Short = `[Deprecated] Execute SQL query in a conversation message.`
	cmd.Long = `[Deprecated] Execute SQL query in a conversation message.

  DEPRECATED: Use [Execute Message Attachment
  Query](:method:genie/executemessageattachmentquery) instead.

  Arguments:
    SPACE_ID: Genie space ID
    CONVERSATION_ID: Conversation ID
    MESSAGE_ID: Message ID`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		executeMessageQueryReq.SpaceId = args[0]
		executeMessageQueryReq.ConversationId = args[1]
		executeMessageQueryReq.MessageId = args[2]

		response, err := w.Genie.ExecuteMessageQuery(ctx, executeMessageQueryReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range executeMessageQueryOverrides {
		fn(cmd, &executeMessageQueryReq)
	}

	return cmd
}

// start get-message command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMessageOverrides []func(
	*cobra.Command,
	*dashboards.GenieGetConversationMessageRequest,
)

func newGetMessage() *cobra.Command {
	cmd := &cobra.Command{}

	var getMessageReq dashboards.GenieGetConversationMessageRequest

	cmd.Use = "get-message SPACE_ID CONVERSATION_ID MESSAGE_ID"
	cmd.Short = `Get conversation message.`
	cmd.Long = `Get conversation message.

  Get message from conversation.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the target conversation is
      located.
    CONVERSATION_ID: The ID associated with the target conversation.
    MESSAGE_ID: The ID associated with the target message from the identified
      conversation.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getMessageReq.SpaceId = args[0]
		getMessageReq.ConversationId = args[1]
		getMessageReq.MessageId = args[2]

		response, err := w.Genie.GetMessage(ctx, getMessageReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getMessageOverrides {
		fn(cmd, &getMessageReq)
	}

	return cmd
}

// start get-message-attachment-query-result command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMessageAttachmentQueryResultOverrides []func(
	*cobra.Command,
	*dashboards.GenieGetMessageAttachmentQueryResultRequest,
)

func newGetMessageAttachmentQueryResult() *cobra.Command {
	cmd := &cobra.Command{}

	var getMessageAttachmentQueryResultReq dashboards.GenieGetMessageAttachmentQueryResultRequest

	cmd.Use = "get-message-attachment-query-result SPACE_ID CONVERSATION_ID MESSAGE_ID ATTACHMENT_ID"
	cmd.Short = `Get message attachment SQL query result.`
	cmd.Long = `Get message attachment SQL query result.

  Get the result of SQL query if the message has a query attachment. This is
  only available if a message has a query attachment and the message status is
  EXECUTING_QUERY OR COMPLETED.

  Arguments:
    SPACE_ID: Genie space ID
    CONVERSATION_ID: Conversation ID
    MESSAGE_ID: Message ID
    ATTACHMENT_ID: Attachment ID`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getMessageAttachmentQueryResultReq.SpaceId = args[0]
		getMessageAttachmentQueryResultReq.ConversationId = args[1]
		getMessageAttachmentQueryResultReq.MessageId = args[2]
		getMessageAttachmentQueryResultReq.AttachmentId = args[3]

		response, err := w.Genie.GetMessageAttachmentQueryResult(ctx, getMessageAttachmentQueryResultReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getMessageAttachmentQueryResultOverrides {
		fn(cmd, &getMessageAttachmentQueryResultReq)
	}

	return cmd
}

// start get-message-query-result command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMessageQueryResultOverrides []func(
	*cobra.Command,
	*dashboards.GenieGetMessageQueryResultRequest,
)

func newGetMessageQueryResult() *cobra.Command {
	cmd := &cobra.Command{}

	var getMessageQueryResultReq dashboards.GenieGetMessageQueryResultRequest

	cmd.Use = "get-message-query-result SPACE_ID CONVERSATION_ID MESSAGE_ID"
	cmd.Short = `[Deprecated] Get conversation message SQL query result.`
	cmd.Long = `[Deprecated] Get conversation message SQL query result.

  DEPRECATED: Use [Get Message Attachment Query
  Result](:method:genie/getmessageattachmentqueryresult) instead.

  Arguments:
    SPACE_ID: Genie space ID
    CONVERSATION_ID: Conversation ID
    MESSAGE_ID: Message ID`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getMessageQueryResultReq.SpaceId = args[0]
		getMessageQueryResultReq.ConversationId = args[1]
		getMessageQueryResultReq.MessageId = args[2]

		response, err := w.Genie.GetMessageQueryResult(ctx, getMessageQueryResultReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getMessageQueryResultOverrides {
		fn(cmd, &getMessageQueryResultReq)
	}

	return cmd
}

// start get-message-query-result-by-attachment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMessageQueryResultByAttachmentOverrides []func(
	*cobra.Command,
	*dashboards.GenieGetQueryResultByAttachmentRequest,
)

func newGetMessageQueryResultByAttachment() *cobra.Command {
	cmd := &cobra.Command{}

	var getMessageQueryResultByAttachmentReq dashboards.GenieGetQueryResultByAttachmentRequest

	cmd.Use = "get-message-query-result-by-attachment SPACE_ID CONVERSATION_ID MESSAGE_ID ATTACHMENT_ID"
	cmd.Short = `[Deprecated] Get conversation message SQL query result.`
	cmd.Long = `[Deprecated] Get conversation message SQL query result.

  DEPRECATED: Use [Get Message Attachment Query
  Result](:method:genie/getmessageattachmentqueryresult) instead.

  Arguments:
    SPACE_ID: Genie space ID
    CONVERSATION_ID: Conversation ID
    MESSAGE_ID: Message ID
    ATTACHMENT_ID: Attachment ID`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getMessageQueryResultByAttachmentReq.SpaceId = args[0]
		getMessageQueryResultByAttachmentReq.ConversationId = args[1]
		getMessageQueryResultByAttachmentReq.MessageId = args[2]
		getMessageQueryResultByAttachmentReq.AttachmentId = args[3]

		response, err := w.Genie.GetMessageQueryResultByAttachment(ctx, getMessageQueryResultByAttachmentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getMessageQueryResultByAttachmentOverrides {
		fn(cmd, &getMessageQueryResultByAttachmentReq)
	}

	return cmd
}

// start get-space command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSpaceOverrides []func(
	*cobra.Command,
	*dashboards.GenieGetSpaceRequest,
)

func newGetSpace() *cobra.Command {
	cmd := &cobra.Command{}

	var getSpaceReq dashboards.GenieGetSpaceRequest

	cmd.Flags().BoolVar(&getSpaceReq.IncludeSerializedSpace, "include-serialized-space", getSpaceReq.IncludeSerializedSpace, `Whether to include the serialized space export in the response.`)

	cmd.Use = "get-space SPACE_ID"
	cmd.Short = `Get Genie Space.`
	cmd.Long = `Get Genie Space.

  Get details of a Genie Space.

  Arguments:
    SPACE_ID: The ID associated with the Genie space`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getSpaceReq.SpaceId = args[0]

		response, err := w.Genie.GetSpace(ctx, getSpaceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSpaceOverrides {
		fn(cmd, &getSpaceReq)
	}

	return cmd
}

// start list-conversation-messages command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listConversationMessagesOverrides []func(
	*cobra.Command,
	*dashboards.GenieListConversationMessagesRequest,
)

func newListConversationMessages() *cobra.Command {
	cmd := &cobra.Command{}

	var listConversationMessagesReq dashboards.GenieListConversationMessagesRequest

	cmd.Flags().IntVar(&listConversationMessagesReq.PageSize, "page-size", listConversationMessagesReq.PageSize, `Maximum number of messages to return per page.`)
	cmd.Flags().StringVar(&listConversationMessagesReq.PageToken, "page-token", listConversationMessagesReq.PageToken, `Token to get the next page of results.`)

	cmd.Use = "list-conversation-messages SPACE_ID CONVERSATION_ID"
	cmd.Short = `List conversation messages.`
	cmd.Long = `List conversation messages.

  List messages in a conversation

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the conversation is located
    CONVERSATION_ID: The ID of the conversation to list messages from`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listConversationMessagesReq.SpaceId = args[0]
		listConversationMessagesReq.ConversationId = args[1]

		response, err := w.Genie.ListConversationMessages(ctx, listConversationMessagesReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listConversationMessagesOverrides {
		fn(cmd, &listConversationMessagesReq)
	}

	return cmd
}

// start list-conversations command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listConversationsOverrides []func(
	*cobra.Command,
	*dashboards.GenieListConversationsRequest,
)

func newListConversations() *cobra.Command {
	cmd := &cobra.Command{}

	var listConversationsReq dashboards.GenieListConversationsRequest

	cmd.Flags().BoolVar(&listConversationsReq.IncludeAll, "include-all", listConversationsReq.IncludeAll, `Include all conversations in the space across all users.`)
	cmd.Flags().IntVar(&listConversationsReq.PageSize, "page-size", listConversationsReq.PageSize, `Maximum number of conversations to return per page.`)
	cmd.Flags().StringVar(&listConversationsReq.PageToken, "page-token", listConversationsReq.PageToken, `Token to get the next page of results.`)

	cmd.Use = "list-conversations SPACE_ID"
	cmd.Short = `List conversations in a Genie Space.`
	cmd.Long = `List conversations in a Genie Space.

  Get a list of conversations in a Genie Space.

  Arguments:
    SPACE_ID: The ID of the Genie space to retrieve conversations from.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listConversationsReq.SpaceId = args[0]

		response, err := w.Genie.ListConversations(ctx, listConversationsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listConversationsOverrides {
		fn(cmd, &listConversationsReq)
	}

	return cmd
}

// start list-spaces command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSpacesOverrides []func(
	*cobra.Command,
	*dashboards.GenieListSpacesRequest,
)

func newListSpaces() *cobra.Command {
	cmd := &cobra.Command{}

	var listSpacesReq dashboards.GenieListSpacesRequest

	cmd.Flags().IntVar(&listSpacesReq.PageSize, "page-size", listSpacesReq.PageSize, `Maximum number of spaces to return per page.`)
	cmd.Flags().StringVar(&listSpacesReq.PageToken, "page-token", listSpacesReq.PageToken, `Pagination token for getting the next page of results.`)

	cmd.Use = "list-spaces"
	cmd.Short = `List Genie spaces.`
	cmd.Long = `List Genie spaces.

  Get list of Genie Spaces.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response, err := w.Genie.ListSpaces(ctx, listSpacesReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSpacesOverrides {
		fn(cmd, &listSpacesReq)
	}

	return cmd
}

// start send-message-feedback command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var sendMessageFeedbackOverrides []func(
	*cobra.Command,
	*dashboards.GenieSendMessageFeedbackRequest,
)

func newSendMessageFeedback() *cobra.Command {
	cmd := &cobra.Command{}

	var sendMessageFeedbackReq dashboards.GenieSendMessageFeedbackRequest
	var sendMessageFeedbackJson flags.JsonFlag

	cmd.Flags().Var(&sendMessageFeedbackJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "send-message-feedback SPACE_ID CONVERSATION_ID MESSAGE_ID RATING"
	cmd.Short = `Send message feedback.`
	cmd.Long = `Send message feedback.

  Send feedback for a message.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the message is located.
    CONVERSATION_ID: The ID associated with the conversation.
    MESSAGE_ID: The ID associated with the message to provide feedback for.
    RATING: The rating (POSITIVE, NEGATIVE, or NONE).
      Supported values: [NEGATIVE, NONE, POSITIVE]`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(3)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only SPACE_ID, CONVERSATION_ID, MESSAGE_ID as positional arguments. Provide 'rating' in your JSON input")
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
			diags := sendMessageFeedbackJson.Unmarshal(&sendMessageFeedbackReq)
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
		sendMessageFeedbackReq.SpaceId = args[0]
		sendMessageFeedbackReq.ConversationId = args[1]
		sendMessageFeedbackReq.MessageId = args[2]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &sendMessageFeedbackReq.Rating)
			if err != nil {
				return fmt.Errorf("invalid RATING: %s", args[3])
			}

		}

		err = w.Genie.SendMessageFeedback(ctx, sendMessageFeedbackReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range sendMessageFeedbackOverrides {
		fn(cmd, &sendMessageFeedbackReq)
	}

	return cmd
}

// start start-conversation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var startConversationOverrides []func(
	*cobra.Command,
	*dashboards.GenieStartConversationMessageRequest,
)

func newStartConversation() *cobra.Command {
	cmd := &cobra.Command{}

	var startConversationReq dashboards.GenieStartConversationMessageRequest
	var startConversationJson flags.JsonFlag

	var startConversationSkipWait bool
	var startConversationTimeout time.Duration

	cmd.Flags().BoolVar(&startConversationSkipWait, "no-wait", startConversationSkipWait, `do not wait to reach COMPLETED state`)
	cmd.Flags().DurationVar(&startConversationTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach COMPLETED state`)

	cmd.Flags().Var(&startConversationJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "start-conversation SPACE_ID CONTENT"
	cmd.Short = `Start conversation.`
	cmd.Long = `Start conversation.

  Start a new conversation.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where you want to start a
      conversation.
    CONTENT: The text of the message that starts the conversation.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only SPACE_ID as positional arguments. Provide 'content' in your JSON input")
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
			diags := startConversationJson.Unmarshal(&startConversationReq)
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
		startConversationReq.SpaceId = args[0]
		if !cmd.Flags().Changed("json") {
			startConversationReq.Content = args[1]
		}

		wait, err := w.Genie.StartConversation(ctx, startConversationReq)
		if err != nil {
			return err
		}
		if startConversationSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *dashboards.GenieMessage) {
			status := i.Status
			statusMessage := fmt.Sprintf("current status: %s", status)
			spinner <- statusMessage
		}).GetWithTimeout(startConversationTimeout)
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
	for _, fn := range startConversationOverrides {
		fn(cmd, &startConversationReq)
	}

	return cmd
}

// start trash-space command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var trashSpaceOverrides []func(
	*cobra.Command,
	*dashboards.GenieTrashSpaceRequest,
)

func newTrashSpace() *cobra.Command {
	cmd := &cobra.Command{}

	var trashSpaceReq dashboards.GenieTrashSpaceRequest

	cmd.Use = "trash-space SPACE_ID"
	cmd.Short = `Trash Genie Space.`
	cmd.Long = `Trash Genie Space.

  Move a Genie Space to the trash.

  Arguments:
    SPACE_ID: The ID associated with the Genie space to be sent to the trash.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		trashSpaceReq.SpaceId = args[0]

		err = w.Genie.TrashSpace(ctx, trashSpaceReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range trashSpaceOverrides {
		fn(cmd, &trashSpaceReq)
	}

	return cmd
}

// start update-space command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateSpaceOverrides []func(
	*cobra.Command,
	*dashboards.GenieUpdateSpaceRequest,
)

func newUpdateSpace() *cobra.Command {
	cmd := &cobra.Command{}

	var updateSpaceReq dashboards.GenieUpdateSpaceRequest
	var updateSpaceJson flags.JsonFlag

	cmd.Flags().Var(&updateSpaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateSpaceReq.Description, "description", updateSpaceReq.Description, `Optional description.`)
	cmd.Flags().StringVar(&updateSpaceReq.SerializedSpace, "serialized-space", updateSpaceReq.SerializedSpace, `The contents of the Genie Space in serialized string form (full replacement).`)
	cmd.Flags().StringVar(&updateSpaceReq.Title, "title", updateSpaceReq.Title, `Optional title override.`)
	cmd.Flags().StringVar(&updateSpaceReq.WarehouseId, "warehouse-id", updateSpaceReq.WarehouseId, `Optional warehouse override.`)

	cmd.Use = "update-space SPACE_ID"
	cmd.Short = `Update Genie Space.`
	cmd.Long = `Update Genie Space.

  Updates a Genie space with a serialized payload.

  Arguments:
    SPACE_ID: Genie space ID`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateSpaceJson.Unmarshal(&updateSpaceReq)
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
		updateSpaceReq.SpaceId = args[0]

		response, err := w.Genie.UpdateSpace(ctx, updateSpaceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateSpaceOverrides {
		fn(cmd, &updateSpaceReq)
	}

	return cmd
}

// end service Genie
