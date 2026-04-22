// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package consumer_listings

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
		Use:   "consumer-listings",
		Short: `Listings are the core entities in the Marketplace.`,
		Long: `Listings are the core entities in the Marketplace. They represent the products
  that are available for consumption.`,
		GroupID: "marketplace",
		RunE:    root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newBatchGet())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newSearch())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start batch-get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var batchGetOverrides []func(
	*cobra.Command,
	*marketplace.BatchGetListingsRequest,
)

func newBatchGet() *cobra.Command {
	cmd := &cobra.Command{}

	var batchGetReq marketplace.BatchGetListingsRequest

	// TODO: array: ids

	cmd.Use = "batch-get"
	cmd.Short = `Get one batch of listings. One may specify up to 50 IDs per request.`
	cmd.Long = `Get one batch of listings. One may specify up to 50 IDs per request.

  Batch get a published listing in the Databricks Marketplace that the consumer
  has access to.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response, err := w.ConsumerListings.BatchGet(ctx, batchGetReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range batchGetOverrides {
		fn(cmd, &batchGetReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*marketplace.GetListingRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq marketplace.GetListingRequest

	cmd.Use = "get ID"
	cmd.Short = `Get listing.`
	cmd.Long = `Get listing.

  Get a published listing in the Databricks Marketplace that the consumer has
  access to.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No ID argument specified. Loading names for Consumer Listings drop-down.")
			names, err := w.ConsumerListings.ListingSummaryNameToIdMap(ctx, marketplace.ListListingsRequest{})
			sp.Close()
			if err != nil {
				return fmt.Errorf("failed to load names for Consumer Listings drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		getReq.Id = args[0]

		response, err := w.ConsumerListings.Get(ctx, getReq)
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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*marketplace.ListListingsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq marketplace.ListListingsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// TODO: array: assets
	// TODO: array: categories
	cmd.Flags().BoolVar(&listReq.IsFree, "is-free", listReq.IsFree, `Filters each listing based on if it is free.`)
	cmd.Flags().BoolVar(&listReq.IsPrivateExchange, "is-private-exchange", listReq.IsPrivateExchange, `Filters each listing based on if it is a private exchange.`)
	cmd.Flags().BoolVar(&listReq.IsStaffPick, "is-staff-pick", listReq.IsStaffPick, `Filters each listing based on whether it is a staff pick.`)
	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, ``)
	// TODO: array: provider_ids
	// TODO: array: tags

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List listings.`
	cmd.Long = `List listings.

  List all published listings in the Databricks Marketplace that the consumer
  has access to.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ConsumerListings.List(ctx, listReq)
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

// start search command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var searchOverrides []func(
	*cobra.Command,
	*marketplace.SearchListingsRequest,
)

func newSearch() *cobra.Command {
	cmd := &cobra.Command{}

	var searchReq marketplace.SearchListingsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var searchLimit int

	// TODO: array: assets
	// TODO: array: categories
	cmd.Flags().BoolVar(&searchReq.IsFree, "is-free", searchReq.IsFree, ``)
	cmd.Flags().BoolVar(&searchReq.IsPrivateExchange, "is-private-exchange", searchReq.IsPrivateExchange, ``)
	cmd.Flags().IntVar(&searchReq.PageSize, "page-size", searchReq.PageSize, ``)
	// TODO: array: provider_ids

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&searchLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&searchReq.PageToken, "page-token", searchReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "search QUERY"
	cmd.Short = `Search listings.`
	cmd.Long = `Search listings.

  Search published listings in the Databricks Marketplace that the consumer has
  access to. This query supports a variety of different search parameters and
  performs fuzzy matching.

  Arguments:
    QUERY: Fuzzy matches query`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No QUERY argument specified. Loading names for Consumer Listings drop-down.")
			names, err := w.ConsumerListings.ListingSummaryNameToIdMap(ctx, marketplace.ListListingsRequest{})
			sp.Close()
			if err != nil {
				return fmt.Errorf("failed to load names for Consumer Listings drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Fuzzy matches query")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have fuzzy matches query")
		}
		searchReq.Query = args[0]

		response := w.ConsumerListings.Search(ctx, searchReq)
		if searchLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", searchLimit)
		}
		if searchLimit > 0 {
			ctx = cmdio.WithLimit(ctx, searchLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range searchOverrides {
		fn(cmd, &searchReq)
	}

	return cmd
}

// end service ConsumerListings
