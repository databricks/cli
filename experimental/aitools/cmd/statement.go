package aitools

import (
	"encoding/json"
	"fmt"
	"io"

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
