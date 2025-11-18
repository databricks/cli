// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package o_auth_published_apps

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/oauth2"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "o-auth-published-apps",
		Short: `These APIs enable administrators to view all the available published OAuth applications in Databricks.`,
		Long: `These APIs enable administrators to view all the available published OAuth
  applications in Databricks. Administrators can add the published OAuth
  applications to their account through the OAuth Published App Integration
  APIs.`,
		GroupID: "oauth2",
		Annotations: map[string]string{
			"package": "oauth2",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newList())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*oauth2.ListOAuthPublishedAppsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq oauth2.ListOAuthPublishedAppsRequest

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `The max number of OAuth published apps to return in one page.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `A token that can be used to get the next page of results.`)

	cmd.Use = "list"
	cmd.Short = `Get all the published OAuth apps.`
	cmd.Long = `Get all the published OAuth apps.
  
  Get all the available published OAuth apps in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		response := a.OAuthPublishedApps.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
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

// end service OAuthPublishedApps
