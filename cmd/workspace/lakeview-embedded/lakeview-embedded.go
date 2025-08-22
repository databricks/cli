// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package lakeview_embedded

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/spf13/cobra"
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
		Annotations: map[string]string{
			"package": "dashboards",
		},
		RunE: root.ReportUnknownSubcommand,
	}

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
	*dashboards.GetPublishedDashboardTokenInfoRequest,
)

func newGetPublishedDashboardTokenInfo() *cobra.Command {
	cmd := &cobra.Command{}

	var getPublishedDashboardTokenInfoReq dashboards.GetPublishedDashboardTokenInfoRequest

	cmd.Flags().StringVar(&getPublishedDashboardTokenInfoReq.ExternalValue, "external-value", getPublishedDashboardTokenInfoReq.ExternalValue, `Provided external value to be included in the custom claim.`)
	cmd.Flags().StringVar(&getPublishedDashboardTokenInfoReq.ExternalViewerId, "external-viewer-id", getPublishedDashboardTokenInfoReq.ExternalViewerId, `Provided external viewer id to be included in the custom claim.`)

	cmd.Use = "get-published-dashboard-token-info DASHBOARD_ID"
	cmd.Short = `Read an information of a published dashboard to mint an OAuth token.`
	cmd.Long = `Read an information of a published dashboard to mint an OAuth token.
  
  Get a required authorization details and scopes of a published dashboard to
  mint an OAuth token.

  Arguments:
    DASHBOARD_ID: UUID identifying the published dashboard.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPublishedDashboardTokenInfoReq.DashboardId = args[0]

		response, err := w.LakeviewEmbedded.GetPublishedDashboardTokenInfo(ctx, getPublishedDashboardTokenInfoReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
