// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package budgets

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	budgets "github.com/databricks/cli/libs/sdktypes/budgets/v1"
	sdkbudgets "github.com/databricks/sdk-go/budgets/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budgets",
		Short: `*Public Preview* These APIs manage budget configurations for this account.`,
		Long: `This command is in Public Preview and may change without notice.

These APIs manage budget configurations for this account. Budgets enable you
  to monitor usage across your account. You can set up budgets to either track
  account-wide spending, or apply filters to track the spending of specific
  teams, projects, or workspaces.`,
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
	*budgets.CreateBudgetConfigurationRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq budgets.CreateBudgetConfigurationRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `*Public Preview* Create new budget.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create new budget.

  Create a new budget configuration for an account. For full details, see
  https://docs.databricks.com/en/admin/account-settings/budgets.html.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		sdkReq, err := budgets.CreateBudgetConfigurationRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.Budgets.CreateBudgetConfiguration(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := budgets.CreateBudgetConfigurationResponseFromSDK(response)
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
	*sdkbudgets.DeleteBudgetConfigurationRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkbudgets.DeleteBudgetConfigurationRequest

	cmd.Use = "delete BUDGET_ID"
	cmd.Short = `*Public Preview* Delete budget.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete budget.

  Deletes a budget configuration for an account. Both account and budget
  configuration are specified by ID. This cannot be undone.

  Arguments:
    BUDGET_ID: The Databricks budget configuration ID.`

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

		deleteReq.BudgetId = new(args[0])

		sdkReq := &deleteReq
		response, err := a.Budgets.DeleteBudgetConfiguration(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
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
	*sdkbudgets.GetBudgetConfigurationRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkbudgets.GetBudgetConfigurationRequest

	cmd.Use = "get BUDGET_ID"
	cmd.Short = `*Public Preview* Get budget.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get budget.

  Gets a budget configuration for an account. Both account and budget
  configuration are specified by ID.

  Arguments:
    BUDGET_ID: The budget configuration ID`

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

		getReq.BudgetId = new(args[0])

		sdkReq := &getReq
		response, err := a.Budgets.GetBudgetConfiguration(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := budgets.GetBudgetConfigurationResponseFromSDK(response)
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
	*budgets.ListBudgetConfigurationsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq budgets.ListBudgetConfigurationsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `*Public Preview* Get all budgets.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get all budgets.

  Gets all budgets associated with this account.`

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

		sdkReq, err := budgets.ListBudgetConfigurationsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := a.Budgets.ListBudgetConfigurationsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, budgets.BudgetConfigurationFromSDK))
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
	*budgets.UpdateBudgetConfigurationRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq budgets.UpdateBudgetConfigurationRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update BUDGET_ID"
	cmd.Short = `*Public Preview* Modify budget.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Modify budget.

  Updates a budget configuration for an account. Both account and budget
  configuration are specified by ID.

  Arguments:
    BUDGET_ID: The Databricks budget configuration ID.`

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
		updateReq.BudgetId = new(args[0])

		sdkReq, err := budgets.UpdateBudgetConfigurationRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := a.Budgets.UpdateBudgetConfiguration(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := budgets.UpdateBudgetConfigurationResponseFromSDK(response)
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

// end service Budgets
