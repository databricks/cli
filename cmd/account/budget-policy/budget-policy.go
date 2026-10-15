// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package budget_policy

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	budgetpolicy "github.com/databricks/cli/libs/sdktypes/budgetpolicy/v1"
	sdkbudgetpolicy "github.com/databricks/sdk-go/budgetpolicy/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budget-policy",
		Short: `*Public Preview* A service serves REST API about Budget policies.`,
		Long: `This command is in Public Preview and may change without notice.

A service serves REST API about Budget policies`,
		GroupID: "billing",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
	*budgetpolicy.CreateBudgetPolicyRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq budgetpolicy.CreateBudgetPolicyRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: policy
	flags.StringPtrVar(cmd.Flags(), &createReq.RequestId, "request-id", `A unique identifier for this request.`)

	cmd.Use = "create"
	cmd.Short = `*Public Preview* Create a budget policy.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create a budget policy.

  Creates a new policy.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

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
		}

		sdkReq, err := budgetpolicy.CreateBudgetPolicyRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.BudgetPolicy.CreateBudgetPolicy(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := budgetpolicy.BudgetPolicyFromSDK(response)
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
	*sdkbudgetpolicy.DeleteBudgetPolicyRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkbudgetpolicy.DeleteBudgetPolicyRequest

	cmd.Use = "delete POLICY_ID"
	cmd.Short = `*Public Preview* Delete a budget policy.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete a budget policy.

  Deletes a policy

  Arguments:
    POLICY_ID: The Id of the policy.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		deleteReq.PolicyId = new(args[0])

		sdkReq := &deleteReq
		err = a.BudgetPolicy.DeleteBudgetPolicy(ctx, *sdkReq)
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
	*sdkbudgetpolicy.GetBudgetPolicyRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkbudgetpolicy.GetBudgetPolicyRequest

	cmd.Use = "get POLICY_ID"
	cmd.Short = `*Public Preview* Get a budget policy.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get a budget policy.

  Retrieves a policy by it's ID.

  Arguments:
    POLICY_ID: The Id of the policy.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getReq.PolicyId = new(args[0])

		sdkReq := &getReq
		response, err := a.BudgetPolicy.GetBudgetPolicy(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := budgetpolicy.BudgetPolicyFromSDK(response)
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
	*budgetpolicy.ListBudgetPoliciesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq budgetpolicy.ListBudgetPoliciesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// TODO: complex arg: filter_by
	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", `The maximum number of budget policies to return.`)
	// TODO: complex arg: sort_spec

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `*Public Preview* List policies.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List policies.

  Lists all policies. Policies are returned in the alphabetically ascending
  order of their names.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		sdkReq, err := budgetpolicy.ListBudgetPoliciesRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := a.BudgetPolicy.ListBudgetPoliciesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, budgetpolicy.BudgetPolicyFromSDK))
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
	*budgetpolicy.UpdateBudgetPolicyRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq budgetpolicy.UpdateBudgetPolicyRequest
	updateReq.Policy = &budgetpolicy.BudgetPolicy{}

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: limit_config
	// TODO: array: binding_workspace_ids
	// TODO: array: custom_tags
	flags.StringPtrVar(cmd.Flags(), &updateReq.Policy.PolicyName, "policy-name", `The name of the policy.`)

	cmd.Use = "update POLICY_ID"
	cmd.Short = `*Public Preview* Update a budget policy.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update a budget policy.

  Updates a policy

  Arguments:
    POLICY_ID: The Id of the policy. This field is generated by Databricks and globally
      unique.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq.Policy)
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
		updateReq.Policy.PolicyId = new(args[0])

		sdkReq, err := budgetpolicy.UpdateBudgetPolicyRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := a.BudgetPolicy.UpdateBudgetPolicy(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := budgetpolicy.BudgetPolicyFromSDK(response)
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

// end service BudgetPolicy
