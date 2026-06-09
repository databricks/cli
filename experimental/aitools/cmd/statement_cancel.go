package aitools

import (
	"context"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/sqlexec"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

func newStatementCancelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel STATEMENT_ID",
		Short: "Request cancellation of a running statement",
		Long: `Send a cancellation request for the given statement_id. The Statements
API returns no body on cancel; this command optimistically reports
state=CANCELED on success. Use 'statement status' afterwards to confirm
the server-side state if you need certainty.`,
		Example: `  databricks experimental aitools tools statement cancel 01ef...`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			statementID := args[0]

			info, err := cancelStatementExecution(ctx, w.StatementExecution, statementID)
			if err != nil {
				return err
			}
			return renderStatementInfo(cmd.OutOrStdout(), info)
		},
	}

	return cmd
}

// cancelStatementExecution issues CancelExecution and reports state=CANCELED on success.
// CancelExecution returns no body; the actual server-side state is verified
// asynchronously. Use 'statement status' to confirm if certainty is required.
func cancelStatementExecution(ctx context.Context, api sql.StatementExecutionInterface, statementID string) (statementInfo, error) {
	// Cancel doesn't use the warehouse ID.
	client := sqlexec.New(api, "")
	if err := client.Cancel(ctx, statementID); err != nil {
		return statementInfo{}, err
	}
	return statementInfo{
		StatementID: statementID,
		State:       sql.StatementStateCanceled,
	}, nil
}
