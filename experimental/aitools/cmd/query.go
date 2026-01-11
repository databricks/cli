package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/aitools/lib/middlewares"
	"github.com/databricks/cli/experimental/aitools/lib/session"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

func newQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query SQL",
		Short: "Execute SQL against a Databricks warehouse",
		Long: `Execute a SQL statement against a Databricks SQL warehouse and return results.

The command auto-detects an available warehouse unless DATABRICKS_WAREHOUSE_ID is set.

Output includes the query results as JSON and row count.`,
		Example: `  databricks experimental aitools query "SELECT * FROM samples.nyctaxi.trips LIMIT 5"`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			sqlStatement := cleanSQL(args[0])
			if sqlStatement == "" {
				return errors.New("SQL statement is required")
			}

			// set up session with client for middleware compatibility
			sess := session.NewSession()
			sess.Set(middlewares.DatabricksClientKey, w)
			ctx = session.WithSession(ctx, sess)

			warehouseID, err := middlewares.GetWarehouseID(ctx)
			if err != nil {
				return err
			}

			resp, err := w.StatementExecution.ExecuteAndWait(ctx, sql.ExecuteStatementRequest{
				WarehouseId: warehouseID,
				Statement:   sqlStatement,
				WaitTimeout: "50s",
			})
			if err != nil {
				return fmt.Errorf("execute statement: %w", err)
			}

			if resp.Status != nil && resp.Status.State == sql.StatementStateFailed {
				errMsg := "query failed"
				if resp.Status.Error != nil {
					errMsg = resp.Status.Error.Message
				}
				return errors.New(errMsg)
			}

			output, err := formatQueryResult(resp)
			if err != nil {
				return err
			}

			cmdio.LogString(ctx, output)
			return nil
		},
	}

	return cmd
}

// cleanSQL removes surrounding quotes, empty lines, and SQL comments.
func cleanSQL(s string) string {
	s = strings.TrimSpace(s)
	// remove surrounding quotes if present
	if (strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)) ||
		(strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`)) {
		s = s[1 : len(s)-1]
	}

	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		// skip empty lines and single-line comments
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func formatQueryResult(resp *sql.StatementResponse) (string, error) {
	var sb strings.Builder

	if resp.Manifest == nil || resp.Result == nil {
		sb.WriteString("Query executed successfully (no results)\n")
		return sb.String(), nil
	}

	// get column names
	var columns []string
	if resp.Manifest.Schema != nil {
		for _, col := range resp.Manifest.Schema.Columns {
			columns = append(columns, col.Name)
		}
	}

	// format as JSON array for consistency with Neon API
	var rows []map[string]any
	if resp.Result.DataArray != nil {
		for _, row := range resp.Result.DataArray {
			rowMap := make(map[string]any)
			for i, val := range row {
				if i < len(columns) {
					rowMap[columns[i]] = val
				}
			}
			rows = append(rows, rowMap)
		}
	}

	output, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal results: %w", err)
	}

	sb.Write(output)
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("Row count: %d\n", len(rows)))

	return sb.String(), nil
}
