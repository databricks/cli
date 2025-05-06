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
		Annotations: map[string]string{
			"package": "dashboards",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateMessage())
	cmd.AddCommand(newExecuteMessageAttachmentQuery())
	cmd.AddCommand(newExecuteMessageQuery())
	cmd.AddCommand(newGenerateDownloadFullQueryResult())
	cmd.AddCommand(newGetDownloadFullQueryResult())
	cmd.AddCommand(newGetMessage())
	cmd.AddCommand(newGetMessageAttachmentQueryResult())
	cmd.AddCommand(newGetMessageQueryResult())
	cmd.AddCommand(newGetMessageQueryResultByAttachment())
	cmd.AddCommand(newGetSpace())
	cmd.AddCommand(newStartConversation())

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
	// TODO: short flags
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

	// TODO: short flags

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

	// TODO: short flags

	cmd.Use = "execute-message-query SPACE_ID CONVERSATION_ID MESSAGE_ID"
	cmd.Short = `[Deprecated] Execute SQL query in a conversation message.`
	cmd.Long = `[Deprecated] Execute SQL query in a conversation message.
  
  Execute the SQL query in the message.

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

// start generate-download-full-query-result command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var generateDownloadFullQueryResultOverrides []func(
	*cobra.Command,
	*dashboards.GenieGenerateDownloadFullQueryResultRequest,
)

func newGenerateDownloadFullQueryResult() *cobra.Command {
	cmd := &cobra.Command{}

	var generateDownloadFullQueryResultReq dashboards.GenieGenerateDownloadFullQueryResultRequest

	// TODO: short flags

	cmd.Use = "generate-download-full-query-result SPACE_ID CONVERSATION_ID MESSAGE_ID ATTACHMENT_ID"
	cmd.Short = `Generate full query result download.`
	cmd.Long = `Generate full query result download.
  
  Initiates a new SQL execution and returns a download_id that you can use to
  track the progress of the download. The query result is stored in an external
  link and can be retrieved using the [Get Download Full Query
  Result](:method:genie/getdownloadfullqueryresult) API. Warning: Databricks
  strongly recommends that you protect the URLs that are returned by the
  EXTERNAL_LINKS disposition. See [Execute
  Statement](:method:statementexecution/executestatement) for more details.

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

		generateDownloadFullQueryResultReq.SpaceId = args[0]
		generateDownloadFullQueryResultReq.ConversationId = args[1]
		generateDownloadFullQueryResultReq.MessageId = args[2]
		generateDownloadFullQueryResultReq.AttachmentId = args[3]

		response, err := w.Genie.GenerateDownloadFullQueryResult(ctx, generateDownloadFullQueryResultReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range generateDownloadFullQueryResultOverrides {
		fn(cmd, &generateDownloadFullQueryResultReq)
	}

	return cmd
}

// start get-download-full-query-result command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDownloadFullQueryResultOverrides []func(
	*cobra.Command,
	*dashboards.GenieGetDownloadFullQueryResultRequest,
)

func newGetDownloadFullQueryResult() *cobra.Command {
	cmd := &cobra.Command{}

	var getDownloadFullQueryResultReq dashboards.GenieGetDownloadFullQueryResultRequest

	// TODO: short flags

	cmd.Use = "get-download-full-query-result SPACE_ID CONVERSATION_ID MESSAGE_ID ATTACHMENT_ID DOWNLOAD_ID"
	cmd.Short = `Get download full query result.`
	cmd.Long = `Get download full query result.
  
  After [Generating a Full Query Result
  Download](:method:genie/getdownloadfullqueryresult) and successfully receiving
  a download_id, use this API to poll the download progress. When the download
  is complete, the API returns one or more external links to the query result
  files. Warning: Databricks strongly recommends that you protect the URLs that
  are returned by the EXTERNAL_LINKS disposition. You must not set an
  Authorization header in download requests. When using the EXTERNAL_LINKS
  disposition, Databricks returns presigned URLs that grant temporary access to
  data. See [Execute Statement](:method:statementexecution/executestatement) for
  more details.

  Arguments:
    SPACE_ID: Genie space ID
    CONVERSATION_ID: Conversation ID
    MESSAGE_ID: Message ID
    ATTACHMENT_ID: Attachment ID
    DOWNLOAD_ID: Download ID. This ID is provided by the [Generate Download
      endpoint](:method:genie/generateDownloadFullQueryResult)`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(5)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDownloadFullQueryResultReq.SpaceId = args[0]
		getDownloadFullQueryResultReq.ConversationId = args[1]
		getDownloadFullQueryResultReq.MessageId = args[2]
		getDownloadFullQueryResultReq.AttachmentId = args[3]
		getDownloadFullQueryResultReq.DownloadId = args[4]

		response, err := w.Genie.GetDownloadFullQueryResult(ctx, getDownloadFullQueryResultReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDownloadFullQueryResultOverrides {
		fn(cmd, &getDownloadFullQueryResultReq)
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

	// TODO: short flags

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

	// TODO: short flags

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

	// TODO: short flags

	cmd.Use = "get-message-query-result SPACE_ID CONVERSATION_ID MESSAGE_ID"
	cmd.Short = `[Deprecated] Get conversation message SQL query result.`
	cmd.Long = `[Deprecated] Get conversation message SQL query result.
  
  Get the result of SQL query if the message has a query attachment. This is
  only available if a message has a query attachment and the message status is
  EXECUTING_QUERY.

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

	// TODO: short flags

	cmd.Use = "get-message-query-result-by-attachment SPACE_ID CONVERSATION_ID MESSAGE_ID ATTACHMENT_ID"
	cmd.Short = `[Deprecated] Get conversation message SQL query result.`
	cmd.Long = `[Deprecated] Get conversation message SQL query result.
  
  Get the result of SQL query if the message has a query attachment. This is
  only available if a message has a query attachment and the message status is
  EXECUTING_QUERY OR COMPLETED.

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

	// TODO: short flags

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
	// TODO: short flags
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

// end service Genie
