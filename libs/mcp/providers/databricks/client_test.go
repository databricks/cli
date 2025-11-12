package databricks

import (
	"context"
	"os"
	"testing"

	"github.com/databricks/cli/libs/mcp"
)

// TestIntegration_RealWorkspace runs integration tests against a real Databricks workspace
// This test is skipped unless DATABRICKS_WAREHOUSE_ID is set
func TestIntegration_RealWorkspace(t *testing.T) {
	if os.Getenv("DATABRICKS_WAREHOUSE_ID") == "" {
		t.Skip("Skipping integration test: DATABRICKS_WAREHOUSE_ID not set")
	}

	// Create config
	cfg := &mcp.Config{
		WarehouseID: os.Getenv("DATABRICKS_WAREHOUSE_ID"),
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create client
	client, err := NewClient(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Test ListCatalogs
	t.Run("ListCatalogs", func(t *testing.T) {
		result, err := client.ListCatalogs(ctx)
		if err != nil {
			t.Fatalf("ListCatalogs() error = %v", err)
		}

		if len(result.Catalogs) == 0 {
			t.Log("Warning: No catalogs found. This might be expected in some workspaces.")
		}

		t.Logf("Found %d catalog(s)", len(result.Catalogs))
		for _, catalog := range result.Catalogs {
			t.Logf("  - %s", catalog)
		}
	})

	// Test ListSchemas (using first available catalog)
	t.Run("ListSchemas", func(t *testing.T) {
		catalogs, err := client.ListCatalogs(ctx)
		if err != nil {
			t.Fatalf("ListCatalogs() error = %v", err)
		}

		if len(catalogs.Catalogs) == 0 {
			t.Skip("No catalogs available to test ListSchemas")
		}

		catalogName := catalogs.Catalogs[0]
		result, err := client.ListSchemas(ctx, &ListSchemasArgs{
			CatalogName: catalogName,
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ListSchemas() error = %v", err)
		}

		t.Logf("Found %d schema(s) in catalog %s", result.TotalCount, catalogName)
		for _, schema := range result.Schemas {
			t.Logf("  - %s", schema)
		}
	})

	// Test ListTables (using first available catalog and schema)
	t.Run("ListTables", func(t *testing.T) {
		catalogs, err := client.ListCatalogs(ctx)
		if err != nil {
			t.Fatalf("ListCatalogs() error = %v", err)
		}

		if len(catalogs.Catalogs) == 0 {
			t.Skip("No catalogs available to test ListTables")
		}

		catalogName := catalogs.Catalogs[0]
		schemas, err := client.ListSchemas(ctx, &ListSchemasArgs{
			CatalogName: catalogName,
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ListSchemas() error = %v", err)
		}

		if len(schemas.Schemas) == 0 {
			t.Skip("No schemas available to test ListTables")
		}

		schemaName := schemas.Schemas[0]
		result, err := client.ListTables(ctx, &ListTablesArgs{
			CatalogName: catalogName,
			SchemaName:  schemaName,
		})
		if err != nil {
			t.Fatalf("ListTables() error = %v", err)
		}

		t.Logf("Found %d table(s) in %s.%s", len(result.Tables), catalogName, schemaName)
		for _, table := range result.Tables {
			t.Logf("  - %s (%s)", table.FullName, table.TableType)
		}
	})

	// Test DescribeTable (using first available table)
	t.Run("DescribeTable", func(t *testing.T) {
		catalogs, err := client.ListCatalogs(ctx)
		if err != nil {
			t.Fatalf("ListCatalogs() error = %v", err)
		}

		if len(catalogs.Catalogs) == 0 {
			t.Skip("No catalogs available to test DescribeTable")
		}

		catalogName := catalogs.Catalogs[0]
		schemas, err := client.ListSchemas(ctx, &ListSchemasArgs{
			CatalogName: catalogName,
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ListSchemas() error = %v", err)
		}

		if len(schemas.Schemas) == 0 {
			t.Skip("No schemas available to test DescribeTable")
		}

		schemaName := schemas.Schemas[0]
		tables, err := client.ListTables(ctx, &ListTablesArgs{
			CatalogName: catalogName,
			SchemaName:  schemaName,
		})
		if err != nil {
			t.Fatalf("ListTables() error = %v", err)
		}

		if len(tables.Tables) == 0 {
			t.Skip("No tables available to test DescribeTable")
		}

		tableFullName := tables.Tables[0].FullName
		result, err := client.DescribeTable(ctx, &DescribeTableArgs{
			TableFullName: tableFullName,
			SampleSize:    3,
		})
		if err != nil {
			t.Fatalf("DescribeTable() error = %v", err)
		}

		t.Logf("Table: %s", result.FullName)
		t.Logf("Type: %s", result.TableType)
		t.Logf("Columns: %d", len(result.Columns))
		for _, col := range result.Columns {
			t.Logf("  - %s: %s", col.Name, col.DataType)
		}

		if result.RowCount != nil {
			t.Logf("Row Count: %d", *result.RowCount)
		}

		if len(result.SampleData) > 0 {
			t.Logf("Sample Data: %d rows", len(result.SampleData))
		}
	})

	// Test ExecuteQuery
	t.Run("ExecuteQuery", func(t *testing.T) {
		// Try a simple query that should work in any workspace
		query := "SELECT 1 as test_column, 'hello' as message"
		result, err := client.ExecuteQuery(ctx, query)
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		if len(result) != 1 {
			t.Errorf("Expected 1 row, got %d", len(result))
		}

		if len(result) > 0 {
			t.Logf("Query result: %v", result[0])
			if val, ok := result[0]["test_column"]; !ok {
				t.Errorf("Expected 'test_column' in result, got %v", result[0])
			} else {
				t.Logf("test_column = %v", val)
			}
		}
	})
}
