// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package genie

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
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

		// This service is being previewed; hide from help output.
		Hidden: true,
	}

	// Add methods
	cmd.AddCommand(newCreateMessage())
	cmd.AddCommand(newExecuteMessageQuery())
	cmd.AddCommand(newGetMessage())
	cmd.AddCommand(newGetMessageQueryResult())
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
  
  Create new message in [conversation](:method:genie/startconversation). The AI
  response uses all previously created messages in the conversation to respond.

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
		w := root.WorkspaceClient(ctx)

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
	cmd.Short = `Execute SQL query in a conversation message.`
	cmd.Long = `Execute SQL query in a conversation message.
  
  Execute the SQL query in the message.

  Arguments:
    SPACE_ID: Genie space ID
    CONVERSATION_ID: Conversation ID
    MESSAGE_ID: Message ID`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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
		w := root.WorkspaceClient(ctx)

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
	cmd.Short = `Get conversation message SQL query result.`
	cmd.Long = `Get conversation message SQL query result.
  
  Get the result of SQL query if the message has a query attachment. This is
  only available if a message has a query attachment and the message status is
  EXECUTING_QUERY.

  Arguments:
    SPACE_ID: Genie space ID
    CONVERSATION_ID: Conversation ID
    MESSAGE_ID: Message ID`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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
		w := root.WorkspaceClient(ctx)

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
