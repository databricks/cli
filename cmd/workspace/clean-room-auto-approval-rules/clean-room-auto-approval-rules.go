// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clean_room_auto_approval_rules

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/cleanrooms"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean-room-auto-approval-rules",
		Short: `Clean room auto-approval rules automatically create an approval on your behalf when an asset (e.g.`,
		Long: `Clean room auto-approval rules automatically create an approval on your behalf
  when an asset (e.g. notebook) meeting specific criteria is shared in a clean
  room.`,
		GroupID: "cleanrooms",
		Annotations: map[string]string{
			"package": "cleanrooms",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*cleanrooms.CreateCleanRoomAutoApprovalRuleRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq cleanrooms.CreateCleanRoomAutoApprovalRuleRequest
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create CLEAN_ROOM_NAME"
	cmd.Short = `Create an auto-approval rule.`
	cmd.Long = `Create an auto-approval rule.
  
  Create an auto-approval rule

  Arguments:
    CLEAN_ROOM_NAME: The name of the clean room this auto-approval rule belongs to.`

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
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		createReq.CleanRoomName = args[0]

		response, err := w.CleanRoomAutoApprovalRules.Create(ctx, createReq)
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
	*cleanrooms.DeleteCleanRoomAutoApprovalRuleRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq cleanrooms.DeleteCleanRoomAutoApprovalRuleRequest

	cmd.Use = "delete CLEAN_ROOM_NAME RULE_ID"
	cmd.Short = `Delete an auto-approval rule.`
	cmd.Long = `Delete an auto-approval rule.
  
  Delete a auto-approval rule by rule ID`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.CleanRoomName = args[0]
		deleteReq.RuleId = args[1]

		err = w.CleanRoomAutoApprovalRules.Delete(ctx, deleteReq)
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
	*cleanrooms.GetCleanRoomAutoApprovalRuleRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq cleanrooms.GetCleanRoomAutoApprovalRuleRequest

	cmd.Use = "get CLEAN_ROOM_NAME RULE_ID"
	cmd.Short = `Get an auto-approval rule.`
	cmd.Long = `Get an auto-approval rule.
  
  Get a auto-approval rule by rule ID`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.CleanRoomName = args[0]
		getReq.RuleId = args[1]

		response, err := w.CleanRoomAutoApprovalRules.Get(ctx, getReq)
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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*cleanrooms.ListCleanRoomAutoApprovalRulesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq cleanrooms.ListCleanRoomAutoApprovalRulesRequest

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Maximum number of auto-approval rules to return. Wire name: 'page_size'.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query. Wire name: 'page_token'.`)

	cmd.Use = "list CLEAN_ROOM_NAME"
	cmd.Short = `List auto-approval rules.`
	cmd.Long = `List auto-approval rules.
  
  List all auto-approval rules for the caller`

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

		response := w.CleanRoomAutoApprovalRules.List(ctx, listReq)
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

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*cleanrooms.UpdateCleanRoomAutoApprovalRuleRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq cleanrooms.UpdateCleanRoomAutoApprovalRuleRequest
	updateReq.AutoApprovalRule = cleanrooms.CleanRoomAutoApprovalRule{}
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.AutoApprovalRule.AuthorCollaboratorAlias, "author-collaborator-alias", updateReq.AutoApprovalRule.AuthorCollaboratorAlias, `Collaborator alias of the author covered by the rule. Wire name: 'author_collaborator_alias'.`)
	cmd.Flags().Var(&updateReq.AutoApprovalRule.AuthorScope, "author-scope", `Scope of authors covered by the rule. Supported values: [ANY_AUTHOR]. Wire name: 'author_scope'.`)
	cmd.Flags().StringVar(&updateReq.AutoApprovalRule.CleanRoomName, "clean-room-name", updateReq.AutoApprovalRule.CleanRoomName, `The name of the clean room this auto-approval rule belongs to. Wire name: 'clean_room_name'.`)
	cmd.Flags().StringVar(&updateReq.AutoApprovalRule.RunnerCollaboratorAlias, "runner-collaborator-alias", updateReq.AutoApprovalRule.RunnerCollaboratorAlias, `Collaborator alias of the runner covered by the rule. Wire name: 'runner_collaborator_alias'.`)

	cmd.Use = "update CLEAN_ROOM_NAME RULE_ID"
	cmd.Short = `Update an auto-approval rule.`
	cmd.Long = `Update an auto-approval rule.
  
  Update a auto-approval rule by rule ID

  Arguments:
    CLEAN_ROOM_NAME: The name of the clean room this auto-approval rule belongs to.
    RULE_ID: A generated UUID identifying the rule.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq.AutoApprovalRule)
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
		updateReq.CleanRoomName = args[0]
		updateReq.RuleId = args[1]

		response, err := w.CleanRoomAutoApprovalRules.Update(ctx, updateReq)
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

// end service CleanRoomAutoApprovalRules
