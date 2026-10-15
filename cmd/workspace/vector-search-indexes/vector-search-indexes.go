// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package vector_search_indexes

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	vectorsearch "github.com/databricks/cli/libs/sdktypes/vectorsearch/v1"
	sdkvectorsearch "github.com/databricks/sdk-go/vectorsearch/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vector-search-indexes",
		Short: `**Index**: An efficient representation of your embedding vectors that supports real-time and efficient approximate nearest neighbor (ANN) search queries.`,
		Long: `**Index**: An efficient representation of your embedding vectors that supports
  real-time and efficient approximate nearest neighbor (ANN) search queries.

  There are 2 types of AI Search indexes: - **Delta Sync Index**: An index that
  automatically syncs with a source Delta Table, automatically and incrementally
  updating the index as the underlying data in the Delta Table changes. -
  **Direct Vector Access Index**: An index that supports direct read and write
  of vectors and metadata through our REST and SDK APIs. With this model, the
  user manages index updates.`,
		GroupID: "vectorsearch",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreateIndex())
	cmd.AddCommand(newDeleteDataVectorIndex())
	cmd.AddCommand(newDeleteIndex())
	cmd.AddCommand(newGetIndex())
	cmd.AddCommand(newListIndexes())
	cmd.AddCommand(newQueryIndex())
	cmd.AddCommand(newQueryNextPage())
	cmd.AddCommand(newScanIndex())
	cmd.AddCommand(newSyncIndex())
	cmd.AddCommand(newUpsertDataVectorIndex())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-index command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createIndexOverrides []func(
	*cobra.Command,
	*vectorsearch.CreateVectorIndexRequest,
)

func newCreateIndex() *cobra.Command {
	cmd := &cobra.Command{}
	var createIndexReq vectorsearch.CreateVectorIndexRequest

	var createIndexJson flags.JsonFlag

	cmd.Flags().Var(&createIndexJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: delta_sync_index_spec
	// TODO: complex arg: direct_access_index_spec
	flags.EnumVar(cmd.Flags(), &createIndexReq.IndexSubtype, "index-subtype", `The subtype of the index. Supported values: [FULL_TEXT, HYBRID, VECTOR]`, "IndexSubtype", "FULL_TEXT", "HYBRID", "VECTOR")

	cmd.Use = "create-index NAME ENDPOINT_NAME PRIMARY_KEY INDEX_TYPE"
	cmd.Short = `Create an index.`
	cmd.Long = `Create an index.

  Create a new index.

  Arguments:
    NAME: Name of the index
    ENDPOINT_NAME: Name of the endpoint to be used for serving the index
    PRIMARY_KEY: Primary key of the index
    INDEX_TYPE:
      Supported values: [DELTA_SYNC, DIRECT_ACCESS]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'endpoint_name', 'primary_key', 'index_type' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createIndexJson.Unmarshal(&createIndexReq)
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
			createIndexReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createIndexReq.EndpointName = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createIndexReq.PrimaryKey = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			createIndexReq.IndexType = vectorsearch.VectorIndexType(args[3])
		}

		sdkReq, err := vectorsearch.CreateVectorIndexRequestToSDK(&createIndexReq)
		if err != nil {
			return err
		}
		response, err := w.VectorSearch.CreateVectorIndex(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := vectorsearch.VectorIndexFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start delete-data-vector-index command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDataVectorIndexOverrides []func(
	*cobra.Command,
	*vectorsearch.DeleteDataVectorIndexRequest,
)

func newDeleteDataVectorIndex() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteDataVectorIndexReq vectorsearch.DeleteDataVectorIndexRequest

	var deleteDataVectorIndexJson flags.JsonFlag

	cmd.Flags().Var(&deleteDataVectorIndexJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete-data-vector-index INDEX_NAME"
	cmd.Short = `Delete data from index.`
	cmd.Long = `Delete data from index.

  Handles the deletion of data from a specified vector index.

  Arguments:
    INDEX_NAME: Name of the vector index where data is to be deleted. Must be a Direct
      Vector Access Index.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
			diags := deleteDataVectorIndexJson.Unmarshal(&deleteDataVectorIndexReq)
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
		deleteDataVectorIndexReq.Name = new(args[0])

		sdkReq, err := vectorsearch.DeleteDataVectorIndexRequestToSDK(&deleteDataVectorIndexReq)
		if err != nil {
			return err
		}
		response, err := w.VectorSearch.DeleteDataVectorIndex(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := vectorsearch.DeleteDataVectorIndexResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDataVectorIndexOverrides {
		fn(cmd, &deleteDataVectorIndexReq)
	}

	return cmd
}

// start delete-index command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteIndexOverrides []func(
	*cobra.Command,
	*sdkvectorsearch.DeleteVectorIndexRequest,
)

func newDeleteIndex() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteIndexReq sdkvectorsearch.DeleteVectorIndexRequest

	cmd.Use = "delete-index INDEX_NAME"
	cmd.Short = `Delete an index.`
	cmd.Long = `Delete an index.

  Arguments:
    INDEX_NAME: Name of the index`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteIndexReq.Name = new(args[0])

		sdkReq := &deleteIndexReq
		response, err := w.VectorSearch.DeleteVectorIndex(ctx, *sdkReq)
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
	for _, fn := range deleteIndexOverrides {
		fn(cmd, &deleteIndexReq)
	}

	return cmd
}

// start get-index command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getIndexOverrides []func(
	*cobra.Command,
	*vectorsearch.GetVectorIndexRequest,
)

func newGetIndex() *cobra.Command {
	cmd := &cobra.Command{}
	var getIndexReq vectorsearch.GetVectorIndexRequest

	flags.BoolPtrVar(cmd.Flags(), &getIndexReq.EnsureRerankerCompatible, "ensure-reranker-compatible", `If true, the URL returned for the index is guaranteed to be compatible with the reranker.`)

	cmd.Use = "get-index INDEX_NAME"
	cmd.Short = `Get an index.`
	cmd.Long = `Get an index.

  Arguments:
    INDEX_NAME: Name of the index`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getIndexReq.Name = new(args[0])

		sdkReq, err := vectorsearch.GetVectorIndexRequestToSDK(&getIndexReq)
		if err != nil {
			return err
		}
		response, err := w.VectorSearch.GetVectorIndex(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := vectorsearch.VectorIndexFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start list-indexes command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listIndexesOverrides []func(
	*cobra.Command,
	*vectorsearch.ListVectorIndexRequest,
)

func newListIndexes() *cobra.Command {
	cmd := &cobra.Command{}
	var listIndexesReq vectorsearch.ListVectorIndexRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listIndexesLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listIndexesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listIndexesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-indexes ENDPOINT_NAME"
	cmd.Short = `List indexes.`
	cmd.Long = `List indexes.

  List all indexes in the given endpoint.

  Arguments:
    ENDPOINT_NAME: Name of the endpoint`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listIndexesReq.EndpointName = new(args[0])

		sdkReq, err := vectorsearch.ListVectorIndexRequestToSDK(&listIndexesReq)
		if err != nil {
			return err
		}
		if listIndexesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listIndexesLimit)
		}
		if listIndexesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listIndexesLimit)
		}
		response := w.VectorSearch.ListVectorIndexIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, vectorsearch.MiniVectorIndexFromSDK))
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
	*vectorsearch.QueryVectorIndexRequest,
)

func newQueryIndex() *cobra.Command {
	cmd := &cobra.Command{}
	var queryIndexReq vectorsearch.QueryVectorIndexRequest

	var queryIndexJson flags.JsonFlag

	cmd.Flags().Var(&queryIndexJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: columns_to_rerank
	// TODO: array: facets
	flags.StringPtrVar(cmd.Flags(), &queryIndexReq.FiltersJson, "filters-json", `JSON string representing query filters.`)
	flags.IntPtrVar(cmd.Flags(), &queryIndexReq.NumResults, "num-results", `Number of results to return.`)
	// TODO: array: query_columns
	flags.StringPtrVar(cmd.Flags(), &queryIndexReq.QueryText, "query-text", `Query text.`)
	flags.StringPtrVar(cmd.Flags(), &queryIndexReq.QueryType, "query-type", `The query type to use.`)
	// TODO: array: query_vector
	// TODO: complex arg: reranker
	flags.Float32PtrVar(cmd.Flags(), &queryIndexReq.ScoreThreshold, "score-threshold", `Threshold for the approximate nearest neighbor search.`)
	// TODO: array: sort_columns

	cmd.Use = "query-index INDEX_NAME"
	cmd.Short = `Query an index.`
	cmd.Long = `Query an index.

  Query the specified vector index.

  Arguments:
    INDEX_NAME: Name of the vector index to query.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}
		queryIndexReq.Name = new(args[0])

		sdkReq, err := vectorsearch.QueryVectorIndexRequestToSDK(&queryIndexReq)
		if err != nil {
			return err
		}
		response, err := w.VectorSearch.QueryVectorIndex(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := vectorsearch.QueryVectorIndexResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start query-next-page command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var queryNextPageOverrides []func(
	*cobra.Command,
	*vectorsearch.QueryVectorIndexNextPageRequest,
)

func newQueryNextPage() *cobra.Command {
	cmd := &cobra.Command{}
	var queryNextPageReq vectorsearch.QueryVectorIndexNextPageRequest

	var queryNextPageJson flags.JsonFlag

	cmd.Flags().Var(&queryNextPageJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &queryNextPageReq.EndpointName, "endpoint-name", `Name of the endpoint.`)
	flags.StringPtrVar(cmd.Flags(), &queryNextPageReq.PageToken, "page-token", `Page token returned from previous QueryVectorIndex or QueryVectorIndexNextPage API.`)

	cmd.Use = "query-next-page INDEX_NAME"
	cmd.Short = `Query next page.`
	cmd.Long = `Query next page.

  Use next_page_token returned from previous QueryVectorIndex or
  QueryVectorIndexNextPage request to fetch next page of results.

  Arguments:
    INDEX_NAME: Name of the vector index to query.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
			diags := queryNextPageJson.Unmarshal(&queryNextPageReq)
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
		queryNextPageReq.Name = new(args[0])

		sdkReq, err := vectorsearch.QueryVectorIndexNextPageRequestToSDK(&queryNextPageReq)
		if err != nil {
			return err
		}
		response, err := w.VectorSearch.QueryVectorIndexNextPage(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := vectorsearch.QueryVectorIndexResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range queryNextPageOverrides {
		fn(cmd, &queryNextPageReq)
	}

	return cmd
}

// start scan-index command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var scanIndexOverrides []func(
	*cobra.Command,
	*vectorsearch.ScanVectorIndexRequest,
)

func newScanIndex() *cobra.Command {
	cmd := &cobra.Command{}
	var scanIndexReq vectorsearch.ScanVectorIndexRequest

	var scanIndexJson flags.JsonFlag

	cmd.Flags().Var(&scanIndexJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &scanIndexReq.LastPrimaryKey, "last-primary-key", `Primary key of the last entry returned in the previous scan.`)
	flags.IntPtrVar(cmd.Flags(), &scanIndexReq.NumResults, "num-results", `Number of results to return.`)

	cmd.Use = "scan-index INDEX_NAME"
	cmd.Short = `Scan an index.`
	cmd.Long = `Scan an index.

  Scan the specified vector index and return the first num_results entries
  after the exclusive primary_key.

  Arguments:
    INDEX_NAME: Name of the vector index to scan.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
		scanIndexReq.Name = new(args[0])

		sdkReq, err := vectorsearch.ScanVectorIndexRequestToSDK(&scanIndexReq)
		if err != nil {
			return err
		}
		response, err := w.VectorSearch.ScanVectorIndex(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := vectorsearch.ScanVectorIndexResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*vectorsearch.SyncVectorIndexRequest,
)

func newSyncIndex() *cobra.Command {
	cmd := &cobra.Command{}
	var syncIndexReq vectorsearch.SyncVectorIndexRequest

	cmd.Use = "sync-index INDEX_NAME"
	cmd.Short = `Synchronize an index.`
	cmd.Long = `Synchronize an index.

  Triggers a synchronization process for a specified vector index.

  Arguments:
    INDEX_NAME: Name of the vector index to synchronize. Must be a Delta Sync Index.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		syncIndexReq.Name = new(args[0])

		sdkReq, err := vectorsearch.SyncVectorIndexRequestToSDK(&syncIndexReq)
		if err != nil {
			return err
		}
		response, err := w.VectorSearch.SyncVectorIndex(ctx, *sdkReq)
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
	for _, fn := range syncIndexOverrides {
		fn(cmd, &syncIndexReq)
	}

	return cmd
}

// start upsert-data-vector-index command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var upsertDataVectorIndexOverrides []func(
	*cobra.Command,
	*vectorsearch.UpsertDataVectorIndexRequest,
)

func newUpsertDataVectorIndex() *cobra.Command {
	cmd := &cobra.Command{}
	var upsertDataVectorIndexReq vectorsearch.UpsertDataVectorIndexRequest

	var upsertDataVectorIndexJson flags.JsonFlag

	cmd.Flags().Var(&upsertDataVectorIndexJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "upsert-data-vector-index INDEX_NAME INPUTS_JSON"
	cmd.Short = `Upsert data into an index.`
	cmd.Long = `Upsert data into an index.

  Handles the upserting of data into a specified vector index.

  Arguments:
    INDEX_NAME: Name of the vector index where data is to be upserted. Must be a Direct
      Vector Access Index.
    INPUTS_JSON: JSON string representing the data to be upserted.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only INDEX_NAME as positional arguments. Provide 'inputs_json' in your JSON input")
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
			diags := upsertDataVectorIndexJson.Unmarshal(&upsertDataVectorIndexReq)
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
		upsertDataVectorIndexReq.Name = new(args[0])
		if !cmd.Flags().Changed("json") {
			upsertDataVectorIndexReq.InputsJson = new(args[1])
		}

		sdkReq, err := vectorsearch.UpsertDataVectorIndexRequestToSDK(&upsertDataVectorIndexReq)
		if err != nil {
			return err
		}
		response, err := w.VectorSearch.UpsertDataVectorIndex(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := vectorsearch.UpsertDataVectorIndexResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range upsertDataVectorIndexOverrides {
		fn(cmd, &upsertDataVectorIndexReq)
	}

	return cmd
}

// end service VectorSearchIndexes
