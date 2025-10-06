// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package consumer_installations

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
		Use:   "consumer-installations",
		Short: `Installations are entities that allow consumers to interact with Databricks Marketplace listings.`,
		Long: `Installations are entities that allow consumers to interact with Databricks
  Marketplace listings.`,
		GroupID: "marketplace",
		Annotations: map[string]string{
			"package": "marketplace",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListListingInstallations())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*marketplace.CreateInstallationRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq marketplace.CreateInstallationRequest
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: accepted_consumer_terms
	cmd.Flags().StringVar(&createReq.CatalogName, "catalog-name", createReq.CatalogName, ``)
	cmd.Flags().Var(&createReq.RecipientType, "recipient-type", `Supported values: [DELTA_SHARING_RECIPIENT_TYPE_DATABRICKS, DELTA_SHARING_RECIPIENT_TYPE_OPEN]`)
	// TODO: complex arg: repo_detail
	cmd.Flags().StringVar(&createReq.ShareName, "share-name", createReq.ShareName, ``)

	cmd.Use = "create LISTING_ID"
	cmd.Short = `Install from a listing.`
	cmd.Long = `Install from a listing.
  
  Install payload associated with a Databricks Marketplace listing.`

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
		}
		createReq.ListingId = args[0]

		response, err := w.ConsumerInstallations.Create(ctx, createReq)
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
	*marketplace.DeleteInstallationRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq marketplace.DeleteInstallationRequest

	cmd.Use = "delete LISTING_ID INSTALLATION_ID"
	cmd.Short = `Uninstall from a listing.`
	cmd.Long = `Uninstall from a listing.
  
  Uninstall an installation associated with a Databricks Marketplace listing.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.ListingId = args[0]
		deleteReq.InstallationId = args[1]

		err = w.ConsumerInstallations.Delete(ctx, deleteReq)
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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*marketplace.ListAllInstallationsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq marketplace.ListAllInstallationsRequest

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, ``)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, ``)

	cmd.Use = "list"
	cmd.Short = `List all installations.`
	cmd.Long = `List all installations.
  
  List all installations across all listings.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ConsumerInstallations.List(ctx, listReq)
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

// start list-listing-installations command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listListingInstallationsOverrides []func(
	*cobra.Command,
	*marketplace.ListInstallationsRequest,
)

func newListListingInstallations() *cobra.Command {
	cmd := &cobra.Command{}

	var listListingInstallationsReq marketplace.ListInstallationsRequest

	cmd.Flags().IntVar(&listListingInstallationsReq.PageSize, "page-size", listListingInstallationsReq.PageSize, ``)
	cmd.Flags().StringVar(&listListingInstallationsReq.PageToken, "page-token", listListingInstallationsReq.PageToken, ``)

	cmd.Use = "list-listing-installations LISTING_ID"
	cmd.Short = `List installations for a listing.`
	cmd.Long = `List installations for a listing.
  
  List all installations for a particular listing.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listListingInstallationsReq.ListingId = args[0]

		response := w.ConsumerInstallations.ListListingInstallations(ctx, listListingInstallationsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listListingInstallationsOverrides {
		fn(cmd, &listListingInstallationsReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*marketplace.UpdateInstallationRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq marketplace.UpdateInstallationRequest
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&updateReq.RotateToken, "rotate-token", updateReq.RotateToken, ``)

	cmd.Use = "update LISTING_ID INSTALLATION_ID"
	cmd.Short = `Update an installation.`
	cmd.Long = `Update an installation.
  
  This is a update API that will update the part of the fields defined in the
  installation table as well as interact with external services according to the
  fields not included in the installation table 1. the token will be rotate if
  the rotateToken flag is true 2. the token will be forcibly rotate if the
  rotateToken flag is true and the tokenInfo field is empty`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
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
		updateReq.ListingId = args[0]
		updateReq.InstallationId = args[1]

		response, err := w.ConsumerInstallations.Update(ctx, updateReq)
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

// end service ConsumerInstallations
