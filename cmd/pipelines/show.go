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
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sqlexec"
	"github.com/databricks/cli/libs/utils"
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

			checkForMoreRows := root.OutputType(cmd) == flags.OutputText && !cmd.Flags().Changed("limit")
			queryLimit := limit
			if checkForMoreRows {
				queryLimit++
			}

			client := sqlexec.New(w.StatementExecution, warehouseID)
			statement := fmt.Sprintf("SELECT * FROM %s LIMIT %d", quoted, queryLimit)
			result, err := client.Execute(ctx, statement)
			if err != nil {
				if ctx.Err() != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "\nCancelled.")
					return root.ErrAlreadyPrinted
				}
				return err
			}

			rows := result.Rows
			hasMoreRows := checkForMoreRows && len(rows) > limit
			if hasMoreRows {
				rows = rows[:limit]
			}

			if err := render(ctx, cmd, result.Columns, rows); err != nil {
				return err
			}
			if hasMoreRows {
				fmt.Fprintln(cmd.OutOrStdout(), "Use --limit <n> to preview more rows.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&warehouseID, "warehouse-id", "", "SQL warehouse to run the query on. Defaults to DATABRICKS_WAREHOUSE_ID or a workspace default.")
	cmd.Flags().IntVarP(&limit, "limit", "n", defaultLimit, "Maximum number of rows to fetch.")

	return cmd
}

// warehouse to query either --warehouse-id flag, DATABRICKS_WAREHOUSE_ID config,
// or the workspace default warehouse.
func resolveWarehouseID(ctx context.Context, w *sdkclients.Clients, flag string) (string, error) {
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
	return utils.Deref(warehouse.Id), nil
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
		return renderTable(ctx, columns, rows, detectWidth(out))
	default:
		return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
	}
}
