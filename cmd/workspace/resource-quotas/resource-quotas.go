// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package resource_quotas

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	ucresourcequotas "github.com/databricks/cli/libs/sdktypes/uc/resourcequotas/v1"
	sdkucresourcequotas "github.com/databricks/sdk-go/uc/resourcequotas/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resource-quotas",
		Short: `Unity Catalog enforces resource quotas on all securable objects, which limits the number of resources that can be created.`,
		Long: `Unity Catalog enforces resource quotas on all securable objects, which limits
  the number of resources that can be created. Quotas are expressed in terms of
  a resource type and a parent (for example, tables per metastore or schemas per
  catalog). The resource quota APIs enable you to monitor your current usage and
  limits. For more information on resource quotas see the [Unity Catalog
  documentation].

  [Unity Catalog documentation]: https://docs.databricks.com/en/data-governance/unity-catalog/index.html#resource-quotas`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newGetQuota())
	cmd.AddCommand(newListQuotas())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-quota command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getQuotaOverrides []func(
	*cobra.Command,
	*sdkucresourcequotas.GetQuotaRequest,
)

func newGetQuota() *cobra.Command {
	cmd := &cobra.Command{}
	var getQuotaReq sdkucresourcequotas.GetQuotaRequest

	cmd.Use = "get-quota PARENT_SECURABLE_TYPE PARENT_FULL_NAME QUOTA_NAME"
	cmd.Short = `Get information for a single resource quota.`
	cmd.Long = `Get information for a single resource quota.

  The GetQuota API returns usage information for a single resource quota,
  defined as a child-parent pair. This API also refreshes the quota count if it
  is out of date. Refreshes are triggered asynchronously. The updated count
  might not be returned in the first call.

  Arguments:
    PARENT_SECURABLE_TYPE: Securable type of the quota parent.
    PARENT_FULL_NAME: Full name of the parent resource. Provide the metastore ID if the parent
      is a metastore.
    QUOTA_NAME: Name of the quota. Follows the pattern of the quota type, with "-quota"
      added as a suffix.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getQuotaReq.ParentSecurableType = new(args[0])
		getQuotaReq.ParentFullName = new(args[1])
		getQuotaReq.QuotaName = new(args[2])

		sdkReq := &getQuotaReq
		response, err := w.ResourceQuotas.GetQuota(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucresourcequotas.GetQuotaResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getQuotaOverrides {
		fn(cmd, &getQuotaReq)
	}

	return cmd
}

// start list-quotas command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listQuotasOverrides []func(
	*cobra.Command,
	*ucresourcequotas.ListQuotasRequest,
)

func newListQuotas() *cobra.Command {
	cmd := &cobra.Command{}
	var listQuotasReq ucresourcequotas.ListQuotasRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listQuotasLimit int

	flags.IntPtrVar(cmd.Flags(), &listQuotasReq.MaxResults, "max-results", `The number of quotas to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listQuotasLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listQuotasReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-quotas"
	cmd.Short = `List all resource quotas under a metastore.`
	cmd.Long = `List all resource quotas under a metastore.

  ListQuotas returns all quota values under the metastore. There are no SLAs on
  the freshness of the counts returned. This API does not trigger a refresh of
  quota counts.

  PAGINATION BEHAVIOR: The API is by default paginated, a page may contain zero
  results while still providing a next_page_token. Clients must continue reading
  pages until next_page_token is absent, which is the only indication that the
  end of results has been reached.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := ucresourcequotas.ListQuotasRequestToSDK(&listQuotasReq)
		if err != nil {
			return err
		}
		if listQuotasLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listQuotasLimit)
		}
		if listQuotasLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listQuotasLimit)
		}
		response := w.ResourceQuotas.ListQuotasIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, ucresourcequotas.QuotaInfoFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listQuotasOverrides {
		fn(cmd, &listQuotasReq)
	}

	return cmd
}

// end service ResourceQuotas
