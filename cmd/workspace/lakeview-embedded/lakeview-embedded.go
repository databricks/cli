// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package lakeview_embedded

import (
	"fmt"

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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				// Check if the subcommand exists
				for _, subcmd := range cmd.Commands() {
					if subcmd.Name() == args[0] {
						// Let Cobra handle the valid subcommand
						return nil
					}
				}
				// Return error for unknown subcommands
				return &root.InvalidArgsError{
					Message: fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath()),
					Command: cmd,
				}
			}
			return cmd.Help()
		},
	}

	// Add methods
	cmd.AddCommand(newGetPublishedDashboardEmbedded())
	cmd.AddCommand(newGetPublishedDashboardTokenInfo())

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
		w := cmdctx.WorkspaceClient(ctx)

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

	// TODO: short flags

	cmd.Flags().StringVar(&getPublishedDashboardTokenInfoReq.ExternalValue, "external-value", getPublishedDashboardTokenInfoReq.ExternalValue, `Provided external value to be included in the custom claim.`)
	cmd.Flags().StringVar(&getPublishedDashboardTokenInfoReq.ExternalViewerId, "external-viewer-id", getPublishedDashboardTokenInfoReq.ExternalViewerId, `Provided external viewer id to be included in the custom claim.`)

	cmd.Use = "get-published-dashboard-token-info DASHBOARD_ID"
	cmd.Short = `Read an information of a published dashboard to mint an OAuth token.`
	cmd.Long = `Read an information of a published dashboard to mint an OAuth token.
  
  Get a required authorization details and scopes of a published dashboard to
  mint an OAuth token. The authorization_details can be enriched to apply
  additional restriction.
  
  Example: Adding the following authorization_details object to downscope the
  viewer permission to specific table  { type: "unity_catalog_privileges",
  privileges: ["SELECT"], object_type: "TABLE", object_full_path:
  "main.default.testdata" } 

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
