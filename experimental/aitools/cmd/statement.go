package aitools

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/databricks/cli/libs/sqlexec"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

// statementInfo is the JSON shape emitted by every `tools statement`
// subcommand. Fields are populated as the subcommand has them. omitempty keeps
// the output tight: `submit` doesn't emit columns/rows, `cancel` doesn't emit a
// warehouse_id, etc.
type statementInfo struct {
	StatementID string             `json:"statement_id"`
	State       sql.StatementState `json:"state,omitempty"`
	WarehouseID string             `json:"warehouse_id,omitempty"`
	Columns     []string           `json:"columns,omitempty"`
	Rows        [][]string         `json:"rows,omitempty"`
	Error       *batchResultError  `json:"error,omitempty"`
}

func renderStatementInfo(w io.Writer, info statementInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal statement info: %w", err)
	}
	fmt.Fprintf(w, "%s\n", data)
	return nil
}

// statementError converts the engine's structured statement error into the
// batchResultError shape emitted in JSON output. It returns nil for a nil error
// or any error that is not a *sqlexec.StatementError (the engine produces the
// latter only on terminal non-success states). The error's Message and Code are
// surfaced directly rather than the formatted Error() string, and the engine
// synthesizes a "statement reached terminal state <STATE>" message when the
// backend reports no ServiceError, so skill consumers can branch on
// `error == null` alone instead of inspecting `state`.
func statementError(err error) *batchResultError {
	if err == nil {
		return nil
	}
	se, ok := errors.AsType[*sqlexec.StatementError](err)
	if !ok {
		return &batchResultError{Message: err.Error()}
	}
	return &batchResultError{
		Message:   se.Message,
		ErrorCode: string(se.Code),
	}
}

func newStatementCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "statement",
		Short: "Manage SQL statement lifecycle (submit, get, status, cancel)",
		Long: `Low-level command tree for asynchronous SQL execution.

Use 'submit' to fire a statement and get its statement_id back, then
'get' to block on results, 'status' to peek without blocking, and
'cancel' to terminate. For "I want results now," use 'tools query'
instead.

All subcommands emit a JSON object with the statement_id and state.
'get' adds columns and rows on success; any subcommand may emit an
error object when the server reports a non-success terminal state.`,
	}

	cmd.AddCommand(newStatementSubmitCmd())
	cmd.AddCommand(newStatementGetCmd())
	cmd.AddCommand(newStatementStatusCmd())
	cmd.AddCommand(newStatementCancelCmd())

	return cmd
}
