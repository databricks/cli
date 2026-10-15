// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clean_room_auto_approval_rules

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	cleanrooms "github.com/databricks/cli/libs/sdktypes/cleanrooms/v1"
	sdkcleanrooms "github.com/databricks/sdk-go/cleanrooms/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean-room-auto-approval-rules",
		Short: `*Beta* Clean room auto-approval rules automatically create an approval on your behalf when an asset (e.g.`,
		Long: `This command is in Beta and may change without notice.

Clean room auto-approval rules automatically create an approval on your behalf
  when an asset (e.g. notebook) meeting specific criteria is shared in a clean
  room.`,
		GroupID: "cleanrooms",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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
	cmd.Short = `*Beta* Create an auto-approval rule.`
	cmd.Long = `This command is in Beta and may change without notice.

Create an auto-approval rule.

  Create an auto-approval rule

  Arguments:
    CLEAN_ROOM_NAME: The name of the clean room this auto-approval rule belongs to.`

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
			diags := createJson.Unmarshal(&createReq)
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
		createReq.AutoApprovalRule.CleanRoomName = new(args[0])

		sdkReq, err := cleanrooms.CreateCleanRoomAutoApprovalRuleRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.CleanRooms.CreateCleanRoomAutoApprovalRule(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := cleanrooms.CleanRoomAutoApprovalRuleFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkcleanrooms.DeleteCleanRoomAutoApprovalRuleRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkcleanrooms.DeleteCleanRoomAutoApprovalRuleRequest

	cmd.Use = "delete CLEAN_ROOM_NAME RULE_ID"
	cmd.Short = `*Beta* Delete an auto-approval rule.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete an auto-approval rule.

  Delete a auto-approval rule by rule ID`

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

		deleteReq.CleanRoomName = new(args[0])
		deleteReq.RuleId = new(args[1])

		sdkReq := &deleteReq
		err = w.CleanRooms.DeleteCleanRoomAutoApprovalRule(ctx, *sdkReq)
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
	*sdkcleanrooms.GetCleanRoomAutoApprovalRuleRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkcleanrooms.GetCleanRoomAutoApprovalRuleRequest

	cmd.Use = "get CLEAN_ROOM_NAME RULE_ID"
	cmd.Short = `*Beta* Get an auto-approval rule.`
	cmd.Long = `This command is in Beta and may change without notice.

Get an auto-approval rule.

  Get a auto-approval rule by rule ID`

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

		getReq.CleanRoomName = new(args[0])
		getReq.RuleId = new(args[1])

		sdkReq := &getReq
		response, err := w.CleanRooms.GetCleanRoomAutoApprovalRule(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := cleanrooms.CleanRoomAutoApprovalRuleFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", `Maximum number of auto-approval rules to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list CLEAN_ROOM_NAME"
	cmd.Short = `*Beta* List auto-approval rules.`
	cmd.Long = `This command is in Beta and may change without notice.

List auto-approval rules.

  List all auto-approval rules for the caller`

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

		listReq.CleanRoomName = new(args[0])

		sdkReq, err := cleanrooms.ListCleanRoomAutoApprovalRulesRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.CleanRooms.ListCleanRoomAutoApprovalRulesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, cleanrooms.CleanRoomAutoApprovalRuleFromSDK))
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
	updateReq.AutoApprovalRule = &cleanrooms.CleanRoomAutoApprovalRule{}

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateReq.AutoApprovalRule.AuthorCollaboratorAlias, "author-collaborator-alias", `Collaborator alias of the author covered by the rule.`)
	flags.EnumVar(cmd.Flags(), &updateReq.AutoApprovalRule.AuthorScope, "author-scope", `Scope of authors covered by the rule. Supported values: [ANY_AUTHOR]`, "CleanRoomAutoApprovalRuleAuthorScope", "ANY_AUTHOR")
	flags.StringPtrVar(cmd.Flags(), &updateReq.AutoApprovalRule.CleanRoomName, "clean-room-name", `The name of the clean room this auto-approval rule belongs to.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.AutoApprovalRule.RunnerCollaboratorAlias, "runner-collaborator-alias", `Collaborator alias of the runner covered by the rule.`)

	cmd.Use = "update CLEAN_ROOM_NAME RULE_ID"
	cmd.Short = `*Beta* Update an auto-approval rule.`
	cmd.Long = `This command is in Beta and may change without notice.

Update an auto-approval rule.

  Update a auto-approval rule by rule ID

  Arguments:
    CLEAN_ROOM_NAME: The name of the clean room this auto-approval rule belongs to.
    RULE_ID: A generated UUID identifying the rule.`

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

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq.AutoApprovalRule)
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
		updateReq.AutoApprovalRule.CleanRoomName = new(args[0])
		updateReq.AutoApprovalRule.RuleId = new(args[1])

		sdkReq, err := cleanrooms.UpdateCleanRoomAutoApprovalRuleRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.CleanRooms.UpdateCleanRoomAutoApprovalRule(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := cleanrooms.CleanRoomAutoApprovalRuleFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
