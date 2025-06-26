// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package usage_dashboards

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
		Use:   "usage-dashboards",
		Short: `These APIs manage usage dashboards for this account.`,
		Long: `These APIs manage usage dashboards for this account. Usage dashboards enable
  you to gain insights into your usage with pre-built dashboards: visualize
  breakdowns, analyze tag attributions, and identify cost drivers.`,
		GroupID: "billing",
		Annotations: map[string]string{
			"package": "billing",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newGet())

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
	*billing.CreateBillingUsageDashboardRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq billing.CreateBillingUsageDashboardRequest
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&createReq.DashboardType, "dashboard-type", `Workspace level usage dashboard shows usage data for the specified workspace ID. Supported values: [USAGE_DASHBOARD_TYPE_GLOBAL, USAGE_DASHBOARD_TYPE_WORKSPACE]`)
	cmd.Flags().Int64Var(&createReq.WorkspaceId, "workspace-id", createReq.WorkspaceId, `The workspace ID of the workspace in which the usage dashboard is created.`)

	cmd.Use = "create"
	cmd.Short = `Create new usage dashboard.`
	cmd.Long = `Create new usage dashboard.
  
  Create a usage dashboard specified by workspaceId, accountId, and dashboard
  type.`

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

		response, err := a.UsageDashboards.Create(ctx, createReq)
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*billing.GetBillingUsageDashboardRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq billing.GetBillingUsageDashboardRequest

	cmd.Flags().Var(&getReq.DashboardType, "dashboard-type", `Workspace level usage dashboard shows usage data for the specified workspace ID. Supported values: [USAGE_DASHBOARD_TYPE_GLOBAL, USAGE_DASHBOARD_TYPE_WORKSPACE]`)
	cmd.Flags().Int64Var(&getReq.WorkspaceId, "workspace-id", getReq.WorkspaceId, `The workspace ID of the workspace in which the usage dashboard is created.`)

	cmd.Use = "get"
	cmd.Short = `Get usage dashboard.`
	cmd.Long = `Get usage dashboard.
  
  Get a usage dashboard specified by workspaceId, accountId, and dashboard type.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		response, err := a.UsageDashboards.Get(ctx, getReq)
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

// end service UsageDashboards
