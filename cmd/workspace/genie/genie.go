// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package genie

import (
	"errors"
	"fmt"
	"time"

	"net/url"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/options/lro"
	"github.com/spf13/cobra"

	genie "github.com/databricks/cli/libs/sdktypes/genie/v1"
	sdkgenie "github.com/databricks/sdk-go/genie/v1"
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

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreateMessage())
	cmd.AddCommand(newCreateMessageComment())
	cmd.AddCommand(newCreateSpace())
	cmd.AddCommand(newDeleteConversation())
	cmd.AddCommand(newDeleteConversationMessage())
	cmd.AddCommand(newDownloadMessageAttachmentVisualization())
	cmd.AddCommand(newExecuteMessageAttachmentQuery())
	cmd.AddCommand(newExecuteMessageQuery())
	cmd.AddCommand(newGenerateDownloadFullQueryResult())
	cmd.AddCommand(newGenieCancelResponse())
	cmd.AddCommand(newGenieCreateEvalRun())
	cmd.AddCommand(newGenieGetEvalResultDetails())
	cmd.AddCommand(newGenieGetEvalRun())
	cmd.AddCommand(newGenieListEvalResults())
	cmd.AddCommand(newGenieListEvalRuns())
	cmd.AddCommand(newGetDownloadFullQueryResult())
	cmd.AddCommand(newGetMessage())
	cmd.AddCommand(newGetMessageAttachmentQueryResult())
	cmd.AddCommand(newGetMessageQueryResult())
	cmd.AddCommand(newGetMessageQueryResultByAttachment())
	cmd.AddCommand(newGetSpace())
	cmd.AddCommand(newListConversationComments())
	cmd.AddCommand(newListConversationMessages())
	cmd.AddCommand(newListConversations())
	cmd.AddCommand(newListMessageComments())
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
	*genie.GenieCreateConversationMessageRequest,
)

func newCreateMessage() *cobra.Command {
	cmd := &cobra.Command{}
	var createMessageReq genie.GenieCreateConversationMessageRequest

	var createMessageJson flags.JsonFlag

	var createMessageSkipWait bool
	var createMessageTimeout time.Duration

	cmd.Flags().BoolVar(&createMessageSkipWait, "no-wait", createMessageSkipWait, `do not wait to reach COMPLETED state`)
	cmd.Flags().DurationVar(&createMessageTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach COMPLETED state`)

	cmd.Flags().Var(&createMessageJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &createMessageReq.EnableVisualization, "enable-visualization", `Enable visualization generation.`)

	cmd.Use = "create-message SPACE_ID CONVERSATION_ID CONTENT"
	cmd.Short = `Chat mode: Send a message.`
	cmd.Long = `Chat mode: Send a message.

  Sends a new message in a chat-mode
  [conversation](:method:genie/startconversation). The AI response uses all
  previously created messages in the conversation to respond.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the conversation is started.
    CONVERSATION_ID: The ID associated with the conversation.
    CONTENT: User message content.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only SPACE_ID, CONVERSATION_ID as positional arguments. Provide 'content' in your JSON input")
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
			diags := createMessageJson.Unmarshal(&createMessageReq)
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
		createMessageReq.SpaceId = new(args[0])
		createMessageReq.ConversationId = new(args[1])
		if !cmd.Flags().Changed("json") {
			createMessageReq.Content = new(args[2])
		}

		sdkReq, err := genie.GenieCreateConversationMessageRequestToSDK(&createMessageReq)
		if err != nil {
			return err
		}
		if createMessageSkipWait {
			api, err := apiclient.New(w.Config)
			if err != nil {
				return err
			}
			var current genie.GenieMessage
			err = api.Do(ctx, "POST", fmt.Sprintf("/api/2.0/genie/spaces/%s/conversations/%s/messages", url.PathEscape(fmt.Sprint(utils.Deref(createMessageReq.SpaceId))), url.PathEscape(fmt.Sprint(utils.Deref(createMessageReq.ConversationId)))), auth.WorkspaceIDHeaders(w.Config), nil, &createMessageReq, &current)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, current)
		}
		wait, err := w.Genie.GenieCreateConversationMessage(ctx, *sdkReq)
		if err != nil {
			return err
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for create-message to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(createMessageTimeout))
		sp.Close()
		if err != nil {
			return &cmdgen.WaitError{Err: err}
		}

		out, err := genie.GenieMessageFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start create-message-comment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createMessageCommentOverrides []func(
	*cobra.Command,
	*genie.GenieCreateMessageCommentRequest,
)

func newCreateMessageComment() *cobra.Command {
	cmd := &cobra.Command{}
	var createMessageCommentReq genie.GenieCreateMessageCommentRequest

	var createMessageCommentJson flags.JsonFlag

	cmd.Flags().Var(&createMessageCommentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create-message-comment SPACE_ID CONVERSATION_ID MESSAGE_ID CONTENT"
	cmd.Short = `*Public Preview* Add a comment to a message.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Add a comment to a message.

  Create a comment on a conversation message.

  Arguments:
    SPACE_ID: The ID associated with the Genie space.
    CONVERSATION_ID: The ID associated with the conversation.
    MESSAGE_ID: The ID associated with the message.
    CONTENT: Comment text content.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(3)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only SPACE_ID, CONVERSATION_ID, MESSAGE_ID as positional arguments. Provide 'content' in your JSON input")
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
			diags := createMessageCommentJson.Unmarshal(&createMessageCommentReq)
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
		createMessageCommentReq.SpaceId = new(args[0])
		createMessageCommentReq.ConversationId = new(args[1])
		createMessageCommentReq.MessageId = new(args[2])
		if !cmd.Flags().Changed("json") {
			createMessageCommentReq.Content = new(args[3])
		}

		sdkReq, err := genie.GenieCreateMessageCommentRequestToSDK(&createMessageCommentReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieCreateMessageComment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieMessageCommentFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createMessageCommentOverrides {
		fn(cmd, &createMessageCommentReq)
	}

	return cmd
}

// start create-space command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSpaceOverrides []func(
	*cobra.Command,
	*genie.GenieCreateSpaceRequest,
)

func newCreateSpace() *cobra.Command {
	cmd := &cobra.Command{}
	var createSpaceReq genie.GenieCreateSpaceRequest

	var createSpaceJson flags.JsonFlag

	cmd.Flags().Var(&createSpaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createSpaceReq.Description, "description", `Optional description.`)
	flags.StringPtrVar(cmd.Flags(), &createSpaceReq.ParentPath, "parent-path", `Parent folder path where the space will be registered.`)
	flags.StringPtrVar(cmd.Flags(), &createSpaceReq.Title, "title", `Optional title override.`)

	cmd.Use = "create-space WAREHOUSE_ID SERIALIZED_SPACE"
	cmd.Short = `Create a Genie space.`
	cmd.Long = `Create a Genie space.

  Creates a Genie space from a serialized payload.

  Arguments:
    WAREHOUSE_ID: Warehouse to associate with the new space
    SERIALIZED_SPACE: The contents of the Genie Space in serialized string form. Use the [Get
      Genie Space](:method:genie/getspace) API to retrieve an example response,
      which includes the serialized_space field. This field provides the
      structure of the JSON string that represents the space's layout and
      components.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'warehouse_id', 'serialized_space' in your JSON input")
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
			diags := createSpaceJson.Unmarshal(&createSpaceReq)
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
			createSpaceReq.WarehouseId = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createSpaceReq.SerializedSpace = new(args[1])
		}

		sdkReq, err := genie.GenieCreateSpaceRequestToSDK(&createSpaceReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.CreateSpace(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieSpaceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkgenie.GenieDeleteConversationRequest,
)

func newDeleteConversation() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteConversationReq sdkgenie.GenieDeleteConversationRequest

	cmd.Use = "delete-conversation SPACE_ID CONVERSATION_ID"
	cmd.Short = `Delete a conversation.`
	cmd.Long = `Delete a conversation.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the conversation is located.
    CONVERSATION_ID: The ID of the conversation to delete.`

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

		deleteConversationReq.SpaceId = new(args[0])
		deleteConversationReq.ConversationId = new(args[1])

		sdkReq := &deleteConversationReq
		err = w.Genie.GenieDeleteConversation(ctx, *sdkReq)
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
	*sdkgenie.GenieDeleteConversationMessageRequest,
)

func newDeleteConversationMessage() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteConversationMessageReq sdkgenie.GenieDeleteConversationMessageRequest

	cmd.Use = "delete-conversation-message SPACE_ID CONVERSATION_ID MESSAGE_ID"
	cmd.Short = `Delete a conversation message.`
	cmd.Long = `Delete a conversation message.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the message is located.
    CONVERSATION_ID: The ID associated with the conversation.
    MESSAGE_ID: The ID associated with the message to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteConversationMessageReq.SpaceId = new(args[0])
		deleteConversationMessageReq.ConversationId = new(args[1])
		deleteConversationMessageReq.MessageId = new(args[2])

		sdkReq := &deleteConversationMessageReq
		err = w.Genie.GenieDeleteConversationMessage(ctx, *sdkReq)
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

// start download-message-attachment-visualization command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var downloadMessageAttachmentVisualizationOverrides []func(
	*cobra.Command,
	*sdkgenie.DownloadMessageAttachmentVisualizationRequest,
)

func newDownloadMessageAttachmentVisualization() *cobra.Command {
	cmd := &cobra.Command{}
	var downloadMessageAttachmentVisualizationReq sdkgenie.DownloadMessageAttachmentVisualizationRequest

	cmd.Use = "download-message-attachment-visualization NAME"
	cmd.Short = `Download an attachment visualization.`
	cmd.Long = `Download an attachment visualization.

  Download a rendered image of a message visualization attachment. The response
  body is the raw PNG image, not a JSON payload. This is only available if the
  attachment is a visualization and the message status is COMPLETED.

  Arguments:
    NAME: The resource name of the attachment to render, in the format
      spaces/{space_id}/conversations/{conversation_id}/messages/{message_id}/attachments/{attachment_id}.`

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

		downloadMessageAttachmentVisualizationReq.Name = new(args[0])

		sdkReq := &downloadMessageAttachmentVisualizationReq
		response, err := w.Genie.DownloadMessageAttachmentVisualization(ctx, *sdkReq)
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
	for _, fn := range downloadMessageAttachmentVisualizationOverrides {
		fn(cmd, &downloadMessageAttachmentVisualizationReq)
	}

	return cmd
}

// start execute-message-attachment-query command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var executeMessageAttachmentQueryOverrides []func(
	*cobra.Command,
	*genie.GenieExecuteMessageAttachmentQueryRequest,
)

func newExecuteMessageAttachmentQuery() *cobra.Command {
	cmd := &cobra.Command{}
	var executeMessageAttachmentQueryReq genie.GenieExecuteMessageAttachmentQueryRequest

	cmd.Use = "execute-message-attachment-query SPACE_ID CONVERSATION_ID MESSAGE_ID ATTACHMENT_ID"
	cmd.Short = `Execute an attachment SQL query.`
	cmd.Long = `Execute an attachment SQL query.

  Execute the SQL for a message query attachment. Use this API when the query
  attachment has expired and needs to be re-executed.

  Arguments:
    SPACE_ID: Genie space ID
    CONVERSATION_ID: Conversation ID
    MESSAGE_ID: Message ID
    ATTACHMENT_ID: Attachment ID`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		executeMessageAttachmentQueryReq.SpaceId = new(args[0])
		executeMessageAttachmentQueryReq.ConversationId = new(args[1])
		executeMessageAttachmentQueryReq.MessageId = new(args[2])
		executeMessageAttachmentQueryReq.AttachmentId = new(args[3])

		sdkReq, err := genie.GenieExecuteMessageAttachmentQueryRequestToSDK(&executeMessageAttachmentQueryReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieExecuteMessageAttachmentQuery(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieGetMessageQueryResultResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*genie.GenieExecuteMessageQueryRequest,
)

func newExecuteMessageQuery() *cobra.Command {
	cmd := &cobra.Command{}
	var executeMessageQueryReq genie.GenieExecuteMessageQueryRequest

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
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		executeMessageQueryReq.SpaceId = new(args[0])
		executeMessageQueryReq.ConversationId = new(args[1])
		executeMessageQueryReq.MessageId = new(args[2])

		sdkReq, err := genie.GenieExecuteMessageQueryRequestToSDK(&executeMessageQueryReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieExecuteMessageQuery(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieGetMessageQueryResultResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*genie.GenieGenerateDownloadFullQueryResultRequest,
)

func newGenerateDownloadFullQueryResult() *cobra.Command {
	cmd := &cobra.Command{}
	var generateDownloadFullQueryResultReq genie.GenieGenerateDownloadFullQueryResultRequest

	cmd.Use = "generate-download-full-query-result SPACE_ID CONVERSATION_ID MESSAGE_ID ATTACHMENT_ID"
	cmd.Short = `Start a full query-result download.`
	cmd.Long = `Start a full query-result download.

  Initiates a new SQL execution and returns a download_id and
  download_id_signature that you can use to track the progress of the
  download. The query result is stored in an external link and can be retrieved
  using the [Get Download Full Query
  Result](:method:genie/getdownloadfullqueryresult) API. Both download_id and
  download_id_signature must be provided when calling the Get endpoint.

  ----

  ### **Warning: Databricks strongly recommends that you protect the URLs that
  are returned by the EXTERNAL_LINKS disposition.**

  When you use the EXTERNAL_LINKS disposition, a short-lived cloud-storage URL
  is generated to download the results. The URL contains temporary access
  credentials, so protect it and do not set an Authorization header in the
  download request.

  See [Execute Statement](:method:statementexecution/executestatement) for more
  details.

  ----

  Arguments:
    SPACE_ID: Genie space ID
    CONVERSATION_ID: Conversation ID
    MESSAGE_ID: Message ID
    ATTACHMENT_ID: Attachment ID`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		generateDownloadFullQueryResultReq.SpaceId = new(args[0])
		generateDownloadFullQueryResultReq.ConversationId = new(args[1])
		generateDownloadFullQueryResultReq.MessageId = new(args[2])
		generateDownloadFullQueryResultReq.AttachmentId = new(args[3])

		sdkReq, err := genie.GenieGenerateDownloadFullQueryResultRequestToSDK(&generateDownloadFullQueryResultReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieGenerateDownloadFullQueryResult(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieGenerateDownloadFullQueryResultResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start genie-cancel-response command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var genieCancelResponseOverrides []func(
	*cobra.Command,
	*genie.GenieCancelResponseRequest,
)

func newGenieCancelResponse() *cobra.Command {
	cmd := &cobra.Command{}
	var genieCancelResponseReq genie.GenieCancelResponseRequest

	cmd.Use = "genie-cancel-response AGENT_ID CONVERSATION_ID RESPONSE_ID"
	cmd.Short = `Agent mode: Cancel a response.`
	cmd.Long = `Agent mode: Cancel a response.

  Cancels an in-flight agent-mode response. response_id is the id returned in
  the response.created event from the agent-mode responses endpoint. The
  response stops at the next agent boundary and its terminal state is returned.

  Arguments:
    AGENT_ID: The ID of the Genie agent (synonymous with the Genie space ID).
    CONVERSATION_ID: The ID of the conversation containing the response.
    RESPONSE_ID: The ID of the response to cancel (the id from the response.created
      event).`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		genieCancelResponseReq.AgentId = new(args[0])
		genieCancelResponseReq.ConversationId = new(args[1])
		genieCancelResponseReq.ResponseId = new(args[2])

		sdkReq, err := genie.GenieCancelResponseRequestToSDK(&genieCancelResponseReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieCancelResponse(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieMessageFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range genieCancelResponseOverrides {
		fn(cmd, &genieCancelResponseReq)
	}

	return cmd
}

// start genie-create-eval-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var genieCreateEvalRunOverrides []func(
	*cobra.Command,
	*genie.GenieCreateEvalRunRequest,
)

func newGenieCreateEvalRun() *cobra.Command {
	cmd := &cobra.Command{}
	var genieCreateEvalRunReq genie.GenieCreateEvalRunRequest

	var genieCreateEvalRunJson flags.JsonFlag

	cmd.Flags().Var(&genieCreateEvalRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: benchmark_question_ids

	cmd.Use = "genie-create-eval-run SPACE_ID"
	cmd.Short = `*Beta* Start a benchmark evaluation run.`
	cmd.Long = `This command is in Beta and may change without notice.

Start a benchmark evaluation run.

  Creates and runs chat-mode evaluations for multiple benchmark questions in a
  Genie space.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the evaluations will be
      executed.`

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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := genieCreateEvalRunJson.Unmarshal(&genieCreateEvalRunReq)
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
		genieCreateEvalRunReq.SpaceId = new(args[0])

		sdkReq, err := genie.GenieCreateEvalRunRequestToSDK(&genieCreateEvalRunReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieCreateEvalRun(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieEvalRunResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range genieCreateEvalRunOverrides {
		fn(cmd, &genieCreateEvalRunReq)
	}

	return cmd
}

// start genie-get-eval-result-details command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var genieGetEvalResultDetailsOverrides []func(
	*cobra.Command,
	*sdkgenie.GenieGetEvalResultDetailsRequest,
)

func newGenieGetEvalResultDetails() *cobra.Command {
	cmd := &cobra.Command{}
	var genieGetEvalResultDetailsReq sdkgenie.GenieGetEvalResultDetailsRequest

	cmd.Use = "genie-get-eval-result-details SPACE_ID EVAL_RUN_ID RESULT_ID"
	cmd.Short = `*Beta* Get benchmark evaluation result details.`
	cmd.Long = `This command is in Beta and may change without notice.

Get benchmark evaluation result details.

  Get details for evaluation results.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the evaluation run is
      located.
    EVAL_RUN_ID: The unique identifier for the evaluation run.
    RESULT_ID: The unique identifier for the evaluation result.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		genieGetEvalResultDetailsReq.SpaceId = new(args[0])
		genieGetEvalResultDetailsReq.EvalRunId = new(args[1])
		genieGetEvalResultDetailsReq.ResultId = new(args[2])

		sdkReq := &genieGetEvalResultDetailsReq
		response, err := w.Genie.GenieGetEvalResultDetails(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieEvalResultDetailsFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range genieGetEvalResultDetailsOverrides {
		fn(cmd, &genieGetEvalResultDetailsReq)
	}

	return cmd
}

// start genie-get-eval-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var genieGetEvalRunOverrides []func(
	*cobra.Command,
	*sdkgenie.GenieGetEvalRunRequest,
)

func newGenieGetEvalRun() *cobra.Command {
	cmd := &cobra.Command{}
	var genieGetEvalRunReq sdkgenie.GenieGetEvalRunRequest

	cmd.Use = "genie-get-eval-run SPACE_ID EVAL_RUN_ID"
	cmd.Short = `*Beta* Get a benchmark evaluation run.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a benchmark evaluation run.

  Get evaluation run details.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the evaluation run is
      located.
    EVAL_RUN_ID: `

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
		_ = w

		genieGetEvalRunReq.SpaceId = new(args[0])
		genieGetEvalRunReq.EvalRunId = new(args[1])

		sdkReq := &genieGetEvalRunReq
		response, err := w.Genie.GenieGetEvalRun(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieEvalRunResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range genieGetEvalRunOverrides {
		fn(cmd, &genieGetEvalRunReq)
	}

	return cmd
}

// start genie-list-eval-results command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var genieListEvalResultsOverrides []func(
	*cobra.Command,
	*genie.GenieListEvalResultsRequest,
)

func newGenieListEvalResults() *cobra.Command {
	cmd := &cobra.Command{}
	var genieListEvalResultsReq genie.GenieListEvalResultsRequest

	flags.IntPtrVar(cmd.Flags(), &genieListEvalResultsReq.PageSize, "page-size", `Maximum number of eval results to return per page.`)
	flags.StringPtrVar(cmd.Flags(), &genieListEvalResultsReq.PageToken, "page-token", `Opaque token to retrieve the next page of results.`)

	cmd.Use = "genie-list-eval-results SPACE_ID EVAL_RUN_ID"
	cmd.Short = `*Beta* List results for a benchmark evaluation run.`
	cmd.Long = `This command is in Beta and may change without notice.

List results for a benchmark evaluation run.

  List evaluation results for a specific evaluation run.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the evaluation run is
      located.
    EVAL_RUN_ID: The unique identifier for the evaluation run.`

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
		_ = w

		genieListEvalResultsReq.SpaceId = new(args[0])
		genieListEvalResultsReq.EvalRunId = new(args[1])

		sdkReq, err := genie.GenieListEvalResultsRequestToSDK(&genieListEvalResultsReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieListEvalResults(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieListEvalResultsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range genieListEvalResultsOverrides {
		fn(cmd, &genieListEvalResultsReq)
	}

	return cmd
}

// start genie-list-eval-runs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var genieListEvalRunsOverrides []func(
	*cobra.Command,
	*genie.GenieListEvalRunsRequest,
)

func newGenieListEvalRuns() *cobra.Command {
	cmd := &cobra.Command{}
	var genieListEvalRunsReq genie.GenieListEvalRunsRequest

	flags.IntPtrVar(cmd.Flags(), &genieListEvalRunsReq.PageSize, "page-size", `Maximum number of evaluation runs to return per page.`)
	flags.StringPtrVar(cmd.Flags(), &genieListEvalRunsReq.PageToken, "page-token", `Token to get the next page of results.`)

	cmd.Use = "genie-list-eval-runs SPACE_ID"
	cmd.Short = `*Beta* List benchmark evaluation runs.`
	cmd.Long = `This command is in Beta and may change without notice.

List benchmark evaluation runs.

  Lists all evaluation runs in a space.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the evaluation run is
      located.`

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
		_ = w

		genieListEvalRunsReq.SpaceId = new(args[0])

		sdkReq, err := genie.GenieListEvalRunsRequestToSDK(&genieListEvalRunsReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieListEvalRuns(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieListEvalRunsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range genieListEvalRunsOverrides {
		fn(cmd, &genieListEvalRunsReq)
	}

	return cmd
}

// start get-download-full-query-result command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDownloadFullQueryResultOverrides []func(
	*cobra.Command,
	*genie.GenieGetDownloadFullQueryResultRequest,
)

func newGetDownloadFullQueryResult() *cobra.Command {
	cmd := &cobra.Command{}
	var getDownloadFullQueryResultReq genie.GenieGetDownloadFullQueryResultRequest

	cmd.Use = "get-download-full-query-result SPACE_ID CONVERSATION_ID MESSAGE_ID ATTACHMENT_ID DOWNLOAD_ID DOWNLOAD_ID_SIGNATURE"
	cmd.Short = `Get full query-result download status.`
	cmd.Long = `Get full query-result download status.

  After [Generating a Full Query Result
  Download](:method:genie/generatedownloadfullqueryresult) and successfully
  receiving a download_id and download_id_signature, use this API to poll
  the download progress. Both download_id and download_id_signature are
  required to call this endpoint. When the download is complete, the API returns
  the result in the EXTERNAL_LINKS disposition, containing one or more
  external links to the query result files.

  ----

  ### **Warning: Databricks strongly recommends that you protect the URLs that
  are returned by the EXTERNAL_LINKS disposition.**

  When you use the EXTERNAL_LINKS disposition, a short-lived cloud-storage URL
  is generated to download the results. The URL contains temporary access
  credentials, so protect it and do not set an Authorization header in the
  download request.

  See [Execute Statement](:method:statementexecution/executestatement) for more
  details.

  ----

  Arguments:
    SPACE_ID: Genie space ID
    CONVERSATION_ID: Conversation ID
    MESSAGE_ID: Message ID
    ATTACHMENT_ID: Attachment ID
    DOWNLOAD_ID: Download ID. This ID is provided by the [Generate Download
      endpoint](:method:genie/generateDownloadFullQueryResult)
    DOWNLOAD_ID_SIGNATURE: JWT signature for the download_id to ensure secure access to query results`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(6)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getDownloadFullQueryResultReq.SpaceId = new(args[0])
		getDownloadFullQueryResultReq.ConversationId = new(args[1])
		getDownloadFullQueryResultReq.MessageId = new(args[2])
		getDownloadFullQueryResultReq.AttachmentId = new(args[3])
		getDownloadFullQueryResultReq.DownloadId = new(args[4])
		getDownloadFullQueryResultReq.DownloadIdSignature = new(args[5])

		sdkReq, err := genie.GenieGetDownloadFullQueryResultRequestToSDK(&getDownloadFullQueryResultReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieGetDownloadFullQueryResult(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieGetDownloadFullQueryResultResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkgenie.GenieGetConversationMessageRequest,
)

func newGetMessage() *cobra.Command {
	cmd := &cobra.Command{}
	var getMessageReq sdkgenie.GenieGetConversationMessageRequest

	cmd.Use = "get-message SPACE_ID CONVERSATION_ID MESSAGE_ID"
	cmd.Short = `Get a conversation message.`
	cmd.Long = `Get a conversation message.

  Gets a message from a chat-mode or agent-mode conversation. For a complete
  agent-mode transcript, use the List conversation items endpoint.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the target conversation is
      located.
    CONVERSATION_ID: The ID associated with the target conversation.
    MESSAGE_ID: The ID associated with the target message from the identified
      conversation.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getMessageReq.SpaceId = new(args[0])
		getMessageReq.ConversationId = new(args[1])
		getMessageReq.MessageId = new(args[2])

		sdkReq := &getMessageReq
		response, err := w.Genie.GenieGetConversationMessage(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieMessageFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkgenie.GenieGetMessageAttachmentQueryResultRequest,
)

func newGetMessageAttachmentQueryResult() *cobra.Command {
	cmd := &cobra.Command{}
	var getMessageAttachmentQueryResultReq sdkgenie.GenieGetMessageAttachmentQueryResultRequest

	cmd.Use = "get-message-attachment-query-result SPACE_ID CONVERSATION_ID MESSAGE_ID ATTACHMENT_ID"
	cmd.Short = `Get an attachment SQL result.`
	cmd.Long = `Get an attachment SQL result.

  Get the result of SQL query if the message has a query attachment. This is
  only available if a message has a query attachment and the message status is
  EXECUTING_QUERY OR COMPLETED.

  Arguments:
    SPACE_ID: Genie space ID
    CONVERSATION_ID: Conversation ID
    MESSAGE_ID: Message ID
    ATTACHMENT_ID: Attachment ID`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getMessageAttachmentQueryResultReq.SpaceId = new(args[0])
		getMessageAttachmentQueryResultReq.ConversationId = new(args[1])
		getMessageAttachmentQueryResultReq.MessageId = new(args[2])
		getMessageAttachmentQueryResultReq.AttachmentId = new(args[3])

		sdkReq := &getMessageAttachmentQueryResultReq
		response, err := w.Genie.GenieGetMessageAttachmentQueryResult(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieGetMessageQueryResultResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkgenie.GenieGetMessageQueryResultRequest,
)

func newGetMessageQueryResult() *cobra.Command {
	cmd := &cobra.Command{}
	var getMessageQueryResultReq sdkgenie.GenieGetMessageQueryResultRequest

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
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getMessageQueryResultReq.SpaceId = new(args[0])
		getMessageQueryResultReq.ConversationId = new(args[1])
		getMessageQueryResultReq.MessageId = new(args[2])

		sdkReq := &getMessageQueryResultReq
		response, err := w.Genie.GenieGetMessageQueryResult(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieGetMessageQueryResultResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkgenie.GenieGetQueryResultByAttachmentRequest,
)

func newGetMessageQueryResultByAttachment() *cobra.Command {
	cmd := &cobra.Command{}
	var getMessageQueryResultByAttachmentReq sdkgenie.GenieGetQueryResultByAttachmentRequest

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
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getMessageQueryResultByAttachmentReq.SpaceId = new(args[0])
		getMessageQueryResultByAttachmentReq.ConversationId = new(args[1])
		getMessageQueryResultByAttachmentReq.MessageId = new(args[2])
		getMessageQueryResultByAttachmentReq.AttachmentId = new(args[3])

		sdkReq := &getMessageQueryResultByAttachmentReq
		response, err := w.Genie.GenieGetQueryResultByAttachment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieGetMessageQueryResultResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*genie.GenieGetSpaceRequest,
)

func newGetSpace() *cobra.Command {
	cmd := &cobra.Command{}
	var getSpaceReq genie.GenieGetSpaceRequest

	flags.BoolPtrVar(cmd.Flags(), &getSpaceReq.IncludeSerializedSpace, "include-serialized-space", `Whether to include the serialized space export in the response.`)

	cmd.Use = "get-space SPACE_ID"
	cmd.Short = `Get a Genie space.`
	cmd.Long = `Get a Genie space.

  Get details of a Genie Space.

  Arguments:
    SPACE_ID: The ID associated with the Genie space`

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

		getSpaceReq.SpaceId = new(args[0])

		sdkReq, err := genie.GenieGetSpaceRequestToSDK(&getSpaceReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieGetSpace(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieSpaceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start list-conversation-comments command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listConversationCommentsOverrides []func(
	*cobra.Command,
	*genie.GenieListConversationCommentsRequest,
)

func newListConversationComments() *cobra.Command {
	cmd := &cobra.Command{}
	var listConversationCommentsReq genie.GenieListConversationCommentsRequest

	flags.IntPtrVar(cmd.Flags(), &listConversationCommentsReq.PageSize, "page-size", `Maximum number of comments to return per page.`)
	flags.StringPtrVar(cmd.Flags(), &listConversationCommentsReq.PageToken, "page-token", `Pagination token for getting the next page of results.`)

	cmd.Use = "list-conversation-comments SPACE_ID CONVERSATION_ID"
	cmd.Short = `*Public Preview* List all comments in a conversation.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List all comments in a conversation.

  List all comments across all messages in a conversation.

  Arguments:
    SPACE_ID: The ID associated with the Genie space.
    CONVERSATION_ID: The ID associated with the conversation.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listConversationCommentsReq.SpaceId = new(args[0])
		listConversationCommentsReq.ConversationId = new(args[1])

		sdkReq, err := genie.GenieListConversationCommentsRequestToSDK(&listConversationCommentsReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieListConversationComments(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieListConversationCommentsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listConversationCommentsOverrides {
		fn(cmd, &listConversationCommentsReq)
	}

	return cmd
}

// start list-conversation-messages command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listConversationMessagesOverrides []func(
	*cobra.Command,
	*genie.GenieListConversationMessagesRequest,
)

func newListConversationMessages() *cobra.Command {
	cmd := &cobra.Command{}
	var listConversationMessagesReq genie.GenieListConversationMessagesRequest

	flags.IntPtrVar(cmd.Flags(), &listConversationMessagesReq.PageSize, "page-size", `Maximum number of messages to return per page.`)
	flags.StringPtrVar(cmd.Flags(), &listConversationMessagesReq.PageToken, "page-token", `Token to get the next page of results.`)

	cmd.Use = "list-conversation-messages SPACE_ID CONVERSATION_ID"
	cmd.Short = `List conversation messages.`
	cmd.Long = `List conversation messages.

  Lists messages in a chat-mode or agent-mode conversation. Agent-mode messages
  are returned as GenieMessage projections. Use the List conversation items
  endpoint for the complete reasoning and tool-call history.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the conversation is located
    CONVERSATION_ID: The ID of the conversation to list messages from`

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

		listConversationMessagesReq.SpaceId = new(args[0])
		listConversationMessagesReq.ConversationId = new(args[1])

		sdkReq, err := genie.GenieListConversationMessagesRequestToSDK(&listConversationMessagesReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieListConversationMessages(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieListConversationMessagesResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*genie.GenieListConversationsRequest,
)

func newListConversations() *cobra.Command {
	cmd := &cobra.Command{}
	var listConversationsReq genie.GenieListConversationsRequest

	flags.BoolPtrVar(cmd.Flags(), &listConversationsReq.IncludeAll, "include-all", `Include all conversations in the space across all users.`)
	flags.IntPtrVar(cmd.Flags(), &listConversationsReq.PageSize, "page-size", `Maximum number of conversations to return per page.`)
	flags.StringPtrVar(cmd.Flags(), &listConversationsReq.PageToken, "page-token", `Token to get the next page of results.`)

	cmd.Use = "list-conversations SPACE_ID"
	cmd.Short = `List conversations.`
	cmd.Long = `List conversations.

  Get a list of conversations in a Genie Space.

  Arguments:
    SPACE_ID: The ID of the Genie space to retrieve conversations from.`

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

		listConversationsReq.SpaceId = new(args[0])

		sdkReq, err := genie.GenieListConversationsRequestToSDK(&listConversationsReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieListConversations(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieListConversationsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start list-message-comments command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listMessageCommentsOverrides []func(
	*cobra.Command,
	*genie.GenieListMessageCommentsRequest,
)

func newListMessageComments() *cobra.Command {
	cmd := &cobra.Command{}
	var listMessageCommentsReq genie.GenieListMessageCommentsRequest

	flags.IntPtrVar(cmd.Flags(), &listMessageCommentsReq.PageSize, "page-size", `Maximum number of comments to return per page.`)
	flags.StringPtrVar(cmd.Flags(), &listMessageCommentsReq.PageToken, "page-token", `Pagination token for getting the next page of results.`)

	cmd.Use = "list-message-comments SPACE_ID CONVERSATION_ID MESSAGE_ID"
	cmd.Short = `*Public Preview* List comments on a message.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List comments on a message.

  List comments on a specific conversation message.

  Arguments:
    SPACE_ID: The ID associated with the Genie space.
    CONVERSATION_ID: The ID associated with the conversation.
    MESSAGE_ID: The ID associated with the message.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listMessageCommentsReq.SpaceId = new(args[0])
		listMessageCommentsReq.ConversationId = new(args[1])
		listMessageCommentsReq.MessageId = new(args[2])

		sdkReq, err := genie.GenieListMessageCommentsRequestToSDK(&listMessageCommentsReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieListMessageComments(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieListMessageCommentsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listMessageCommentsOverrides {
		fn(cmd, &listMessageCommentsReq)
	}

	return cmd
}

// start list-spaces command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSpacesOverrides []func(
	*cobra.Command,
	*genie.GenieListSpacesRequest,
)

func newListSpaces() *cobra.Command {
	cmd := &cobra.Command{}
	var listSpacesReq genie.GenieListSpacesRequest

	flags.IntPtrVar(cmd.Flags(), &listSpacesReq.PageSize, "page-size", `Maximum number of spaces to return per page.`)
	flags.StringPtrVar(cmd.Flags(), &listSpacesReq.PageToken, "page-token", `Pagination token for getting the next page of results.`)

	cmd.Use = "list-spaces"
	cmd.Short = `List Genie spaces.`
	cmd.Long = `List Genie spaces.

  Get list of Genie Spaces.`

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

		sdkReq, err := genie.GenieListSpacesRequestToSDK(&listSpacesReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.GenieListSpaces(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieListSpacesResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*genie.GenieSendMessageFeedbackRequest,
)

func newSendMessageFeedback() *cobra.Command {
	cmd := &cobra.Command{}
	var sendMessageFeedbackReq genie.GenieSendMessageFeedbackRequest

	var sendMessageFeedbackJson flags.JsonFlag

	cmd.Flags().Var(&sendMessageFeedbackJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &sendMessageFeedbackReq.Comment, "comment", `Optional text feedback that will be stored as a comment.`)

	cmd.Use = "send-message-feedback SPACE_ID CONVERSATION_ID MESSAGE_ID RATING"
	cmd.Short = `Send feedback on a message.`
	cmd.Long = `Send feedback on a message.

  Sends feedback for a message in a chat-mode or agent-mode conversation.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where the message is located.
    CONVERSATION_ID: The ID associated with the conversation.
    MESSAGE_ID: The ID associated with the message to provide feedback for.
    RATING: The rating (POSITIVE, NEGATIVE, or NONE).
      Supported values: [NEGATIVE, NONE, POSITIVE]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(3)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only SPACE_ID, CONVERSATION_ID, MESSAGE_ID as positional arguments. Provide 'rating' in your JSON input")
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
			diags := sendMessageFeedbackJson.Unmarshal(&sendMessageFeedbackReq)
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
		sendMessageFeedbackReq.SpaceId = new(args[0])
		sendMessageFeedbackReq.ConversationId = new(args[1])
		sendMessageFeedbackReq.MessageId = new(args[2])
		if !cmd.Flags().Changed("json") {
			sendMessageFeedbackReq.Rating = genie.GenieFeedbackRating(args[3])
		}

		sdkReq, err := genie.GenieSendMessageFeedbackRequestToSDK(&sendMessageFeedbackReq)
		if err != nil {
			return err
		}
		err = w.Genie.GenieSendMessageFeedback(ctx, *sdkReq)
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
	*genie.GenieStartConversationRequest,
)

func newStartConversation() *cobra.Command {
	cmd := &cobra.Command{}
	var startConversationReq genie.GenieStartConversationRequest

	var startConversationJson flags.JsonFlag

	var startConversationSkipWait bool
	var startConversationTimeout time.Duration

	cmd.Flags().BoolVar(&startConversationSkipWait, "no-wait", startConversationSkipWait, `do not wait to reach COMPLETED state`)
	cmd.Flags().DurationVar(&startConversationTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach COMPLETED state`)

	cmd.Flags().Var(&startConversationJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &startConversationReq.EnableVisualization, "enable-visualization", `Enable visualization generation.`)

	cmd.Use = "start-conversation SPACE_ID CONTENT"
	cmd.Short = `Chat mode: Start a conversation.`
	cmd.Long = `Chat mode: Start a conversation.

  Starts a new chat-mode conversation and sends its first message.

  Arguments:
    SPACE_ID: The ID associated with the Genie space where you want to start a
      conversation.
    CONTENT: The text of the message that starts the conversation.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only SPACE_ID as positional arguments. Provide 'content' in your JSON input")
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
			diags := startConversationJson.Unmarshal(&startConversationReq)
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
		startConversationReq.SpaceId = new(args[0])
		if !cmd.Flags().Changed("json") {
			startConversationReq.Content = new(args[1])
		}

		sdkReq, err := genie.GenieStartConversationRequestToSDK(&startConversationReq)
		if err != nil {
			return err
		}
		if startConversationSkipWait {
			api, err := apiclient.New(w.Config)
			if err != nil {
				return err
			}
			var current genie.GenieStartConversationResponse
			err = api.Do(ctx, "POST", fmt.Sprintf("/api/2.0/genie/spaces/%s/start-conversation", url.PathEscape(fmt.Sprint(utils.Deref(startConversationReq.SpaceId)))), auth.WorkspaceIDHeaders(w.Config), nil, &startConversationReq, &current)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, current)
		}
		wait, err := w.Genie.GenieStartConversation(ctx, *sdkReq)
		if err != nil {
			return err
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for start-conversation to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(startConversationTimeout))
		sp.Close()
		if err != nil {
			return &cmdgen.WaitError{Err: err}
		}

		out, err := genie.GenieMessageFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkgenie.GenieTrashSpaceRequest,
)

func newTrashSpace() *cobra.Command {
	cmd := &cobra.Command{}
	var trashSpaceReq sdkgenie.GenieTrashSpaceRequest

	cmd.Use = "trash-space SPACE_ID"
	cmd.Short = `Trash a Genie space.`
	cmd.Long = `Trash a Genie space.

  Move a Genie Space to the trash.

  Arguments:
    SPACE_ID: The ID associated with the Genie space to be sent to the trash.`

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

		trashSpaceReq.SpaceId = new(args[0])

		sdkReq := &trashSpaceReq
		err = w.Genie.GenieTrashSpace(ctx, *sdkReq)
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
	*genie.GenieUpdateSpaceRequest,
)

func newUpdateSpace() *cobra.Command {
	cmd := &cobra.Command{}
	var updateSpaceReq genie.GenieUpdateSpaceRequest

	var updateSpaceJson flags.JsonFlag

	cmd.Flags().Var(&updateSpaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateSpaceReq.Description, "description", `Optional description.`)
	flags.StringPtrVar(cmd.Flags(), &updateSpaceReq.Etag, "etag", `ETag returned by a previous GET or UPDATE.`)
	flags.StringPtrVar(cmd.Flags(), &updateSpaceReq.ParentPath, "parent-path", `Parent workspace folder path to move this Genie space under.`)
	flags.StringPtrVar(cmd.Flags(), &updateSpaceReq.SerializedSpace, "serialized-space", `The contents of the Genie Space in serialized string form (full replacement).`)
	flags.StringPtrVar(cmd.Flags(), &updateSpaceReq.Title, "title", `Optional title override.`)
	flags.StringPtrVar(cmd.Flags(), &updateSpaceReq.WarehouseId, "warehouse-id", `Optional warehouse override.`)

	cmd.Use = "update-space SPACE_ID"
	cmd.Short = `Update a Genie space.`
	cmd.Long = `Update a Genie space.

  Updates a Genie space with a serialized payload.

  Arguments:
    SPACE_ID: Genie space ID`

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
			diags := updateSpaceJson.Unmarshal(&updateSpaceReq)
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
		updateSpaceReq.SpaceId = new(args[0])

		sdkReq, err := genie.GenieUpdateSpaceRequestToSDK(&updateSpaceReq)
		if err != nil {
			return err
		}
		response, err := w.Genie.UpdateSpace(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := genie.GenieSpaceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
