// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package ai_search

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/aisearch"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai-search",
		Short: `*Beta* **AI Search Endpoint**: Represents the compute resources to host AI Search indexes.`,
		Long: `This command is in Beta and may change without notice.

**AI Search Endpoint**: Represents the compute resources to host AI Search
  indexes. AIP-conformant replacement for the legacy VectorSearchEndpoints API;
  functionally equivalent.`,
		GroupID: "aisearch",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	// Add methods
	cmd.AddCommand(newCreateEndpoint())
	cmd.AddCommand(newCreateIndex())
	cmd.AddCommand(newDeleteEndpoint())
	cmd.AddCommand(newDeleteIndex())
	cmd.AddCommand(newGetEndpoint())
	cmd.AddCommand(newGetIndex())
	cmd.AddCommand(newListEndpoints())
	cmd.AddCommand(newListIndexes())
	cmd.AddCommand(newQueryIndex())
	cmd.AddCommand(newRemoveData())
	cmd.AddCommand(newScanIndex())
	cmd.AddCommand(newSyncIndex())
	cmd.AddCommand(newUpdateEndpoint())
	cmd.AddCommand(newUpsertData())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createEndpointOverrides []func(
	*cobra.Command,
	*aisearch.CreateEndpointRequest,
)

func newCreateEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var createEndpointReq aisearch.CreateEndpointRequest
	createEndpointReq.Endpoint = aisearch.Endpoint{}
	var createEndpointJson flags.JsonFlag

	cmd.Flags().Var(&createEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createEndpointReq.EndpointId, "endpoint-id", createEndpointReq.EndpointId, `The user-supplied short name for the Endpoint, per AIP-133.`)
	cmd.Flags().StringVar(&createEndpointReq.Endpoint.BudgetPolicyId, "budget-policy-id", createEndpointReq.Endpoint.BudgetPolicyId, `The user-selected budget policy id for the endpoint.`)
	// TODO: array: custom_tags
	// TODO: complex arg: endpoint_status
	cmd.Flags().StringVar(&createEndpointReq.Endpoint.Name, "name", createEndpointReq.Endpoint.Name, `Name of the AI Search endpoint.`)
	cmd.Flags().IntVar(&createEndpointReq.Endpoint.ReplicaCount, "replica-count", createEndpointReq.Endpoint.ReplicaCount, `The client-supplied desired number of replicas for the endpoint, applied at create/update time.`)
	// TODO: complex arg: scaling_info
	cmd.Flags().IntVar(&createEndpointReq.Endpoint.TargetQps, "target-qps", createEndpointReq.Endpoint.TargetQps, `Target QPS for the endpoint.`)
	// TODO: complex arg: throughput_info
	cmd.Flags().StringVar(&createEndpointReq.Endpoint.UsagePolicyId, "usage-policy-id", createEndpointReq.Endpoint.UsagePolicyId, `The usage policy id applied to the endpoint.`)

	cmd.Use = "create-endpoint PARENT ENDPOINT_TYPE"
	cmd.Short = `*Beta* Create an AI Search endpoint.`
	cmd.Long = `This command is in Beta and may change without notice.

Create an AI Search endpoint.

  Create a new AI Search endpoint.

  Arguments:
    PARENT: The Workspace where this Endpoint will be created. Format:
      workspaces/{workspace_id}
    ENDPOINT_TYPE: Type of endpoint. Required on create and immutable thereafter.
      Supported values: [STANDARD, STORAGE_OPTIMIZED]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only PARENT as positional arguments. Provide 'endpoint_type' in your JSON input")
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
			diags := createEndpointJson.Unmarshal(&createEndpointReq.Endpoint)
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
		createEndpointReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createEndpointReq.Endpoint.EndpointType)
			if err != nil {
				return fmt.Errorf("invalid ENDPOINT_TYPE: %s", args[1])
			}

		}

		response, err := w.AiSearch.CreateEndpoint(ctx, createEndpointReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createEndpointOverrides {
		fn(cmd, &createEndpointReq)
	}

	return cmd
}

// start create-index command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createIndexOverrides []func(
	*cobra.Command,
	*aisearch.CreateIndexRequest,
)

func newCreateIndex() *cobra.Command {
	cmd := &cobra.Command{}

	var createIndexReq aisearch.CreateIndexRequest
	createIndexReq.Index = aisearch.Index{}
	var createIndexJson flags.JsonFlag

	cmd.Flags().Var(&createIndexJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createIndexReq.IndexId, "index-id", createIndexReq.IndexId, `The user-supplied Unity Catalog table name for the Index, per AIP-133.`)
	// TODO: complex arg: delta_sync_index_spec
	// TODO: complex arg: direct_access_index_spec
	cmd.Flags().Var(&createIndexReq.Index.IndexSubtype, "index-subtype", `The subtype of the index. Supported values: [FULL_TEXT, HYBRID, VECTOR]`)
	cmd.Flags().StringVar(&createIndexReq.Index.Name, "name", createIndexReq.Index.Name, `Name of the AI Search index.`)
	// TODO: complex arg: status

	cmd.Use = "create-index PARENT PRIMARY_KEY INDEX_TYPE"
	cmd.Short = `*Beta* Create an AI Search index.`
	cmd.Long = `This command is in Beta and may change without notice.

Create an AI Search index.

  Create a new AI Search index.

  Arguments:
    PARENT: The Endpoint where this Index will be created. Format:
      workspaces/{workspace_id}/endpoints/{endpoint_id}
    PRIMARY_KEY: Primary key of the index. Set on create and immutable thereafter.
    INDEX_TYPE: Type of index. Required on create and immutable thereafter.
      Supported values: [DELTA_SYNC, DIRECT_ACCESS]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only PARENT as positional arguments. Provide 'primary_key', 'index_type' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createIndexJson.Unmarshal(&createIndexReq.Index)
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
		createIndexReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			createIndexReq.Index.PrimaryKey = args[1]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &createIndexReq.Index.IndexType)
			if err != nil {
				return fmt.Errorf("invalid INDEX_TYPE: %s", args[2])
			}

		}

		response, err := w.AiSearch.CreateIndex(ctx, createIndexReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createIndexOverrides {
		fn(cmd, &createIndexReq)
	}

	return cmd
}

// start delete-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteEndpointOverrides []func(
	*cobra.Command,
	*aisearch.DeleteEndpointRequest,
)

func newDeleteEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteEndpointReq aisearch.DeleteEndpointRequest

	cmd.Use = "delete-endpoint NAME"
	cmd.Short = `*Beta* Delete an AI Search endpoint.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete an AI Search endpoint.

  Arguments:
    NAME: Full resource name of the endpoint to delete. Format:
      workspaces/{workspace_id}/endpoints/{endpoint_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteEndpointReq.Name = args[0]

		err = w.AiSearch.DeleteEndpoint(ctx, deleteEndpointReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteEndpointOverrides {
		fn(cmd, &deleteEndpointReq)
	}

	return cmd
}

// start delete-index command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteIndexOverrides []func(
	*cobra.Command,
	*aisearch.DeleteIndexRequest,
)

func newDeleteIndex() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteIndexReq aisearch.DeleteIndexRequest

	cmd.Use = "delete-index NAME"
	cmd.Short = `*Beta* Delete an AI Search index.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete an AI Search index.

  Arguments:
    NAME: Full resource name of the index to delete. Format:
      workspaces/{workspace_id}/endpoints/{endpoint_id}/indexes/{index_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteIndexReq.Name = args[0]

		err = w.AiSearch.DeleteIndex(ctx, deleteIndexReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteIndexOverrides {
		fn(cmd, &deleteIndexReq)
	}

	return cmd
}

// start get-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getEndpointOverrides []func(
	*cobra.Command,
	*aisearch.GetEndpointRequest,
)

func newGetEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var getEndpointReq aisearch.GetEndpointRequest

	cmd.Use = "get-endpoint NAME"
	cmd.Short = `*Beta* Get an AI Search endpoint.`
	cmd.Long = `This command is in Beta and may change without notice.

Get an AI Search endpoint.

  Get details for a single AI Search endpoint.

  Arguments:
    NAME: Full resource name of the endpoint. Format:
      workspaces/{workspace_id}/endpoints/{endpoint_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getEndpointReq.Name = args[0]

		response, err := w.AiSearch.GetEndpoint(ctx, getEndpointReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getEndpointOverrides {
		fn(cmd, &getEndpointReq)
	}

	return cmd
}

// start get-index command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getIndexOverrides []func(
	*cobra.Command,
	*aisearch.GetIndexRequest,
)

func newGetIndex() *cobra.Command {
	cmd := &cobra.Command{}

	var getIndexReq aisearch.GetIndexRequest

	cmd.Use = "get-index NAME"
	cmd.Short = `*Beta* Get an AI Search index.`
	cmd.Long = `This command is in Beta and may change without notice.

Get an AI Search index.

  Get details for a single AI Search index.

  Arguments:
    NAME: Full resource name of the index. Format:
      workspaces/{workspace_id}/endpoints/{endpoint_id}/indexes/{index_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getIndexReq.Name = args[0]

		response, err := w.AiSearch.GetIndex(ctx, getIndexReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getIndexOverrides {
		fn(cmd, &getIndexReq)
	}

	return cmd
}

// start list-endpoints command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listEndpointsOverrides []func(
	*cobra.Command,
	*aisearch.ListEndpointsRequest,
)

func newListEndpoints() *cobra.Command {
	cmd := &cobra.Command{}

	var listEndpointsReq aisearch.ListEndpointsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listEndpointsLimit int

	cmd.Flags().IntVar(&listEndpointsReq.PageSize, "page-size", listEndpointsReq.PageSize, `Best-effort upper bound on the number of results to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listEndpointsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listEndpointsReq.PageToken, "page-token", listEndpointsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-endpoints PARENT"
	cmd.Short = `*Beta* List AI Search endpoints.`
	cmd.Long = `This command is in Beta and may change without notice.

List AI Search endpoints.

  List AI Search endpoints in a workspace.

  Arguments:
    PARENT: The Workspace that owns this collection of endpoints. Format:
      workspaces/{workspace_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listEndpointsReq.Parent = args[0]

		response := w.AiSearch.ListEndpoints(ctx, listEndpointsReq)
		if listEndpointsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listEndpointsLimit)
		}
		if listEndpointsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listEndpointsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listEndpointsOverrides {
		fn(cmd, &listEndpointsReq)
	}

	return cmd
}

// start list-indexes command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listIndexesOverrides []func(
	*cobra.Command,
	*aisearch.ListIndexesRequest,
)

func newListIndexes() *cobra.Command {
	cmd := &cobra.Command{}

	var listIndexesReq aisearch.ListIndexesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listIndexesLimit int

	cmd.Flags().IntVar(&listIndexesReq.PageSize, "page-size", listIndexesReq.PageSize, `Best-effort upper bound on the number of results to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listIndexesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listIndexesReq.PageToken, "page-token", listIndexesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-indexes PARENT"
	cmd.Short = `*Beta* List AI Search indexes.`
	cmd.Long = `This command is in Beta and may change without notice.

List AI Search indexes.

  List AI Search indexes on an endpoint.

  Arguments:
    PARENT: The Endpoint that owns this collection of indexes. Format:
      workspaces/{workspace_id}/endpoints/{endpoint_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listIndexesReq.Parent = args[0]

		response := w.AiSearch.ListIndexes(ctx, listIndexesReq)
		if listIndexesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listIndexesLimit)
		}
		if listIndexesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listIndexesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listIndexesOverrides {
		fn(cmd, &listIndexesReq)
	}

	return cmd
}

// start query-index command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var queryIndexOverrides []func(
	*cobra.Command,
	*aisearch.QueryIndexRequest,
)

func newQueryIndex() *cobra.Command {
	cmd := &cobra.Command{}

	var queryIndexReq aisearch.QueryIndexRequest
	var queryIndexJson flags.JsonFlag

	cmd.Flags().Var(&queryIndexJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: columns_to_rerank
	// TODO: array: facets
	cmd.Flags().StringVar(&queryIndexReq.FiltersJson, "filters-json", queryIndexReq.FiltersJson, `JSON string describing query filters (e.g.`)
	cmd.Flags().IntVar(&queryIndexReq.MaxResults, "max-results", queryIndexReq.MaxResults, `Maximum number of results to return (the legacy num_results).`)
	// TODO: array: query_columns
	cmd.Flags().StringVar(&queryIndexReq.QueryText, "query-text", queryIndexReq.QueryText, `Query text.`)
	cmd.Flags().StringVar(&queryIndexReq.QueryType, "query-type", queryIndexReq.QueryType, `Query type: ANN, HYBRID, or FULL_TEXT.`)
	// TODO: array: query_vector
	// TODO: complex arg: reranker
	cmd.Flags().Float64Var(&queryIndexReq.ScoreThreshold, "score-threshold", queryIndexReq.ScoreThreshold, `Score threshold for the approximate nearest-neighbor search.`)
	// TODO: array: sort_columns

	cmd.Use = "query-index NAME"
	cmd.Short = `*Beta* Query an AI Search index.`
	cmd.Long = `This command is in Beta and may change without notice.

Query an AI Search index.

  Query (search) an AI Search index. Read-only, so a read-scoped token may
  invoke it.

  Arguments:
    NAME: Full resource name of the index to query. Format:
      workspaces/{workspace_id}/endpoints/{endpoint_id}/indexes/{index_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := queryIndexJson.Unmarshal(&queryIndexReq)
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
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		queryIndexReq.Name = args[0]

		response, err := w.AiSearch.QueryIndex(ctx, queryIndexReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range queryIndexOverrides {
		fn(cmd, &queryIndexReq)
	}

	return cmd
}

// start remove-data command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var removeDataOverrides []func(
	*cobra.Command,
	*aisearch.RemoveDataRequest,
)

func newRemoveData() *cobra.Command {
	cmd := &cobra.Command{}

	var removeDataReq aisearch.RemoveDataRequest
	var removeDataJson flags.JsonFlag

	cmd.Flags().Var(&removeDataJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "remove-data NAME"
	cmd.Short = `*Beta* Remove data from an AI Search index.`
	cmd.Long = `This command is in Beta and may change without notice.

Remove data from an AI Search index.

  Remove rows by primary key from a Direct Access AI Search index.

  Arguments:
    NAME: Full resource name of the index. Must be a Direct Access index. Format:
      workspaces/{workspace_id}/endpoints/{endpoint_id}/indexes/{index_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := removeDataJson.Unmarshal(&removeDataReq)
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
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		removeDataReq.Name = args[0]

		response, err := w.AiSearch.RemoveData(ctx, removeDataReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range removeDataOverrides {
		fn(cmd, &removeDataReq)
	}

	return cmd
}

// start scan-index command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var scanIndexOverrides []func(
	*cobra.Command,
	*aisearch.ScanIndexRequest,
)

func newScanIndex() *cobra.Command {
	cmd := &cobra.Command{}

	var scanIndexReq aisearch.ScanIndexRequest
	var scanIndexJson flags.JsonFlag

	cmd.Flags().Var(&scanIndexJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&scanIndexReq.PageSize, "page-size", scanIndexReq.PageSize, `Maximum number of rows to return in this page.`)
	cmd.Flags().StringVar(&scanIndexReq.PageToken, "page-token", scanIndexReq.PageToken, `Page token from a previous response; if unset, scanning starts from the beginning.`)

	cmd.Use = "scan-index NAME"
	cmd.Short = `*Beta* Scan an AI Search index.`
	cmd.Long = `This command is in Beta and may change without notice.

Scan an AI Search index.

  Scan (paginate over) the rows of an AI Search index.

  Arguments:
    NAME: Full resource name of the index to scan. Format:
      workspaces/{workspace_id}/endpoints/{endpoint_id}/indexes/{index_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := scanIndexJson.Unmarshal(&scanIndexReq)
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
		scanIndexReq.Name = args[0]

		response, err := w.AiSearch.ScanIndex(ctx, scanIndexReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range scanIndexOverrides {
		fn(cmd, &scanIndexReq)
	}

	return cmd
}

// start sync-index command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var syncIndexOverrides []func(
	*cobra.Command,
	*aisearch.SyncIndexRequest,
)

func newSyncIndex() *cobra.Command {
	cmd := &cobra.Command{}

	var syncIndexReq aisearch.SyncIndexRequest

	cmd.Use = "sync-index NAME"
	cmd.Short = `*Beta* Synchronize an AI Search index.`
	cmd.Long = `This command is in Beta and may change without notice.

Synchronize an AI Search index.

  Synchronize a Delta Sync AI Search index with its source Delta table. Applies
  only to Delta Sync indexes; Direct Access indexes are written via the
  data-plane upsert path.

  Arguments:
    NAME: Full resource name of the index to synchronize. Must be a Delta Sync
      index. Format:
      workspaces/{workspace_id}/endpoints/{endpoint_id}/indexes/{index_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		syncIndexReq.Name = args[0]

		response, err := w.AiSearch.SyncIndex(ctx, syncIndexReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range syncIndexOverrides {
		fn(cmd, &syncIndexReq)
	}

	return cmd
}

// start update-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateEndpointOverrides []func(
	*cobra.Command,
	*aisearch.UpdateEndpointRequest,
)

func newUpdateEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var updateEndpointReq aisearch.UpdateEndpointRequest
	updateEndpointReq.Endpoint = aisearch.Endpoint{}
	var updateEndpointJson flags.JsonFlag

	cmd.Flags().Var(&updateEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateEndpointReq.Endpoint.BudgetPolicyId, "budget-policy-id", updateEndpointReq.Endpoint.BudgetPolicyId, `The user-selected budget policy id for the endpoint.`)
	// TODO: array: custom_tags
	// TODO: complex arg: endpoint_status
	cmd.Flags().StringVar(&updateEndpointReq.Endpoint.Name, "name", updateEndpointReq.Endpoint.Name, `Name of the AI Search endpoint.`)
	cmd.Flags().IntVar(&updateEndpointReq.Endpoint.ReplicaCount, "replica-count", updateEndpointReq.Endpoint.ReplicaCount, `The client-supplied desired number of replicas for the endpoint, applied at create/update time.`)
	// TODO: complex arg: scaling_info
	cmd.Flags().IntVar(&updateEndpointReq.Endpoint.TargetQps, "target-qps", updateEndpointReq.Endpoint.TargetQps, `Target QPS for the endpoint.`)
	// TODO: complex arg: throughput_info
	cmd.Flags().StringVar(&updateEndpointReq.Endpoint.UsagePolicyId, "usage-policy-id", updateEndpointReq.Endpoint.UsagePolicyId, `The usage policy id applied to the endpoint.`)

	cmd.Use = "update-endpoint NAME UPDATE_MASK ENDPOINT_TYPE"
	cmd.Short = `*Beta* Update an AI Search endpoint.`
	cmd.Long = `This command is in Beta and may change without notice.

Update an AI Search endpoint.

  Update an existing AI Search endpoint. Multi-bucket masks are supported and
  dispatched in deterministic bucket order: budget policy, custom tags,
  throughput, then scaling/replicas. Per-bucket dispatch is not atomic across
  buckets — if a later bucket fails, earlier buckets may already have been
  applied.

  Arguments:
    NAME: Name of the AI Search endpoint. Server-assigned full resource path
      (workspaces/{workspace}/endpoints/{endpoint}) on output. On create, the
      user-supplied short name is conveyed via
      CreateEndpointRequest.endpoint_id; the server composes the full name
      and returns it on the response.
    UPDATE_MASK: The list of fields to update.
    ENDPOINT_TYPE: Type of endpoint. Required on create and immutable thereafter.
      Supported values: [STANDARD, STORAGE_OPTIMIZED]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'endpoint_type' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateEndpointJson.Unmarshal(&updateEndpointReq.Endpoint)
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
		updateEndpointReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateEndpointReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &updateEndpointReq.Endpoint.EndpointType)
			if err != nil {
				return fmt.Errorf("invalid ENDPOINT_TYPE: %s", args[2])
			}

		}

		response, err := w.AiSearch.UpdateEndpoint(ctx, updateEndpointReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateEndpointOverrides {
		fn(cmd, &updateEndpointReq)
	}

	return cmd
}

// start upsert-data command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var upsertDataOverrides []func(
	*cobra.Command,
	*aisearch.UpsertDataRequest,
)

func newUpsertData() *cobra.Command {
	cmd := &cobra.Command{}

	var upsertDataReq aisearch.UpsertDataRequest
	var upsertDataJson flags.JsonFlag

	cmd.Flags().Var(&upsertDataJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "upsert-data NAME INPUTS_JSON"
	cmd.Short = `*Beta* Upsert data into an AI Search index.`
	cmd.Long = `This command is in Beta and may change without notice.

Upsert data into an AI Search index.

  Upsert rows into a Direct Access AI Search index.

  Arguments:
    NAME: Full resource name of the index. Must be a Direct Access index. Format:
      workspaces/{workspace_id}/endpoints/{endpoint_id}/indexes/{index_id}
    INPUTS_JSON: JSON document describing the rows to upsert.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME as positional arguments. Provide 'inputs_json' in your JSON input")
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
			diags := upsertDataJson.Unmarshal(&upsertDataReq)
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
		upsertDataReq.Name = args[0]
		if !cmd.Flags().Changed("json") {
			upsertDataReq.InputsJson = args[1]
		}

		response, err := w.AiSearch.UpsertData(ctx, upsertDataReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range upsertDataOverrides {
		fn(cmd, &upsertDataReq)
	}

	return cmd
}

// end service AiSearch
