// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package consumer_personalization_requests

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
		Use:   "consumer-personalization-requests",
		Short: `*Public Preview* Personalization Requests allow customers to interact with the individualized Marketplace listing flow.`,
		Long: `This command is in Public Preview and may change without notice.

Personalization Requests allow customers to interact with the individualized
  Marketplace listing flow.`,
		GroupID: "marketplace",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())

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
	*marketplaces.CreatePersonalizationRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq marketplaces.CreatePersonalizationRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.Comment, "comment", ``)
	flags.StringPtrVar(cmd.Flags(), &createReq.Company, "company", ``)
	flags.StringPtrVar(cmd.Flags(), &createReq.FirstName, "first-name", ``)
	flags.BoolPtrVar(cmd.Flags(), &createReq.IsFromLighthouse, "is-from-lighthouse", ``)
	flags.StringPtrVar(cmd.Flags(), &createReq.LastName, "last-name", ``)
	flags.EnumVar(cmd.Flags(), &createReq.RecipientType, "recipient-type", `Supported values: [DELTA_SHARING_RECIPIENT_TYPE_DATABRICKS, DELTA_SHARING_RECIPIENT_TYPE_OPEN]`, "DeltaSharingRecipientType", "DELTA_SHARING_RECIPIENT_TYPE_DATABRICKS", "DELTA_SHARING_RECIPIENT_TYPE_OPEN")

	cmd.Use = "create LISTING_ID"
	cmd.Short = `*Public Preview* Create a personalization request.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create a personalization request.

  Create a personalization request for a listing.`

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
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}
		createReq.ListingId = new(args[0])

		sdkReq, err := marketplaces.CreatePersonalizationRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Marketplaces.CreatePersonalizationRequest(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.CreatePersonalizationResponseFromSDK(response)
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*sdkmarketplaces.GetPersonalizationRequestsForConsumerRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkmarketplaces.GetPersonalizationRequestsForConsumerRequest

	cmd.Use = "get LISTING_ID"
	cmd.Short = `*Public Preview* Get the personalization request for a listing.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get the personalization request for a listing.

  Get the personalization request for a listing. Each consumer can make at
  *most* one personalization request for a listing.`

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

		sdkReq := &getReq
		response, err := w.Marketplaces.GetPersonalizationRequestsForConsumer(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.GetPersonalizationRequestsForConsumerResponseFromSDK(response)
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
	*marketplaces.ListPersonalizationRequestsForConsumerRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq marketplaces.ListPersonalizationRequestsForConsumerRequest

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
	cmd.Short = `*Public Preview* List all personalization requests.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List all personalization requests.

  List personalization requests for a consumer across all listings.`

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

		sdkReq, err := marketplaces.ListPersonalizationRequestsForConsumerRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Marketplaces.ListPersonalizationRequestsForConsumerIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, marketplaces.PersonalizationRequestFromSDK))
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

// end service ConsumerPersonalizationRequests
