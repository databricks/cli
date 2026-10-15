// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package consumer_fulfillments

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	marketplaces "github.com/databricks/cli/libs/sdktypes/marketplaces/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consumer-fulfillments",
		Short: `*Public Preview* Fulfillments are entities that allow consumers to preview installations.`,
		Long: `This command is in Public Preview and may change without notice.

Fulfillments are entities that allow consumers to preview installations.`,
		GroupID: "marketplace",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
	*marketplaces.GetListingContentMetadataRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq marketplaces.GetListingContentMetadataRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var getLimit int

	flags.IntPtrVar(cmd.Flags(), &getReq.PageSize, "page-size", ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&getLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &getReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "get LISTING_ID"
	cmd.Short = `*Public Preview* Get listing content metadata.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get listing content metadata.

  Get a high level preview of the metadata of listing installable content.`

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

		getReq.ListingId = new(args[0])

		sdkReq, err := marketplaces.GetListingContentMetadataRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		if getLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", getLimit)
		}
		if getLimit > 0 {
			ctx = cmdio.WithLimit(ctx, getLimit)
		}
		response := w.Marketplaces.GetListingContentIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, marketplaces.SharedDataObjectFromSDK))
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
	*marketplaces.ListListingFulfillmentsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq marketplaces.ListListingFulfillmentsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list LISTING_ID"
	cmd.Short = `*Public Preview* List all listing fulfillments.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List all listing fulfillments.

  Get all listings fulfillments associated with a listing. A _fulfillment_ is a
  potential installation. Standard installations contain metadata about the
  attached share or git repo. Only one of these fields will be present.
  Personalized installations contain metadata about the attached share or git
  repo, as well as the Delta Sharing recipient type.`

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

		listReq.ListingId = new(args[0])

		sdkReq, err := marketplaces.ListListingFulfillmentsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Marketplaces.ListListingFulfillmentsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, marketplaces.ListingFulfillmentFromSDK))
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
