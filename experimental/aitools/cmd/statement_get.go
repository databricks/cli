package aitools

import (
	"context"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

func newStatementGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get STATEMENT_ID",
		Short: "Block until a previously submitted statement is terminal and emit its result",
		Long: `Poll a statement_id until it reaches a terminal state, then emit
columns and rows on success or an error object on failure.

Ctrl+C stops polling but does NOT cancel the server-side statement.
Use 'statement cancel <id>' to terminate explicitly. (This differs from
'tools query', which cancels server-side on Ctrl+C because the user
invoked the synchronous path.)`,
		Example: `  databricks experimental aitools tools statement get 01ef...`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			statementID := args[0]

			info, err := getStatementResult(ctx, w.StatementExecution, statementID)
			if err != nil {
				return err
			}

			if err := renderStatementInfo(cmd.OutOrStdout(), info); err != nil {
				return err
			}

			// Non-zero exit when the statement reached a non-success terminal
			// state OR a chunk-fetch failure prevented assembling the rows.
			// In both cases the failure detail is already in the JSON output.
			if info.State != sql.StatementStateSucceeded || info.Error != nil {
				return root.ErrAlreadyPrinted
			}
			return nil
		},
	}

	return cmd
}

// getStatementResult polls a statement until terminal, then assembles a
// statementInfo with rows on success or an error object on failure.
//
// Context cancellation propagates from pollStatement WITHOUT cancelling the
// server-side statement (intentional: 'get' is a poll-only operation; use
// 'cancel' to terminate explicitly).
func getStatementResult(ctx context.Context, api sql.StatementExecutionInterface, statementID string) (statementInfo, error) {
	// Fetch the current state first so pollStatement can short-circuit if
	// the statement is already terminal.
	resp, err := api.GetStatementByStatementId(ctx, statementID)
	if err != nil {
		return statementInfo{}, fmt.Errorf("get statement: %w", err)
	}

	pollResp, err := pollStatement(ctx, api, resp)
	if err != nil {
		return statementInfo{}, err
	}

	info := statementInfo{StatementID: pollResp.StatementId}
	if pollResp.Status != nil {
		info.State = pollResp.Status.State
	}
	info.Error = statementErrorFromStatus(pollResp.Status)

	if info.State == sql.StatementStateSucceeded {
		info.Columns = extractColumns(pollResp.Manifest)
		rows, err := fetchAllRows(ctx, api, pollResp)
		if err != nil {
			// The query succeeded server-side but a later chunk fetch failed
			// (network blip, throttling, transient 5xx). Surface this as a
			// structured error on the same statementInfo so the caller still
			// gets a parseable JSON response with the statement_id; RunE then
			// signals exit-non-zero based on info.Error.
			info.Error = &batchResultError{
				Message: fmt.Sprintf("fetch result rows: %v", err),
			}
			return info, nil
		}
		info.Rows = rows
	}
	return info, nil
}
