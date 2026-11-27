// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package statement_execution_v2

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/statementexecutionv2"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

// Spec declares the command for this service.
var Spec = root.CommandSpec{
	Use:     "statement-execution-v2",
	Short:   `*Public Preview* The Statement Execution API v2 can be used to execute SQL statements on a SQL warehouse and fetch the result.`,
	GroupID: "sql",
	Build:   buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `This command is in Public Preview and may change without notice.

The Statement Execution API v2 can be used to execute SQL statements on a SQL
  warehouse and fetch the result. It works the same as the Statement Execution
  API (v1), except that warehouse_id is a path parameter on every call,
  including polling, fetching result chunks, and canceling. Routing on the
  warehouse ID gives lower latency and higher concurrency.

  Version 2 currently supports [Lakehouse//RT] warehouses only. For other
  warehouse types, use the Statement Execution API (v1). You can also run
  statements on Lakehouse//RT warehouses with version 1, but version 2 is
  optimized for them.

  [Lakehouse//RT]: https://docs.databricks.com/compute/sql-warehouse/real-time.html`
	cmd.RunE = root.ReportUnknownSubcommand

	cmd.Annotations = map[string]string{
		"launch_stage":         "PUBLIC_PREVIEW",
		"launch_stage_display": "Public Preview",
	}

	// Add methods
	cmd.AddCommand(newCancelStatement())
	cmd.AddCommand(newExecuteStatement())
	cmd.AddCommand(newGetResultData())
	cmd.AddCommand(newGetStatementResult())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}
}

// start cancel-statement command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cancelStatementOverrides []func(
	*cobra.Command,
	*statementexecutionv2.CancelStatementRequest,
)

func newCancelStatement() *cobra.Command {
	cmd := &cobra.Command{}

	var cancelStatementReq statementexecutionv2.CancelStatementRequest

	cmd.Use = "cancel-statement WAREHOUSE_ID STATEMENT_ID"
	cmd.Short = `*Public Preview* Cancel statement execution.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Cancel statement execution.

  Requests that an executing statement be canceled. Callers must poll for status
  to see the terminal state. The cancel response is empty; receiving a response
  indicates successful receipt.

  Arguments:
    WAREHOUSE_ID: The ID of the SQL warehouse that ran the statement.
    STATEMENT_ID: The statement ID is returned upon successfully submitting a SQL statement,
      and is a required reference for all subsequent calls.`

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

		cancelStatementReq.WarehouseId = args[0]
		cancelStatementReq.StatementId = args[1]

		response, err := w.StatementExecutionV2.CancelStatement(ctx, cancelStatementReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range cancelStatementOverrides {
		fn(cmd, &cancelStatementReq)
	}

	return cmd
}

// start execute-statement command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var executeStatementOverrides []func(
	*cobra.Command,
	*statementexecutionv2.ExecuteStatementRequest,
)

func newExecuteStatement() *cobra.Command {
	cmd := &cobra.Command{}

	var executeStatementReq statementexecutionv2.ExecuteStatementRequest
	var executeStatementJson flags.JsonFlag

	cmd.Flags().Var(&executeStatementJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Int64Var(&executeStatementReq.ByteLimit, "byte-limit", executeStatementReq.ByteLimit, `Applies the given byte limit to the statement's result size.`)
	cmd.Flags().StringVar(&executeStatementReq.Catalog, "catalog", executeStatementReq.Catalog, `Sets default catalog for statement execution, similar to [USE CATALOG](https://docs.databricks.com/sql/language-manual/sql-ref-syntax-ddl-use-catalog.html) in SQL.`)
	cmd.Flags().Var(&executeStatementReq.Disposition, "disposition", `The fetch disposition provides two modes of fetching results: INLINE and EXTERNAL_LINKS. Supported values: [EXTERNAL_LINKS, INLINE]`)
	cmd.Flags().Var(&executeStatementReq.Format, "format", `Statement execution supports three result formats: JSON_ARRAY, ARROW_STREAM, and CSV. Supported values: [ARROW_STREAM, CSV, JSON_ARRAY]`)
	cmd.Flags().Var(&executeStatementReq.OnWaitTimeout, "on-wait-timeout", `When wait_timeout > 0s, the call will block up to the specified time. Supported values: [CANCEL, CONTINUE]`)
	// TODO: array: parameters
	// TODO: array: query_tags
	cmd.Flags().Int64Var(&executeStatementReq.RowLimit, "row-limit", executeStatementReq.RowLimit, `Applies the given row limit to the statement's result set, but unlike the LIMIT clause in SQL, it also sets the truncated field in the response to indicate whether the result was trimmed due to the limit or not.`)
	cmd.Flags().StringVar(&executeStatementReq.Schema, "schema", executeStatementReq.Schema, `Sets default schema for statement execution, similar to [USE SCHEMA](https://docs.databricks.com/sql/language-manual/sql-ref-syntax-ddl-use-schema.html) in SQL.`)
	cmd.Flags().StringVar(&executeStatementReq.WaitTimeout, "wait-timeout", executeStatementReq.WaitTimeout, `The time in seconds the call will wait for the statement's result set as Ns, where N can be set to 0 or to a value between 5 and 50.`)

	cmd.Use = "execute-statement WAREHOUSE_ID STATEMENT"
	cmd.Short = `*Public Preview* Execute a SQL statement.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Execute a SQL statement.

  Execute a SQL statement and optionally await its results for a specified time.

  **Use case: small result sets with INLINE + JSON_ARRAY**

  For flows that generate small and predictable result sets (<= 25 MiB),
  INLINE responses of JSON_ARRAY result data are typically the simplest way
  to execute and fetch result data.

  **Use case: large result sets with EXTERNAL_LINKS**

  Using EXTERNAL_LINKS to fetch result data allows you to fetch large result
  sets efficiently. The main differences from using INLINE disposition are
  that the result data is accessed with short-lived cloud-storage URLs, and that
  there are 3 supported formats: JSON_ARRAY, ARROW_STREAM and CSV compared
  to only JSON_ARRAY with INLINE.

  **External-link URLs**

  External links point to data stored within your workspace's internal cloud
  storage. The URLs are valid for only a short period, <= 15 minutes. Alongside
  each external_link is an expiration field indicating the time at which the
  URL is no longer valid. In EXTERNAL_LINKS mode, chunks can be resolved and
  fetched multiple times and in parallel.

  ----

  ### **Warning: Databricks strongly recommends that you protect the URLs that
  are returned by the EXTERNAL_LINKS disposition.**

  When you use the EXTERNAL_LINKS disposition, a short-lived cloud-storage URL
  is generated to download the results. The URL contains temporary access
  credentials, so protect it and do not set an Authorization header in the
  download request.

  The EXTERNAL_LINKS disposition can be disabled upon request by creating a
  [support case].

  See also [Security best practices].

  ----

  StatementResponse contains statement_id and status; other fields might be
  absent or present depending on context. If the SQL warehouse fails to execute
  the provided statement, a 200 response is returned with status.state set to
  FAILED (in contrast to a failure when accepting the request, which results
  in a non-200 response). Details of the error can be found at status.error in
  case of execution failures.

  [Security best practices]: https://docs.databricks.com/sql/admin/sql-execution-tutorial.html#security-best-practices
  [support case]: https://docs.databricks.com/resources/support.html

  Arguments:
    WAREHOUSE_ID: The ID of the SQL warehouse to run the statement on. Currently, only
      [Lakehouse//RT] warehouses are supported. See also [What are SQL
      warehouses?].

      [Lakehouse//RT]: https://docs.databricks.com/compute/sql-warehouse/real-time.html
      [What are SQL warehouses?]: https://docs.databricks.com/sql/admin/warehouse-type.html
    STATEMENT: The SQL statement to execute. The statement can optionally be
      parameterized, see parameters. The maximum query text size is 16 MiB.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only WAREHOUSE_ID as positional arguments. Provide 'statement' in your JSON input")
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
			diags := executeStatementJson.Unmarshal(&executeStatementReq)
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
		executeStatementReq.WarehouseId = args[0]
		if !cmd.Flags().Changed("json") {
			executeStatementReq.Statement = args[1]
		}

		response, err := w.StatementExecutionV2.ExecuteStatement(ctx, executeStatementReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range executeStatementOverrides {
		fn(cmd, &executeStatementReq)
	}

	return cmd
}

// start get-result-data command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getResultDataOverrides []func(
	*cobra.Command,
	*statementexecutionv2.GetResultDataRequest,
)

func newGetResultData() *cobra.Command {
	cmd := &cobra.Command{}

	var getResultDataReq statementexecutionv2.GetResultDataRequest

	cmd.Use = "get-result-data WAREHOUSE_ID STATEMENT_ID CHUNK_INDEX"
	cmd.Short = `*Public Preview* Get result chunk by index.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get result chunk by index.

  After the statement execution has SUCCEEDED, this request can be used to
  fetch any chunk by index. Whereas the first chunk with chunk_index=0 is
  typically fetched with the execute or the get-status request, this request can
  be used to fetch subsequent chunks. The response structure is identical to the
  nested result element of those responses, and similarly includes the
  next_chunk_index field for simple iteration through the result set.
  Depending on disposition, the response returns chunks of data either inline,
  or as links.

  Arguments:
    WAREHOUSE_ID: The ID of the SQL warehouse that ran the statement.
    STATEMENT_ID: The statement ID is returned upon successfully submitting a SQL statement,
      and is a required reference for all subsequent calls.
    CHUNK_INDEX: The zero-based index of the result chunk to fetch.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getResultDataReq.WarehouseId = args[0]
		getResultDataReq.StatementId = args[1]
		_, err = fmt.Sscan(args[2], &getResultDataReq.ChunkIndex)
		if err != nil {
			return fmt.Errorf("invalid CHUNK_INDEX: %s", args[2])
		}

		response, err := w.StatementExecutionV2.GetResultData(ctx, getResultDataReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getResultDataOverrides {
		fn(cmd, &getResultDataReq)
	}

	return cmd
}

// start get-statement-result command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getStatementResultOverrides []func(
	*cobra.Command,
	*statementexecutionv2.GetStatementResultRequest,
)

func newGetStatementResult() *cobra.Command {
	cmd := &cobra.Command{}

	var getStatementResultReq statementexecutionv2.GetStatementResultRequest

	cmd.Use = "get-statement-result WAREHOUSE_ID STATEMENT_ID"
	cmd.Short = `*Public Preview* Get status, manifest, and result first chunk.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get status, manifest, and result first chunk.

  This request can be used to poll for the statement's status. StatementResponse
  contains statement_id and status; other fields might be absent or present
  depending on context. When the status.state field is SUCCEEDED it will
  also return the result manifest and the first chunk of the result data. When
  the statement is in the terminal states CANCELED, CLOSED or FAILED, it
  returns HTTP 200 with the state set. After at least 12 hours in a terminal
  state, the statement is removed from the warehouse and further calls will
  receive an HTTP 404 response.

  **NOTE** This call currently might take up to 5 seconds to get the latest
  status and result.

  Arguments:
    WAREHOUSE_ID: The ID of the SQL warehouse that ran the statement.
    STATEMENT_ID: The statement ID is returned upon successfully submitting a SQL statement,
      and is a required reference for all subsequent calls.`

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

		getStatementResultReq.WarehouseId = args[0]
		getStatementResultReq.StatementId = args[1]

		response, err := w.StatementExecutionV2.GetStatementResult(ctx, getStatementResultReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getStatementResultOverrides {
		fn(cmd, &getStatementResultReq)
	}

	return cmd
}

// end service StatementExecutionV2
