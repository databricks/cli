// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package provider_provider_analytics_dashboards

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	marketplaces "github.com/databricks/cli/libs/sdktypes/marketplaces/v1"
	sdkmarketplaces "github.com/databricks/sdk-go/marketplaces/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider-provider-analytics-dashboards",
		Short: `*Public Preview* Manage templated analytics solution for providers.`,
		Long: `This command is in Public Preview and may change without notice.

Manage templated analytics solution for providers.`,
		GroupID: "marketplace",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetLatestVersion())
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
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "create"
	cmd.Short = `*Public Preview* Create provider analytics dashboard.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create provider analytics dashboard.

  Create provider analytics dashboard. Returns Marketplace specific id. Not to
  be confused with the Lakeview dashboard id.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq := &sdkmarketplaces.CreateProviderAnalyticsDashboardRequest{}

		response, err := w.Marketplaces.CreateProviderAnalyticsDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.CreateProviderAnalyticsDashboardResponseFromSDK(response)
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
		fn(cmd)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "get"
	cmd.Short = `*Public Preview* Get provider analytics dashboard.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get provider analytics dashboard.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq := &sdkmarketplaces.ListProviderAnalyticsDashboardRequest{}

		response, err := w.Marketplaces.ListProviderAnalyticsDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.ListProviderAnalyticsDashboardResponseFromSDK(response)
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
		fn(cmd)
	}

	return cmd
}

// start get-latest-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getLatestVersionOverrides []func(
	*cobra.Command,
)

func newGetLatestVersion() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "get-latest-version"
	cmd.Short = `*Public Preview* Get latest version of provider analytics dashboard.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get latest version of provider analytics dashboard.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq := &sdkmarketplaces.GetLatestVersionProviderAnalyticsDashboardRequest{}

		response, err := w.Marketplaces.GetLatestVersionProviderAnalyticsDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.GetLatestVersionProviderAnalyticsDashboardResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getLatestVersionOverrides {
		fn(cmd)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*marketplaces.UpdateProviderAnalyticsDashboardRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq marketplaces.UpdateProviderAnalyticsDashboardRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.Int64PtrVar(cmd.Flags(), &updateReq.Version, "version", `this is the version of the dashboard template we want to update our user to current expectation is that it should be equal to latest version of the dashboard template.`)

	cmd.Use = "update ID"
	cmd.Short = `*Public Preview* Update provider analytics dashboard.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update provider analytics dashboard.

  Arguments:
    ID: id is immutable property and can't be updated.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

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
		}
		updateReq.Id = new(args[0])

		sdkReq, err := marketplaces.UpdateProviderAnalyticsDashboardRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Marketplaces.UpdateProviderAnalyticsDashboard(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.UpdateProviderAnalyticsDashboardResponseFromSDK(response)
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

// end service ProviderProviderAnalyticsDashboards
