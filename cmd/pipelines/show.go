package pipelines

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/databrickscfg/cfgpickers"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/sqlexec"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go"
	"github.com/spf13/cobra"
)

const defaultLimit = 10

func showCommand() *cobra.Command {
	var warehouseID string
	var limit int

	cmd := &cobra.Command{
		Use:   "show TABLE",
		Short: "Preview a table's columns and sample rows",
		Long: `Preview a table's columns and sample rows on a SQL warehouse.

TABLE is a fully-qualified name: catalog.schema.table (Unity Catalog) or
schema.table (legacy Hive metastore).`,
		Args:    root.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			// The CLI root doesn't cancel ctx on signals, so Ctrl-C / kill would
			// otherwise leave Execute polling the warehouse until the query ends.
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			w := cmdctx.WorkspaceClient(ctx)

			if limit <= 0 {
				return errors.New("--limit must be a positive integer")
			}

			quoted, err := quoteTableName(args[0])
			if err != nil {
				return err
			}

			warehouseID, err = resolveWarehouseID(ctx, w, warehouseID)
			if err != nil {
				return err
			}

			client := sqlexec.New(w.StatementExecution, warehouseID)
			statement := fmt.Sprintf("SELECT * FROM %s LIMIT %d", quoted, limit)
			result, err := client.Execute(ctx, statement)
			if err != nil {
				if ctx.Err() != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "\nCancelled.")
					return root.ErrAlreadyPrinted
				}
				return err
			}

			return render(ctx, cmd, result.Columns, result.Rows)
		},
	}

	cmd.Flags().StringVar(&warehouseID, "warehouse-id", "", "SQL warehouse to run the query on. Defaults to DATABRICKS_WAREHOUSE_ID or a workspace default.")
	cmd.Flags().IntVarP(&limit, "limit", "n", defaultLimit, "Maximum number of rows to fetch.")

	return cmd
}

// warehouse to query either --warehouse-id flag, DATABRICKS_WAREHOUSE_ID config,
// or the workspace default warehouse.
func resolveWarehouseID(ctx context.Context, w *databricks.WorkspaceClient, flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if w.Config.WarehouseID != "" {
		return w.Config.WarehouseID, nil
	}
	warehouse, err := cfgpickers.GetDefaultWarehouse(ctx, w)
	if errors.Is(err, cfgpickers.ErrNoCompatibleWarehouses) {
		return "", errors.New("no SQL warehouse available; pass --warehouse-id or set DATABRICKS_WAREHOUSE_ID")
	}
	if err != nil {
		return "", err
	}
	return warehouse.Id, nil
}

// writes the result as JSON (--output json) or a terminal-width-aware table.
func render(ctx context.Context, cmd *cobra.Command, columns []string, rows [][]string) error {
	out := cmd.OutOrStdout()
	switch root.OutputType(cmd) {
	case flags.OutputJSON:
		return renderJSON(out, columns, rows)
	case flags.OutputText:
		if len(columns) == 0 {
			return nil
		}
		return tableview.RenderStaticWithTruncation(ctx, out, columns, rows, tableview.DetectWidth(out))
	default:
		return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
	}
}
