package mcp

import (
	"fmt"
	"os"

	"github.com/databricks/cli/cmd/root"
	mcplib "github.com/databricks/cli/experimental/apps-mcp/lib"
	"github.com/databricks/cli/experimental/apps-mcp/lib/providers/databricks"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

func newToolFindTablesCmd() *cobra.Command {
	var catalogName string
	var schemaName string
	var filter string
	var limit int
	var offset int

	cmd := &cobra.Command{
		Use:   "tool-find-tables",
		Short: "Find tables in Databricks Unity Catalog",
		Long: `Find tables in Databricks Unity Catalog. Supports searching within a specific catalog and schema,
across all schemas in a catalog, or across all catalogs. Supports wildcard patterns (* for multiple
characters, ? for single character) in table name and schema name filtering.`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			warehouseID := os.Getenv("DATABRICKS_WAREHOUSE_ID")
			if warehouseID == "" {
				return fmt.Errorf("DATABRICKS_WAREHOUSE_ID environment variable is required")
			}

			cfg := &mcplib.Config{
				WarehouseID:    warehouseID,
				DatabricksHost: w.Config.Host,
			}

			client, err := databricks.NewDatabricksRestClient(ctx, cfg)
			if err != nil {
				return err
			}

			// Build request using FindTablesInput structure
			var catalogPtr *string
			if catalogName != "" {
				catalogPtr = &catalogName
			}

			var schemaPtr *string
			if schemaName != "" {
				schemaPtr = &schemaName
			}

			var filterPtr *string
			if filter != "" {
				filterPtr = &filter
			}

			// Set default limit
			if limit == 0 {
				limit = 1000
			}

			request := &databricks.ListTablesRequest{
				CatalogName: catalogPtr,
				SchemaName:  schemaPtr,
				Filter:      filterPtr,
				Limit:       limit,
				Offset:      offset,
			}

			result, err := client.ListTables(ctx, request)
			if err != nil {
				return err
			}

			fmt.Println(result.Display())
			return nil
		},
	}

	cmd.Flags().StringVar(&catalogName, "catalog-name", "", "Name of the catalog (optional - searches all catalogs if not provided)")
	cmd.Flags().StringVar(&schemaName, "schema-name", "", "Name of the schema (optional - searches all schemas if not provided)")
	cmd.Flags().StringVar(&filter, "filter", "", "Filter pattern for table names (supports * and ? wildcards)")
	cmd.Flags().IntVar(&limit, "limit", 1000, "Maximum number of tables to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "Offset for pagination")

	return cmd
}
