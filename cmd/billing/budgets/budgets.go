package budgets

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/billing"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "budgets",
	Short: `These APIs manage budget configuration including notifications for exceeding a budget for a period.`,
}

var createReq billing.WrappedBudget

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	// TODO: complex arg: budget
	createCmd.Flags().StringVar(&createReq.BudgetId, "budget-id", "", `Budget ID.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new budget.`,
	Long: `Create a new budget.
  
  Creates a new budget in the specified account.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Budgets.Create(ctx, createReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var deleteReq billing.DeleteBudgetRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.BudgetId, "budget-id", "", `Budget ID.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete budget.`,
	Long: `Delete budget.
  
  Deletes the budget specified by its UUID.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err := a.Budgets.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq billing.GetBudgetRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.BudgetId, "budget-id", "", `Budget ID.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get budget and its status.`,
	Long: `Get budget and its status.
  
  Gets the budget specified by its UUID, including noncumulative status for each
  day that the budget is configured to include.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Budgets.Get(ctx, getReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get all budgets.`,
	Long: `Get all budgets.
  
  Gets all budgets associated with this account, including noncumulative status
  for each day that the budget is configured to include.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Budgets.ListAll(ctx)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var updateReq billing.WrappedBudget

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	// TODO: complex arg: budget
	updateCmd.Flags().StringVar(&updateReq.BudgetId, "budget-id", "", `Budget ID.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Modify budget.`,
	Long: `Modify budget.
  
  Modifies a budget in this account. Budget properties are completely
  overwritten.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err := a.Budgets.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
