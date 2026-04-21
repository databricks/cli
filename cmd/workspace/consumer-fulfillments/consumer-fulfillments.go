// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package consumer_fulfillments

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/marketplace"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "consumer-fulfillments",
		Short:   `Fulfillments are entities that allow consumers to preview installations.`,
		Long:    `Fulfillments are entities that allow consumers to preview installations.`,
		GroupID: "marketplace",
		RunE:    root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*marketplace.GetListingContentMetadataRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq marketplace.GetListingContentMetadataRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var getLimit int

	cmd.Flags().IntVar(&getReq.PageSize, "page-size", getReq.PageSize, ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&getLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&getReq.PageToken, "page-token", getReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "get LISTING_ID"
	cmd.Short = `Get listing content metadata.`
	cmd.Long = `Get listing content metadata.

  Get a high level preview of the metadata of listing installable content.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.ListingId = args[0]

		response := w.ConsumerFulfillments.Get(ctx, getReq)
		if getLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", getLimit)
		}
		if getLimit > 0 {
			ctx = cmdio.WithLimit(ctx, getLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*marketplace.ListFulfillmentsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq marketplace.ListFulfillmentsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list LISTING_ID"
	cmd.Short = `List all listing fulfillments.`
	cmd.Long = `List all listing fulfillments.

  Get all listings fulfillments associated with a listing. A _fulfillment_ is a
  potential installation. Standard installations contain metadata about the
  attached share or git repo. Only one of these fields will be present.
  Personalized installations contain metadata about the attached share or git
  repo, as well as the Delta Sharing recipient type.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listReq.ListingId = args[0]

		response := w.ConsumerFulfillments.List(ctx, listReq)
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

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

// end service ConsumerFulfillments
