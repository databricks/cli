// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package provider_exchanges

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/marketplace"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider-exchanges",
		Short: `Marketplace exchanges allow providers to share their listings with a curated set of customers.`,
		Long: `Marketplace exchanges allow providers to share their listings with a curated
  set of customers.`,
		GroupID: "marketplace",
		Annotations: map[string]string{
			"package": "marketplace",
		},
		RunE: root.ReportUnknownSubcommand,
	}

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
	*marketplace.AddExchangeForListingRequest,
)

func newAddListingToExchange() *cobra.Command {
	cmd := &cobra.Command{}

	var addListingToExchangeReq marketplace.AddExchangeForListingRequest
	var addListingToExchangeJson flags.JsonFlag

	cmd.Flags().Var(&addListingToExchangeJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "add-listing-to-exchange LISTING_ID EXCHANGE_ID"
	cmd.Short = `Add an exchange for listing.`
	cmd.Long = `Add an exchange for listing.
  
  Associate an exchange with a listing`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'listing_id', 'exchange_id' in your JSON input")
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

		if cmd.Flags().Changed("json") {
			diags := addListingToExchangeJson.Unmarshal(&addListingToExchangeReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			addListingToExchangeReq.ListingId = args[0]
		}
		if !cmd.Flags().Changed("json") {
			addListingToExchangeReq.ExchangeId = args[1]
		}

		response, err := w.ProviderExchanges.AddListingToExchange(ctx, addListingToExchangeReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*marketplace.CreateExchangeRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq marketplace.CreateExchangeRequest
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create an exchange.`
	cmd.Long = `Create an exchange.
  
  Create an exchange`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := w.ProviderExchanges.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*marketplace.DeleteExchangeRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq marketplace.DeleteExchangeRequest

	cmd.Use = "delete ID"
	cmd.Short = `Delete an exchange.`
	cmd.Long = `Delete an exchange.
  
  This removes a listing from marketplace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.Id = args[0]

		err = w.ProviderExchanges.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
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
	*marketplace.RemoveExchangeForListingRequest,
)

func newDeleteListingFromExchange() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteListingFromExchangeReq marketplace.RemoveExchangeForListingRequest

	cmd.Use = "delete-listing-from-exchange ID"
	cmd.Short = `Remove an exchange for listing.`
	cmd.Long = `Remove an exchange for listing.
  
  Disassociate an exchange with a listing`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteListingFromExchangeReq.Id = args[0]

		err = w.ProviderExchanges.DeleteListingFromExchange(ctx, deleteListingFromExchangeReq)
		if err != nil {
			return err
		}
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
	*marketplace.GetExchangeRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq marketplace.GetExchangeRequest

	cmd.Use = "get ID"
	cmd.Short = `Get an exchange.`
	cmd.Long = `Get an exchange.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.Id = args[0]

		response, err := w.ProviderExchanges.Get(ctx, getReq)
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
	*marketplace.ListExchangesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq marketplace.ListExchangesRequest

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, ``)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, ``)

	cmd.Use = "list"
	cmd.Short = `List exchanges.`
	cmd.Long = `List exchanges.
  
  List exchanges visible to provider`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ProviderExchanges.List(ctx, listReq)
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

// start list-exchanges-for-listing command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listExchangesForListingOverrides []func(
	*cobra.Command,
	*marketplace.ListExchangesForListingRequest,
)

func newListExchangesForListing() *cobra.Command {
	cmd := &cobra.Command{}

	var listExchangesForListingReq marketplace.ListExchangesForListingRequest

	cmd.Flags().IntVar(&listExchangesForListingReq.PageSize, "page-size", listExchangesForListingReq.PageSize, ``)
	cmd.Flags().StringVar(&listExchangesForListingReq.PageToken, "page-token", listExchangesForListingReq.PageToken, ``)

	cmd.Use = "list-exchanges-for-listing LISTING_ID"
	cmd.Short = `List exchanges for listing.`
	cmd.Long = `List exchanges for listing.
  
  List exchanges associated with a listing`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listExchangesForListingReq.ListingId = args[0]

		response := w.ProviderExchanges.ListExchangesForListing(ctx, listExchangesForListingReq)
		return cmdio.RenderIterator(ctx, response)
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
	*marketplace.ListListingsForExchangeRequest,
)

func newListListingsForExchange() *cobra.Command {
	cmd := &cobra.Command{}

	var listListingsForExchangeReq marketplace.ListListingsForExchangeRequest

	cmd.Flags().IntVar(&listListingsForExchangeReq.PageSize, "page-size", listListingsForExchangeReq.PageSize, ``)
	cmd.Flags().StringVar(&listListingsForExchangeReq.PageToken, "page-token", listListingsForExchangeReq.PageToken, ``)

	cmd.Use = "list-listings-for-exchange EXCHANGE_ID"
	cmd.Short = `List listings for exchange.`
	cmd.Long = `List listings for exchange.
  
  List listings associated with an exchange`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listListingsForExchangeReq.ExchangeId = args[0]

		response := w.ProviderExchanges.ListListingsForExchange(ctx, listListingsForExchangeReq)
		return cmdio.RenderIterator(ctx, response)
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
	*marketplace.UpdateExchangeRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq marketplace.UpdateExchangeRequest
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update ID"
	cmd.Short = `Update exchange.`
	cmd.Long = `Update exchange.
  
  Update an exchange`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		updateReq.Id = args[0]

		response, err := w.ProviderExchanges.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
