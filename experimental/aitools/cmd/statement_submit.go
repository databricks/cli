package aitools

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

func newStatementSubmitCmd() *cobra.Command {
	var warehouseID string
	var filePath string
	// resolved by PreRunE so input validation runs before any auth/profile
	// work and the documented "validates input before WorkspaceClient" claim
	// in the PR description is actually true.
	var sqlStatement string

	cmd := &cobra.Command{
		Use:   "submit [SQL | file.sql]",
		Short: "Submit a SQL statement asynchronously and return its statement_id",
		Long: `Submit a SQL statement to a Databricks SQL warehouse and return its
statement_id immediately, without waiting for results.

The statement keeps running server-side. Harvest results with
'statement get <id>', inspect with 'statement status <id>', or stop
with 'statement cancel <id>'.`,
		Example: `  databricks experimental aitools tools statement submit "SELECT pg_sleep(60)" --warehouse <wh>
  databricks experimental aitools tools statement submit --file query.sql`,
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			var fps []string
			if filePath != "" {
				fps = []string{filePath}
			}
			sqls, err := resolveSQLs(ctx, cmd, args, fps)
			if err != nil {
				return err
			}
			if len(sqls) != 1 {
				return errors.New("submit accepts exactly one SQL statement; pass multiple to 'query' for batch")
			}
			sqlStatement = sqls[0]

			return root.MustWorkspaceClient(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			wID, err := resolveWarehouseID(ctx, w, warehouseID)
			if err != nil {
				return err
			}

			info, err := submitStatement(ctx, w.StatementExecution, sqlStatement, wID)
			if err != nil {
				return err
			}
			return renderStatementInfo(cmd.OutOrStdout(), info)
		},
	}

	cmd.Flags().StringVarP(&warehouseID, "warehouse", "w", "", "SQL warehouse ID to use for execution")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to a SQL file to execute")

	return cmd
}

// submitStatement issues an asynchronous ExecuteStatement and returns the handle.
func submitStatement(ctx context.Context, api sql.StatementExecutionInterface, statement, warehouseID string) (statementInfo, error) {
	resp, err := api.ExecuteStatement(ctx, sql.ExecuteStatementRequest{
		WarehouseId:   warehouseID,
		Statement:     statement,
		WaitTimeout:   "0s",
		OnWaitTimeout: sql.ExecuteStatementRequestOnWaitTimeoutContinue,
	})
	if err != nil {
		return statementInfo{}, fmt.Errorf("execute statement: %w", err)
	}

	info := statementInfo{
		StatementID: resp.StatementId,
		WarehouseID: warehouseID,
	}
	if resp.Status != nil {
		info.State = resp.Status.State
	}
	return info, nil
}
