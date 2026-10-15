// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package usage_dashboards

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	usagedashboards "github.com/databricks/cli/libs/sdktypes/usagedashboards/v1"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*usagedashboards.CreateBillingUsageDashboardRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq usagedashboards.CreateBillingUsageDashboardRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.EnumVar(cmd.Flags(), &createReq.DashboardType, "dashboard-type", `Workspace level usage dashboard shows usage data for the specified workspace ID. Supported values: [USAGE_DASHBOARD_TYPE_GLOBAL, USAGE_DASHBOARD_TYPE_WORKSPACE]`, "UsageDashboardType", "USAGE_DASHBOARD_TYPE_GLOBAL", "USAGE_DASHBOARD_TYPE_WORKSPACE")
	flags.EnumVar(cmd.Flags(), &createReq.MajorVersion, "major-version", `The major version of the usage dashboard template to use. Supported values: [USAGE_DASHBOARD_MAJOR_VERSION_1, USAGE_DASHBOARD_MAJOR_VERSION_2]`, "UsageDashboardMajorVersion", "USAGE_DASHBOARD_MAJOR_VERSION_1", "USAGE_DASHBOARD_MAJOR_VERSION_2")
	flags.Int64PtrVar(cmd.Flags(), &createReq.WorkspaceId, "workspace-id", `The workspace ID of the workspace in which the usage dashboard is created.`)

	cmd.Use = "create"
	cmd.Short = `Create new usage dashboard.`
	cmd.Long = `Create new usage dashboard.

  Create a usage dashboard specified by workspaceId, accountId, and dashboard
  type.`

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

		sdkReq, err := usagedashboards.CreateBillingUsageDashboardRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.UsageDashboards.CreateBillingUsageDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := usagedashboards.CreateBillingUsageDashboardResponseFromSDK(response)
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*usagedashboards.GetBillingUsageDashboardRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq usagedashboards.GetBillingUsageDashboardRequest

	flags.EnumVar(cmd.Flags(), &getReq.DashboardType, "dashboard-type", `Workspace level usage dashboard shows usage data for the specified workspace ID. Supported values: [USAGE_DASHBOARD_TYPE_GLOBAL, USAGE_DASHBOARD_TYPE_WORKSPACE]`, "UsageDashboardType", "USAGE_DASHBOARD_TYPE_GLOBAL", "USAGE_DASHBOARD_TYPE_WORKSPACE")
	flags.Int64PtrVar(cmd.Flags(), &getReq.WorkspaceId, "workspace-id", `The workspace ID of the workspace in which the usage dashboard is created.`)

	cmd.Use = "get"
	cmd.Short = `Get usage dashboard.`
	cmd.Long = `Get usage dashboard.

  Get a usage dashboard specified by workspaceId, accountId, and dashboard type.`

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
		_ = a

		sdkReq, err := usagedashboards.GetBillingUsageDashboardRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		response, err := a.UsageDashboards.GetBillingUsageDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := usagedashboards.GetBillingUsageDashboardResponseFromSDK(response)
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

// end service UsageDashboards
