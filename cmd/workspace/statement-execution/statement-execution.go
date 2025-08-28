package statement_execution

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "statement-execution",
		Short: "Execute SQL statements",
		Long:  "Execute SQL statements against Databricks SQL warehouses",
	}

	cmd.AddCommand(newExecuteStatementCommand())
	cmd.AddCommand(newGetStatementCommand())

	return cmd
}

func newExecuteStatementCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute-statement STATEMENT",
		Short: "Execute a SQL statement",
		Long: `Execute a SQL statement and optionally await its results.

The warehouse_id is automatically set to a5e694153a0d5e8c by default, but can be overridden:
- Via DATABRICKS_WAREHOUSE_ID environment variable
- As an explicit parameter (for different warehouses)

Default settings:
- warehouse_id: a5e694153a0d5e8c
- wait_timeout: 10s
- format: JSON_ARRAY
- disposition: INLINE
- byte_limit: 16MB (16777216 bytes)

The command supports various options for controlling execution behavior:
- Synchronous execution with timeout
- Asynchronous execution for long-running queries
- Different result dispositions (INLINE or EXTERNAL_LINKS)
- Various output formats (JSON_ARRAY, ARROW_STREAM, CSV)
- Parameterized queries

Examples:
  # Execute with default warehouse (simplest)
  databricks statement-execution execute-statement "SELECT * FROM main.gold_mls.search_listings LIMIT 5"

  # Override with environment variable
  export DATABRICKS_WAREHOUSE_ID=your-other-warehouse-id
  databricks statement-execution execute-statement "SELECT * FROM my_table"

  # Override with explicit parameter
  databricks statement-execution execute-statement your-warehouse-id "SELECT * FROM my_table"

  # Execute with custom timeout
  databricks statement-execution execute-statement "SELECT * FROM my_table" --wait-timeout 30s

  # Execute asynchronously
  databricks statement-execution execute-statement "SELECT * FROM my_table" --wait-timeout 0s

  # Execute with external links for large results
  databricks statement-execution execute-statement "SELECT * FROM my_table" --disposition EXTERNAL_LINKS`,
	}

	var req ExecuteStatementRequest

	cmd.Flags().StringVar(&req.Catalog, "catalog", "", "Default catalog for statement execution")
	cmd.Flags().StringVar(&req.Schema, "schema", "", "Default schema for statement execution")
	cmd.Flags().StringVar(&req.Disposition, "disposition", "INLINE", "Result disposition (INLINE or EXTERNAL_LINKS)")
	cmd.Flags().StringVar(&req.Format, "format", "JSON_ARRAY", "Result format (JSON_ARRAY, ARROW_STREAM, CSV)")
	cmd.Flags().StringVar(&req.WaitTimeout, "wait-timeout", "10s", "Wait timeout for synchronous execution")
	cmd.Flags().StringVar(&req.OnWaitTimeout, "on-wait-timeout", "CONTINUE", "Action on wait timeout (CONTINUE or CANCEL)")
	cmd.Flags().Int64Var(&req.RowLimit, "row-limit", 0, "Row limit for result set")
	cmd.Flags().Int64Var(&req.ByteLimit, "byte-limit", 16777216, "Byte limit for result size (default 16MB)")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 || len(args) > 2 {
			return cmd.Usage()
		}

		// Handle optional warehouse_id parameter
		if len(args) == 2 {
			req.WarehouseId = args[0]
			req.Statement = args[1]
		} else {
			req.Statement = args[0]
			// warehouse_id will be set automatically in the implementation
		}

		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		return executeStatement(ctx, w, &req)
	}

	return cmd
}

func newGetStatementCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-statement STATEMENT_ID",
		Short: "Get statement execution status and results",
		Long: `Get the status and results of a SQL statement execution.

This command is used to poll for the results of an asynchronously executed statement.

Examples:
  # Get statement status
  databricks statement-execution get-statement 12345678-1234-1234-1234-123456789012`,
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return cmd.Usage()
		}

		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		return getStatement(ctx, w, args[0])
	}

	return cmd
}
