// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package budget_policy

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/billing"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "budget-policy",
		Short:   `A service serves REST API about Budget policies.`,
		Long:    `A service serves REST API about Budget policies`,
		GroupID: "billing",
		Annotations: map[string]string{
			"package": "billing",
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
	*billing.CreateBudgetPolicyRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq billing.CreateBudgetPolicyRequest
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: policy
	cmd.Flags().StringVar(&createReq.RequestId, "request-id", createReq.RequestId, `A unique identifier for this request.`)

	cmd.Use = "create"
	cmd.Short = `Create a budget policy.`
	cmd.Long = `Create a budget policy.
  
  Creates a new policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

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
		}

		response, err := a.BudgetPolicy.Create(ctx, createReq)
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
	*billing.DeleteBudgetPolicyRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq billing.DeleteBudgetPolicyRequest

	cmd.Use = "delete POLICY_ID"
	cmd.Short = `Delete a budget policy.`
	cmd.Long = `Delete a budget policy.
  
  Deletes a policy

  Arguments:
    POLICY_ID: The Id of the policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		deleteReq.PolicyId = args[0]

		err = a.BudgetPolicy.Delete(ctx, deleteReq)
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
	*billing.GetBudgetPolicyRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq billing.GetBudgetPolicyRequest

	cmd.Use = "get POLICY_ID"
	cmd.Short = `Get a budget policy.`
	cmd.Long = `Get a budget policy.
  
  Retrieves a policy by it's ID.

  Arguments:
    POLICY_ID: The Id of the policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		getReq.PolicyId = args[0]

		response, err := a.BudgetPolicy.Get(ctx, getReq)
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
	*billing.ListBudgetPoliciesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq billing.ListBudgetPoliciesRequest

	// TODO: complex arg: filter_by
	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `The maximum number of budget policies to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `A page token, received from a previous ListServerlessPolicies call.`)
	// TODO: complex arg: sort_spec

	cmd.Use = "list"
	cmd.Short = `List policies.`
	cmd.Long = `List policies.
  
  Lists all policies. Policies are returned in the alphabetically ascending
  order of their names.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		response := a.BudgetPolicy.List(ctx, listReq)
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
	*billing.UpdateBudgetPolicyRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq billing.UpdateBudgetPolicyRequest
	updateReq.Policy = billing.BudgetPolicy{}
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: limit_config
	// TODO: array: binding_workspace_ids
	// TODO: array: custom_tags
	cmd.Flags().StringVar(&updateReq.Policy.PolicyName, "policy-name", updateReq.Policy.PolicyName, `The name of the policy.`)

	cmd.Use = "update POLICY_ID"
	cmd.Short = `Update a budget policy.`
	cmd.Long = `Update a budget policy.
  
  Updates a policy

  Arguments:
    POLICY_ID: The Id of the policy. This field is generated by Databricks and globally
      unique.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq.Policy)
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
		updateReq.PolicyId = args[0]

		response, err := a.BudgetPolicy.Update(ctx, updateReq)
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

// end service BudgetPolicy
