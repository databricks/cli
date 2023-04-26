// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package budgets

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/bricks/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/billing"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "budgets",
	Short: `These APIs manage budget configuration including notifications for exceeding a budget for a period.`,
	Long: `These APIs manage budget configuration including notifications for exceeding a
  budget for a period. They can also retrieve the status of each budget.`,
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
		err = createJson.Unmarshal(&createReq)
		if err != nil {
			return err
		}
		_, err = fmt.Sscan(args[0], &createReq.Budget)
		if err != nil {
			return fmt.Errorf("invalid BUDGET: %s", args[0])
		}
		createReq.BudgetId = args[1]

		response, err := a.Budgets.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq billing.DeleteBudgetRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

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
		if len(args) == 0 {
			names, err := a.Budgets.BudgetWithStatusNameToBudgetIdMap(ctx)
			if err != nil {
				return err
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

		err = a.Budgets.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq billing.GetBudgetRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

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
		if len(args) == 0 {
			names, err := a.Budgets.BudgetWithStatusNameToBudgetIdMap(ctx)
			if err != nil {
				return err
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

		response, err := a.Budgets.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
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
		err = updateJson.Unmarshal(&updateReq)
		if err != nil {
			return err
		}
		_, err = fmt.Sscan(args[0], &updateReq.Budget)
		if err != nil {
			return fmt.Errorf("invalid BUDGET: %s", args[0])
		}
		updateReq.BudgetId = args[1]

		err = a.Budgets.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Budgets
