// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package lakeview_embedded

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	lakeview "github.com/databricks/cli/libs/sdktypes/lakeview/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lakeview-embedded",
		Short:   `Token-based Lakeview APIs for embedding dashboards in external applications.`,
		Long:    `Token-based Lakeview APIs for embedding dashboards in external applications.`,
		GroupID: "dashboards",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newGetPublishedDashboardTokenInfo())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-published-dashboard-token-info command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPublishedDashboardTokenInfoOverrides []func(
	*cobra.Command,
	*lakeview.GetPublishedDashboardTokenInfoRequest,
)

func newGetPublishedDashboardTokenInfo() *cobra.Command {
	cmd := &cobra.Command{}
	var getPublishedDashboardTokenInfoReq lakeview.GetPublishedDashboardTokenInfoRequest

	flags.StringPtrVar(cmd.Flags(), &getPublishedDashboardTokenInfoReq.ExternalValue, "external-value", `Provided external value to be included in the custom claim.`)
	flags.StringPtrVar(cmd.Flags(), &getPublishedDashboardTokenInfoReq.ExternalViewerId, "external-viewer-id", `Provided external viewer id to be included in the custom claim.`)

	cmd.Use = "get-published-dashboard-token-info DASHBOARD_ID"
	cmd.Short = `*Public Preview* Read information of a published dashboard to mint an OAuth token.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Read information of a published dashboard to mint an OAuth token.

  Get a required authorization details and scopes of a published dashboard to
  mint an OAuth token.

  The caller must be a workspace user with one of the following entitlements:
  Workspace access, Databricks SQL access, or Consumer access.

  Account-level users who are not members of the workspace cannot call this
  endpoint, even if the dashboard has been shared with them.

  Arguments:
    DASHBOARD_ID: UUID identifying the published dashboard.`

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

		getPublishedDashboardTokenInfoReq.DashboardId = new(args[0])

		sdkReq, err := lakeview.GetPublishedDashboardTokenInfoRequestToSDK(&getPublishedDashboardTokenInfoReq)
		if err != nil {
			return err
		}
		response, err := w.Lakeview.GetPublishedDashboardTokenInfo(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := lakeview.GetPublishedDashboardTokenInfoResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPublishedDashboardTokenInfoOverrides {
		fn(cmd, &getPublishedDashboardTokenInfoReq)
	}

	return cmd
}

// end service LakeviewEmbedded
