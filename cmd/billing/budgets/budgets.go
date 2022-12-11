// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package budgets

import (
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
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
var createJson jsonflag.JsonFlag

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
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Budgets.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
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
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		deleteReq.BudgetId = args[0]
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
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
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		getReq.BudgetId = args[0]
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Budgets.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
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
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Budgets.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq billing.WrappedBudget
var updateJson jsonflag.JsonFlag

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
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err = a.Budgets.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Budgets
