// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package provider_exchanges

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
		Use:   "provider-exchanges",
		Short: `*Public Preview* Marketplace exchanges allow providers to share their listings with a curated set of customers.`,
		Long: `This command is in Public Preview and may change without notice.

Marketplace exchanges allow providers to share their listings with a curated
  set of customers.`,
		GroupID: "marketplace",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	// Add methods
	cmd.AddCommand(newAddListingToExchange())
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newDeleteListingFromExchange())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListExchangesForListing())
	cmd.AddCommand(newListListingsForExchange())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start add-listing-to-exchange command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var addListingToExchangeOverrides []func(
	*cobra.Command,
	*marketplaces.AddExchangeForListingRequest,
)

func newAddListingToExchange() *cobra.Command {
	cmd := &cobra.Command{}
	var addListingToExchangeReq marketplaces.AddExchangeForListingRequest

	var addListingToExchangeJson flags.JsonFlag

	cmd.Flags().Var(&addListingToExchangeJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "add-listing-to-exchange LISTING_ID EXCHANGE_ID"
	cmd.Short = `*Public Preview* Add an exchange for listing.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Add an exchange for listing.

  Associate an exchange with a listing`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'listing_id', 'exchange_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := addListingToExchangeJson.Unmarshal(&addListingToExchangeReq)
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
		if !cmd.Flags().Changed("json") {
			addListingToExchangeReq.ListingId = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			addListingToExchangeReq.ExchangeId = new(args[1])
		}

		sdkReq, err := marketplaces.AddExchangeForListingRequestToSDK(&addListingToExchangeReq)
		if err != nil {
			return err
		}
		response, err := w.Marketplaces.AddExchangeForListing(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.AddExchangeForListingResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range addListingToExchangeOverrides {
		fn(cmd, &addListingToExchangeReq)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*marketplaces.CreateExchangeRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq marketplaces.CreateExchangeRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `*Public Preview* Create an exchange.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create an exchange.

  Create an exchange`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		sdkReq, err := marketplaces.CreateExchangeRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Marketplaces.CreateExchange(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.CreateExchangeResponseFromSDK(response)
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
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*sdkmarketplaces.DeleteExchangeRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkmarketplaces.DeleteExchangeRequest

	cmd.Use = "delete ID"
	cmd.Short = `*Public Preview* Delete an exchange.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete an exchange.

  This removes a listing from marketplace.`

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

		deleteReq.Id = new(args[0])

		sdkReq := &deleteReq
		response, err := w.Marketplaces.DeleteExchange(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start delete-listing-from-exchange command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteListingFromExchangeOverrides []func(
	*cobra.Command,
	*sdkmarketplaces.RemoveExchangeForListingRequest,
)

func newDeleteListingFromExchange() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteListingFromExchangeReq sdkmarketplaces.RemoveExchangeForListingRequest

	cmd.Use = "delete-listing-from-exchange ID"
	cmd.Short = `*Public Preview* Remove an exchange for listing.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Remove an exchange for listing.

  Disassociate an exchange with a listing`

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

		deleteListingFromExchangeReq.Id = new(args[0])

		sdkReq := &deleteListingFromExchangeReq
		response, err := w.Marketplaces.RemoveExchangeForListing(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteListingFromExchangeOverrides {
		fn(cmd, &deleteListingFromExchangeReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*sdkmarketplaces.GetExchangeRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkmarketplaces.GetExchangeRequest

	cmd.Use = "get ID"
	cmd.Short = `*Public Preview* Get an exchange.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get an exchange.`

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

		getReq.Id = new(args[0])

		sdkReq := &getReq
		response, err := w.Marketplaces.GetExchange(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.GetExchangeResponseFromSDK(response)
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
	*marketplaces.ListExchangesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq marketplaces.ListExchangesRequest

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

	cmd.Use = "list"
	cmd.Short = `*Public Preview* List exchanges.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List exchanges.

  List exchanges visible to provider`

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

		sdkReq, err := marketplaces.ListExchangesRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Marketplaces.ListExchangesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, marketplaces.ExchangeFromSDK))
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

// start list-exchanges-for-listing command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listExchangesForListingOverrides []func(
	*cobra.Command,
	*marketplaces.ListExchangesForListingRequest,
)

func newListExchangesForListing() *cobra.Command {
	cmd := &cobra.Command{}
	var listExchangesForListingReq marketplaces.ListExchangesForListingRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listExchangesForListingLimit int

	flags.IntPtrVar(cmd.Flags(), &listExchangesForListingReq.PageSize, "page-size", ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listExchangesForListingLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listExchangesForListingReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-exchanges-for-listing LISTING_ID"
	cmd.Short = `*Public Preview* List exchanges for listing.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List exchanges for listing.

  List exchanges associated with a listing`

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

		listExchangesForListingReq.ListingId = new(args[0])

		sdkReq, err := marketplaces.ListExchangesForListingRequestToSDK(&listExchangesForListingReq)
		if err != nil {
			return err
		}
		if listExchangesForListingLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listExchangesForListingLimit)
		}
		if listExchangesForListingLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listExchangesForListingLimit)
		}
		response := w.Marketplaces.ListExchangesForListingIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, marketplaces.ExchangeListingFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listExchangesForListingOverrides {
		fn(cmd, &listExchangesForListingReq)
	}

	return cmd
}

// start list-listings-for-exchange command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listListingsForExchangeOverrides []func(
	*cobra.Command,
	*marketplaces.ListListingsForExchangeRequest,
)

func newListListingsForExchange() *cobra.Command {
	cmd := &cobra.Command{}
	var listListingsForExchangeReq marketplaces.ListListingsForExchangeRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listListingsForExchangeLimit int

	flags.IntPtrVar(cmd.Flags(), &listListingsForExchangeReq.PageSize, "page-size", ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listListingsForExchangeLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listListingsForExchangeReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-listings-for-exchange EXCHANGE_ID"
	cmd.Short = `*Public Preview* List listings for exchange.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List listings for exchange.

  List listings associated with an exchange`

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

		listListingsForExchangeReq.ExchangeId = new(args[0])

		sdkReq, err := marketplaces.ListListingsForExchangeRequestToSDK(&listListingsForExchangeReq)
		if err != nil {
			return err
		}
		if listListingsForExchangeLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listListingsForExchangeLimit)
		}
		if listListingsForExchangeLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listListingsForExchangeLimit)
		}
		response := w.Marketplaces.ListListingsForExchangeIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, marketplaces.ExchangeListingFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listListingsForExchangeOverrides {
		fn(cmd, &listListingsForExchangeReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*marketplaces.UpdateExchangeRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq marketplaces.UpdateExchangeRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update ID"
	cmd.Short = `*Public Preview* Update exchange.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update exchange.

  Update an exchange`

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
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}
		updateReq.Id = new(args[0])

		sdkReq, err := marketplaces.UpdateExchangeRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Marketplaces.UpdateExchange(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.UpdateExchangeResponseFromSDK(response)
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

// end service ProviderExchanges
