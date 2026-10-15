// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package consumer_providers

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
		Use:   "consumer-providers",
		Short: `*Public Preview* Providers are the entities that publish listings to the Marketplace.`,
		Long: `This command is in Public Preview and may change without notice.

Providers are the entities that publish listings to the Marketplace.`,
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
	*marketplaces.BatchGetProvidersRequest,
)

func newBatchGet() *cobra.Command {
	cmd := &cobra.Command{}
	var batchGetReq marketplaces.BatchGetProvidersRequest

	// TODO: array: ids

	cmd.Use = "batch-get"
	cmd.Short = `*Public Preview* Get one batch of providers. One may specify up to 50 IDs per request.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get one batch of providers. One may specify up to 50 IDs per request.

  Batch get a provider in the Databricks Marketplace with at least one visible
  listing.`

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

		sdkReq, err := marketplaces.BatchGetProvidersRequestToSDK(&batchGetReq)
		if err != nil {
			return err
		}
		response, err := w.Marketplaces.BatchGetProviders(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.BatchGetProvidersResponseFromSDK(response)
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
	*sdkmarketplaces.GetPublishedProviderForConsumerRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkmarketplaces.GetPublishedProviderForConsumerRequest

	cmd.Use = "get ID"
	cmd.Short = `*Public Preview* Get a provider.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get a provider.

  Get a provider in the Databricks Marketplace with at least one visible
  listing.`

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
			sp.Update("No ID argument specified. Loading names for Consumer Providers drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Marketplaces.ListPublishedProvidersForConsumerIter(ctx, sdkmarketplaces.ListPublishedProvidersForConsumerRequest{}), []string{"Name"}, []string{"Id"})
			sp.Close()
			if err != nil {
				return fmt.Errorf("failed to load names for Consumer Providers drop-down. Please manually specify required arguments. Original error: %w", err)
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
		response, err := w.Marketplaces.GetPublishedProviderForConsumer(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := marketplaces.GetPublishedProviderForConsumerResponseFromSDK(response)
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
	*marketplaces.ListPublishedProvidersForConsumerRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq marketplaces.ListPublishedProvidersForConsumerRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.BoolPtrVar(cmd.Flags(), &listReq.IsFeatured, "is-featured", ``)
	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `*Public Preview* List providers.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List providers.

  List all providers in the Databricks Marketplace with at least one visible
  listing.`

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

		sdkReq, err := marketplaces.ListPublishedProvidersForConsumerRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Marketplaces.ListPublishedProvidersForConsumerIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, marketplaces.ProviderInfoFromSDK))
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

// end service ConsumerProviders
