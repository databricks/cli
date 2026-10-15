// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package consumer_listings

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
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
		Use:   "consumer-listings",
		Short: `*Public Preview* Listings are the core entities in the Marketplace.`,
		Long: `This command is in Public Preview and may change without notice.

Listings are the core entities in the Marketplace. They represent the products
  that are available for consumption.`,
		GroupID: "marketplace",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
	*marketplaces.BatchGetListingsRequest,
)

func newBatchGet() *cobra.Command {
	cmd := &cobra.Command{}
	var batchGetReq marketplaces.BatchGetListingsRequest

	// TODO: array: ids

	cmd.Use = "batch-get"
	cmd.Short = `*Public Preview* Get one batch of listings. One may specify up to 50 IDs per request.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get one batch of listings. One may specify up to 50 IDs per request.

  Batch get a published listing in the Databricks Marketplace that the consumer
  has access to.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := marketplaces.BatchGetListingsRequestToSDK(&batchGetReq)
		if err != nil {
			return err
		}
		response, err := w.Marketplaces.BatchGetListings(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.BatchGetListingsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkmarketplaces.GetPublishedListingForConsumerRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkmarketplaces.GetPublishedListingForConsumerRequest

	cmd.Use = "get ID"
	cmd.Short = `*Public Preview* Get listing.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get listing.

  Get a published listing in the Databricks Marketplace that the consumer has
  access to.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No ID argument specified. Loading names for Consumer Listings drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Marketplaces.ListPublishedListingsForConsumerIter(ctx, sdkmarketplaces.ListPublishedListingsForConsumerRequest{}), []string{"Summary", "Name"}, []string{"Id"})
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
			return errors.New("expected to have ")
		}
		getReq.Id = new(args[0])

		sdkReq := &getReq
		response, err := w.Marketplaces.GetPublishedListingForConsumer(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.GetPublishedListingForConsumerResponseFromSDK(response)
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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*marketplaces.ListPublishedListingsForConsumerRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq marketplaces.ListPublishedListingsForConsumerRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// TODO: array: assets
	// TODO: array: categories
	flags.BoolPtrVar(cmd.Flags(), &listReq.IsFree, "is-free", `Filters each listing based on if it is free.`)
	flags.BoolPtrVar(cmd.Flags(), &listReq.IsPrivateExchange, "is-private-exchange", `Filters each listing based on if it is a private exchange.`)
	flags.BoolPtrVar(cmd.Flags(), &listReq.IsStaffPick, "is-staff-pick", `Filters each listing based on whether it is a staff pick.`)
	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", ``)
	// TODO: array: provider_ids
	// TODO: complex arg: tags

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `*Public Preview* List listings.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List listings.

  List all published listings in the Databricks Marketplace that the consumer
  has access to.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := marketplaces.ListPublishedListingsForConsumerRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Marketplaces.ListPublishedListingsForConsumerIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, marketplaces.ListingFromSDK))
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
	*marketplaces.SearchPublishedListingsForConsumerRequest,
)

func newSearch() *cobra.Command {
	cmd := &cobra.Command{}
	var searchReq marketplaces.SearchPublishedListingsForConsumerRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var searchLimit int

	// TODO: array: assets
	// TODO: array: categories
	flags.BoolPtrVar(cmd.Flags(), &searchReq.IsFree, "is-free", ``)
	flags.BoolPtrVar(cmd.Flags(), &searchReq.IsPrivateExchange, "is-private-exchange", ``)
	flags.IntPtrVar(cmd.Flags(), &searchReq.PageSize, "page-size", ``)
	// TODO: array: provider_ids

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&searchLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &searchReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "search QUERY"
	cmd.Short = `*Public Preview* Search listings.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Search listings.

  Search published listings in the Databricks Marketplace that the consumer has
  access to. This query supports a variety of different search parameters and
  performs fuzzy matching.

  Arguments:
    QUERY: Fuzzy matches query`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No QUERY argument specified. Loading names for Consumer Listings drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Marketplaces.ListPublishedListingsForConsumerIter(ctx, sdkmarketplaces.ListPublishedListingsForConsumerRequest{}), []string{"Summary", "Name"}, []string{"Id"})
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
			return errors.New("expected to have fuzzy matches query")
		}
		searchReq.Query = new(args[0])

		sdkReq, err := marketplaces.SearchPublishedListingsForConsumerRequestToSDK(&searchReq)
		if err != nil {
			return err
		}
		if searchLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", searchLimit)
		}
		if searchLimit > 0 {
			ctx = cmdio.WithLimit(ctx, searchLimit)
		}
		response := w.Marketplaces.SearchPublishedListingsForConsumerIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, marketplaces.ListingFromSDK))
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
