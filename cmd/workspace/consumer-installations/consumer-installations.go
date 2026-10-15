// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package consumer_installations

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
		Use:   "consumer-installations",
		Short: `*Public Preview* Installations are entities that allow consumers to interact with Databricks Marketplace listings.`,
		Long: `This command is in Public Preview and may change without notice.

Installations are entities that allow consumers to interact with Databricks
  Marketplace listings.`,
		GroupID: "marketplace",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
	*marketplaces.CreateInstallationRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq marketplaces.CreateInstallationRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: accepted_consumer_terms
	flags.StringPtrVar(cmd.Flags(), &createReq.CatalogName, "catalog-name", ``)
	flags.EnumVar(cmd.Flags(), &createReq.RecipientType, "recipient-type", `Supported values: [DELTA_SHARING_RECIPIENT_TYPE_DATABRICKS, DELTA_SHARING_RECIPIENT_TYPE_OPEN]`, "DeltaSharingRecipientType", "DELTA_SHARING_RECIPIENT_TYPE_DATABRICKS", "DELTA_SHARING_RECIPIENT_TYPE_OPEN")
	// TODO: complex arg: repo_detail
	flags.StringPtrVar(cmd.Flags(), &createReq.ShareName, "share-name", ``)

	cmd.Use = "create LISTING_ID"
	cmd.Short = `*Public Preview* Install from a listing.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Install from a listing.

  Install payload associated with a Databricks Marketplace listing.`

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
		}
		createReq.ListingId = new(args[0])

		sdkReq, err := marketplaces.CreateInstallationRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Marketplaces.InstallListing(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.CreateInstallationResponseFromSDK(response)
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
	*sdkmarketplaces.DeleteInstallationRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkmarketplaces.DeleteInstallationRequest

	cmd.Use = "delete LISTING_ID INSTALLATION_ID"
	cmd.Short = `*Public Preview* Uninstall from a listing.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Uninstall from a listing.

  Uninstall an installation associated with a Databricks Marketplace listing.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteReq.ListingId = new(args[0])
		deleteReq.InstallationId = new(args[1])

		sdkReq := &deleteReq
		response, err := w.Marketplaces.UninstallListing(ctx, *sdkReq)
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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*marketplaces.ListInstallationsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq marketplaces.ListInstallationsRequest

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
	cmd.Short = `*Public Preview* List all installations.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List all installations.

  List all installations across all listings.`

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

		sdkReq, err := marketplaces.ListInstallationsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Marketplaces.ListInstallationsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, marketplaces.InstallationDetailFromSDK))
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
	*marketplaces.GetInstallationDetailsRequest,
)

func newListListingInstallations() *cobra.Command {
	cmd := &cobra.Command{}
	var listListingInstallationsReq marketplaces.GetInstallationDetailsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listListingInstallationsLimit int

	flags.IntPtrVar(cmd.Flags(), &listListingInstallationsReq.PageSize, "page-size", ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listListingInstallationsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listListingInstallationsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-listing-installations LISTING_ID"
	cmd.Short = `*Public Preview* List installations for a listing.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List installations for a listing.

  List all installations for a particular listing.`

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

		listListingInstallationsReq.ListingId = new(args[0])

		sdkReq, err := marketplaces.GetInstallationDetailsRequestToSDK(&listListingInstallationsReq)
		if err != nil {
			return err
		}
		if listListingInstallationsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listListingInstallationsLimit)
		}
		if listListingInstallationsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listListingInstallationsLimit)
		}
		response := w.Marketplaces.GetInstallationDetailsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, marketplaces.InstallationDetailFromSDK))
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
	*marketplaces.UpdateInstallationRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq marketplaces.UpdateInstallationRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &updateReq.RotateToken, "rotate-token", ``)

	cmd.Use = "update LISTING_ID INSTALLATION_ID"
	cmd.Short = `*Public Preview* Update an installation.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update an installation.

  This is a update API that will update the part of the fields defined in the
  installation table as well as interact with external services according to the
  fields not included in the installation table 1. the token will be rotate if
  the rotateToken flag is true 2. the token will be forcibly rotate if the
  rotateToken flag is true and the tokenInfo field is empty`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
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
		updateReq.ListingId = new(args[0])
		updateReq.InstallationId = new(args[1])

		sdkReq, err := marketplaces.UpdateInstallationRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Marketplaces.UpdateInstallationDetail(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.UpdateInstallationResponseFromSDK(response)
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

// end service ConsumerInstallations
