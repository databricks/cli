// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package budgets

import (
	"errors"
	"fmt"

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
		Use:   "budgets",
		Short: `These APIs manage budget configurations for this account.`,
		Long: `These APIs manage budget configurations for this account. Budgets enable you
  to monitor usage across your account. You can set up budgets to either track
  account-wide spending, or apply filters to track the spending of specific
  teams, projects, or workspaces.`,
		GroupID: "billing",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetBudgetSpendStatus())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListBudgetSpendStatuses())
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
	*billing.CreateBudgetConfigurationRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq billing.CreateBudgetConfigurationRequest
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create new budget.`
	cmd.Long = `Create new budget.

  Create a new budget configuration for an account. For full details, see
  https://docs.databricks.com/en/admin/account-settings/budgets.html.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := a.Budgets.Create(ctx, createReq)
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
	*billing.DeleteBudgetConfigurationRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq billing.DeleteBudgetConfigurationRequest

	cmd.Use = "delete BUDGET_ID"
	cmd.Short = `Delete budget.`
	cmd.Long = `Delete budget.

  Deletes a budget configuration for an account. Both account and budget
  configuration are specified by ID. This cannot be undone.

  Arguments:
    BUDGET_ID: The Databricks budget configuration ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		deleteReq.BudgetId = args[0]

		err = a.Budgets.Delete(ctx, deleteReq)
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
	*billing.GetBudgetConfigurationRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq billing.GetBudgetConfigurationRequest

	cmd.Use = "get BUDGET_ID"
	cmd.Short = `Get budget.`
	cmd.Long = `Get budget.

  Gets a budget configuration for an account. Both account and budget
  configuration are specified by ID.

  Arguments:
    BUDGET_ID: The budget configuration ID`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		getReq.BudgetId = args[0]

		response, err := a.Budgets.Get(ctx, getReq)
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

// start get-budget-spend-status command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getBudgetSpendStatusOverrides []func(
	*cobra.Command,
	*billing.GetBudgetSpendStatusRequest,
)

func newGetBudgetSpendStatus() *cobra.Command {
	cmd := &cobra.Command{}

	var getBudgetSpendStatusReq billing.GetBudgetSpendStatusRequest

	cmd.Use = "get-budget-spend-status NAME"
	cmd.Short = `*Beta* Get the calling user's spend status for a budget.`
	cmd.Long = `This command is in Beta and may change without notice.

Get the calling user's spend status for a budget.

  Get the calling user's live spend against the blocking thresholds of a single
  Unity Gateway budget. Any account user can call this for their own spend.
  Spend is reported in USD at list price and is near-real-time, so it may differ
  slightly from billing data. Returns NOT_FOUND if the budget doesn't exist or
  has no blocking thresholds.

  Arguments:
    NAME: The resource name of the budget spend status to read, of the form
      accounts/{account_id}/budget-spend-statuses/{budget_id}. A budget_id
      that isn't a valid UUID fails with INVALID_PARAMETER_VALUE.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		getBudgetSpendStatusReq.Name = args[0]

		response, err := a.Budgets.GetBudgetSpendStatus(ctx, getBudgetSpendStatusReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getBudgetSpendStatusOverrides {
		fn(cmd, &getBudgetSpendStatusReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*billing.ListBudgetConfigurationsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq billing.ListBudgetConfigurationsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `Get all budgets.`
	cmd.Long = `Get all budgets.

  Gets all budgets associated with this account.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		response := a.Budgets.List(ctx, listReq)
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

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

// start list-budget-spend-statuses command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listBudgetSpendStatusesOverrides []func(
	*cobra.Command,
	*billing.ListBudgetSpendStatusesRequest,
)

func newListBudgetSpendStatuses() *cobra.Command {
	cmd := &cobra.Command{}

	var listBudgetSpendStatusesReq billing.ListBudgetSpendStatusesRequest

	cmd.Flags().StringVar(&listBudgetSpendStatusesReq.Filter, "filter", listBudgetSpendStatusesReq.Filter, `A filter expression that narrows the returned budgets by workspace.`)
	cmd.Flags().IntVar(&listBudgetSpendStatusesReq.PageSize, "page-size", listBudgetSpendStatusesReq.PageSize, `The maximum number of spend statuses to return per page.`)
	cmd.Flags().StringVar(&listBudgetSpendStatusesReq.PageToken, "page-token", listBudgetSpendStatusesReq.PageToken, `A page token from a previous ListBudgetSpendStatuses response, used to fetch the next page.`)

	cmd.Use = "list-budget-spend-statuses PARENT"
	cmd.Short = `*Beta* List the calling user's budget spend statuses.`
	cmd.Long = `This command is in Beta and may change without notice.

List the calling user's budget spend statuses.

  List the calling user's live spend against the blocking thresholds across the
  account's Unity Gateway budgets. Any account user can call this for their own
  spend. Spend is reported in USD at list price and is near-real-time, so it may
  differ slightly from billing data. By default, returns every budget in the
  account that has a blocking threshold. Use filter to narrow results by
  workspace.

  Arguments:
    PARENT: The parent account whose budget spend statuses to list, of the form
      accounts/{account_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		listBudgetSpendStatusesReq.Parent = args[0]

		response, err := a.Budgets.ListBudgetSpendStatuses(ctx, listBudgetSpendStatusesReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listBudgetSpendStatusesOverrides {
		fn(cmd, &listBudgetSpendStatusesReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*billing.UpdateBudgetConfigurationRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq billing.UpdateBudgetConfigurationRequest
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update BUDGET_ID"
	cmd.Short = `Modify budget.`
	cmd.Long = `Modify budget.

  Updates a budget configuration for an account. Both account and budget
  configuration are specified by ID.

  Arguments:
    BUDGET_ID: The Databricks budget configuration ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
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
		updateReq.BudgetId = args[0]

		response, err := a.Budgets.Update(ctx, updateReq)
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

// end service Budgets
