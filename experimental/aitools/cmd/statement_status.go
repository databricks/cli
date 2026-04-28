package aitools

import (
	"context"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

func newStatementStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status STATEMENT_ID",
		Short: "Return the current state of a statement without polling",
		Long: `Single GET against the Statements API. Use this to peek at progress
without blocking. For a blocking poll-until-terminal call, use
'statement get'.`,
		Example: `  databricks experimental aitools tools statement status 01ef...`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			statementID := args[0]

			info, err := getStatementStatus(ctx, w.StatementExecution, statementID)
			if err != nil {
				return err
			}
			return renderStatementInfo(cmd.OutOrStdout(), info)
		},
	}

	return cmd
}

// getStatementStatus performs a single GET against the Statements API, no polling.
func getStatementStatus(ctx context.Context, api sql.StatementExecutionInterface, statementID string) (statementInfo, error) {
	resp, err := api.GetStatementByStatementId(ctx, statementID)
	if err != nil {
		return statementInfo{}, fmt.Errorf("get statement: %w", err)
	}

	info := statementInfo{StatementID: resp.StatementId}
	if resp.Status != nil {
		info.State = resp.Status.State
	}
	info.Error = statementErrorFromStatus(resp.Status)
	return info, nil
}
