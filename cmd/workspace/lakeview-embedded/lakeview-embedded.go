// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package lakeview_embedded

import (
	"github.com/databricks/cli/cmd/root"
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
	}

	// Add methods
	cmd.AddCommand(newGetPublishedDashboardEmbedded())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-published-dashboard-embedded command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPublishedDashboardEmbeddedOverrides []func(
	*cobra.Command,
	*dashboards.GetPublishedDashboardEmbeddedRequest,
)

func newGetPublishedDashboardEmbedded() *cobra.Command {
	cmd := &cobra.Command{}

	var getPublishedDashboardEmbeddedReq dashboards.GetPublishedDashboardEmbeddedRequest

	// TODO: short flags

	cmd.Use = "get-published-dashboard-embedded DASHBOARD_ID"
	cmd.Short = `Read a published dashboard in an embedded ui.`
	cmd.Long = `Read a published dashboard in an embedded ui.
  
  Get the current published dashboard within an embedded context.

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
		w := root.WorkspaceClient(ctx)

		getPublishedDashboardEmbeddedReq.DashboardId = args[0]

		err = w.LakeviewEmbedded.GetPublishedDashboardEmbedded(ctx, getPublishedDashboardEmbeddedReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPublishedDashboardEmbeddedOverrides {
		fn(cmd, &getPublishedDashboardEmbeddedReq)
	}

	return cmd
}

// end service LakeviewEmbedded
