// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package budgets

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/billing"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "budgets",
	Short: `These APIs manage budget configuration including notifications for exceeding a budget for a period.`,
	Long: `These APIs manage budget configuration including notifications for exceeding a
  budget for a period. They can also retrieve the status of each budget.`,
	Annotations: map[string]string{
		"package": "billing",
	},

	// This service is being previewed; hide from help output.
	Hidden: true,
}

// start create command

var createReq billing.WrappedBudget
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new budget.`,
	Long: `Create a new budget.
  
  Creates a new budget in the specified account.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &createReq.Budget)
			if err != nil {
				return fmt.Errorf("invalid BUDGET: %s", args[0])
			}
		}

		response, err := a.Budgets.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start delete command

var deleteReq billing.DeleteBudgetRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete BUDGET_ID",
	Short: `Delete budget.`,
	Long: `Delete budget.
  
  Deletes the budget specified by its UUID.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No BUDGET_ID argument specified. Loading names for Budgets drop-down."
				names, err := a.Budgets.BudgetWithStatusNameToBudgetIdMap(ctx)
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Budgets drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "Budget ID")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have budget id")
			}
			deleteReq.BudgetId = args[0]
		}

		err = a.Budgets.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start get command

var getReq billing.GetBudgetRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get BUDGET_ID",
	Short: `Get budget and its status.`,
	Long: `Get budget and its status.
  
  Gets the budget specified by its UUID, including noncumulative status for each
  day that the budget is configured to include.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No BUDGET_ID argument specified. Loading names for Budgets drop-down."
				names, err := a.Budgets.BudgetWithStatusNameToBudgetIdMap(ctx)
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Budgets drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "Budget ID")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have budget id")
			}
			getReq.BudgetId = args[0]
		}

		response, err := a.Budgets.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get all budgets.`,
	Long: `Get all budgets.
  
  Gets all budgets associated with this account, including noncumulative status
  for each day that the budget is configured to include.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		response, err := a.Budgets.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start update command

var updateReq billing.WrappedBudget
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Modify budget.`,
	Long: `Modify budget.
  
  Modifies a budget in this account. Budget properties are completely
  overwritten.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &updateReq.Budget)
			if err != nil {
				return fmt.Errorf("invalid BUDGET: %s", args[0])
			}
			updateReq.BudgetId = args[1]
		}

		err = a.Budgets.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service Budgets
