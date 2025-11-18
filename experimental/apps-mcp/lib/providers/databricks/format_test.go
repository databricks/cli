package databricks

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatCatalogsResult(t *testing.T) {
	t.Run("EmptyCatalogs", func(t *testing.T) {
		result := &ListCatalogsResult{
			Catalogs: []string{},
		}
		output := formatCatalogsResult(result)
		assert.Equal(t, "No catalogs found.", output)
	})

	t.Run("SingleCatalog", func(t *testing.T) {
		result := &ListCatalogsResult{
			Catalogs: []string{"main"},
		}
		output := formatCatalogsResult(result)
		assert.Contains(t, output, "Found 1 catalogs:")
		assert.Contains(t, output, "• main")
	})

	t.Run("MultipleCatalogs", func(t *testing.T) {
		result := &ListCatalogsResult{
			Catalogs: []string{"main", "dev", "prod"},
		}
		output := formatCatalogsResult(result)
		assert.Contains(t, output, "Found 3 catalogs:")
		assert.Contains(t, output, "• main")
		assert.Contains(t, output, "• dev")
		assert.Contains(t, output, "• prod")
	})
}

func TestFormatSchemasResult(t *testing.T) {
	t.Run("EmptySchemas", func(t *testing.T) {
		result := &ListSchemasResult{
			Schemas:    []string{},
			TotalCount: 0,
			ShownCount: 0,
			Offset:     0,
			Limit:      100,
		}
		output := formatSchemasResult(result)
		assert.Contains(t, output, "Showing all 0 items")
	})

	t.Run("SingleSchema", func(t *testing.T) {
		result := &ListSchemasResult{
			Schemas:    []string{"default"},
			TotalCount: 1,
			ShownCount: 1,
			Offset:     0,
			Limit:      100,
		}
		output := formatSchemasResult(result)
		assert.Contains(t, output, "Showing all 1 items")
		assert.Contains(t, output, "• default")
	})

	t.Run("WithPagination", func(t *testing.T) {
		result := &ListSchemasResult{
			Schemas:    []string{"schema1", "schema2"},
			TotalCount: 10,
			ShownCount: 2,
			Offset:     0,
			Limit:      2,
		}
		output := formatSchemasResult(result)
		// When TotalCount (10) > Limit + Offset (2 + 0), it shows offset, limit
		assert.Contains(t, output, "Showing 2 items (offset 0, limit 2). Total: 10")
		assert.Contains(t, output, "• schema1")
		assert.Contains(t, output, "• schema2")
	})

	t.Run("WithOffset", func(t *testing.T) {
		result := &ListSchemasResult{
			Schemas:    []string{"schema3", "schema4"},
			TotalCount: 10,
			ShownCount: 2,
			Offset:     2,
			Limit:      2,
		}
		output := formatSchemasResult(result)
		assert.Contains(t, output, "Showing 2 items (offset 2, limit 2). Total: 10")
	})
}

func TestFormatTablesResult(t *testing.T) {
	t.Run("EmptyTables", func(t *testing.T) {
		result := &ListTablesResult{
			Tables:        []TableInfo{},
			NextPageToken: nil,
			TotalCount:    0,
		}
		output := formatTablesResult(result)
		assert.Equal(t, "No tables found.", output)
	})

	t.Run("SingleTable", func(t *testing.T) {
		owner := "user@example.com"
		comment := "Test table"
		result := &ListTablesResult{
			Tables: []TableInfo{
				{
					Name:        "table1",
					CatalogName: "main",
					SchemaName:  "default",
					FullName:    "main.default.table1",
					TableType:   "MANAGED",
					Owner:       &owner,
					Comment:     &comment,
				},
			},
			TotalCount: 1,
		}
		output := formatTablesResult(result)
		assert.Contains(t, output, "Found 1 tables:")
		assert.Contains(t, output, "• main.default.table1 (MANAGED)")
		assert.Contains(t, output, "Owner: user@example.com")
		assert.Contains(t, output, "Test table")
	})

	t.Run("WithNextPageToken", func(t *testing.T) {
		token := "next_token_123"
		result := &ListTablesResult{
			Tables: []TableInfo{
				{
					Name:        "table1",
					CatalogName: "main",
					SchemaName:  "default",
					FullName:    "main.default.table1",
					TableType:   "MANAGED",
				},
			},
			NextPageToken: &token,
			TotalCount:    1,
		}
		output := formatTablesResult(result)
		assert.Contains(t, output, "Found 1 tables (more results available")
		assert.Contains(t, output, "Next page token: next_token_123")
	})

	t.Run("WithoutOwnerAndComment", func(t *testing.T) {
		result := &ListTablesResult{
			Tables: []TableInfo{
				{
					Name:        "table1",
					CatalogName: "main",
					SchemaName:  "default",
					FullName:    "main.default.table1",
					TableType:   "EXTERNAL",
					Owner:       nil,
					Comment:     nil,
				},
			},
			TotalCount: 1,
		}
		output := formatTablesResult(result)
		assert.Contains(t, output, "• main.default.table1 (EXTERNAL)")
		assert.NotContains(t, output, "Owner:")
	})
}

func TestFormatTableDetails(t *testing.T) {
	t.Run("MinimalTableDetails", func(t *testing.T) {
		details := &TableDetails{
			FullName:  "main.default.table1",
			TableType: "MANAGED",
			Columns:   []ColumnMetadata{},
		}
		output := formatTableDetails(details)
		assert.Contains(t, output, "Table: main.default.table1")
		assert.Contains(t, output, "Table Type: MANAGED")
	})

	t.Run("FullTableDetails", func(t *testing.T) {
		owner := "user@example.com"
		comment := "Test table"
		rowCount := int64(1000)
		storageLocation := "/path/to/storage"
		dataSourceFormat := "DELTA"
		colComment := "ID column"

		details := &TableDetails{
			FullName:         "main.default.table1",
			TableType:        "MANAGED",
			Owner:            &owner,
			Comment:          &comment,
			RowCount:         &rowCount,
			StorageLocation:  &storageLocation,
			DataSourceFormat: &dataSourceFormat,
			Columns: []ColumnMetadata{
				{
					Name:     "id",
					DataType: "bigint",
					Comment:  &colComment,
				},
				{
					Name:     "name",
					DataType: "string",
					Comment:  nil,
				},
			},
			SampleData: []map[string]any{
				{"id": int64(1), "name": "Alice"},
				{"id": int64(2), "name": "Bob"},
			},
		}
		output := formatTableDetails(details)
		assert.Contains(t, output, "Table: main.default.table1")
		assert.Contains(t, output, "Owner: user@example.com")
		assert.Contains(t, output, "Comment: Test table")
		assert.Contains(t, output, "Row Count: 1000")
		assert.Contains(t, output, "Storage Location: /path/to/storage")
		assert.Contains(t, output, "Data Source Format: DELTA")
		assert.Contains(t, output, "Columns (2):")
		assert.Contains(t, output, "id: bigint (ID column)")
		assert.Contains(t, output, "name: string")
		assert.Contains(t, output, "Sample Data (2 rows):")
		assert.Contains(t, output, "Row 1:")
		assert.Contains(t, output, "Row 2:")
	})

	t.Run("WithManySampleRows", func(t *testing.T) {
		details := &TableDetails{
			FullName:  "main.default.table1",
			TableType: "MANAGED",
			Columns:   []ColumnMetadata{},
			SampleData: []map[string]any{
				{"id": int64(1)},
				{"id": int64(2)},
				{"id": int64(3)},
				{"id": int64(4)},
				{"id": int64(5)},
				{"id": int64(6)},
				{"id": int64(7)},
			},
		}
		output := formatTableDetails(details)
		assert.Contains(t, output, "Sample Data (7 rows):")
		assert.Contains(t, output, "Row 5:")
		assert.Contains(t, output, "...")
		// Should only show first 5 rows plus ellipsis
		lines := strings.Split(output, "\n")
		sampleLines := 0
		for _, line := range lines {
			if strings.Contains(line, "Row") {
				sampleLines++
			}
		}
		assert.Equal(t, 5, sampleLines)
	})
}

func TestFormatValue(t *testing.T) {
	t.Run("NilValue", func(t *testing.T) {
		assert.Equal(t, "null", formatValue(nil))
	})

	t.Run("StringValue", func(t *testing.T) {
		assert.Equal(t, "hello", formatValue("hello"))
	})

	t.Run("IntValue", func(t *testing.T) {
		assert.Equal(t, "42", formatValue(42))
	})

	t.Run("BoolValue", func(t *testing.T) {
		assert.Equal(t, "true", formatValue(true))
	})
}

func TestFormatQueryResult(t *testing.T) {
	t.Run("EmptyResult", func(t *testing.T) {
		result := &ExecuteQueryResult{
			Columns:       []string{},
			Rows:          [][]any{},
			RowCount:      0,
			Truncated:     false,
			ExecutionTime: 0.5,
		}
		output := formatQueryResult(result)
		assert.Contains(t, output, "Query executed in 0.50 seconds")
		assert.Contains(t, output, "Query executed successfully but returned no results.")
	})

	t.Run("SimpleResult", func(t *testing.T) {
		result := &ExecuteQueryResult{
			Columns: []string{"id", "name"},
			Rows: [][]any{
				{int64(1), "Alice"},
				{int64(2), "Bob"},
			},
			RowCount:      2,
			Truncated:     false,
			ExecutionTime: 1.23,
		}
		output := formatQueryResult(result)
		assert.Contains(t, output, "Query executed in 1.23 seconds")
		assert.Contains(t, output, "Query returned 2 rows:")
		assert.Contains(t, output, "Columns: id, name")
		assert.Contains(t, output, "Row 1: id: 1, name: Alice")
		assert.Contains(t, output, "Row 2: id: 2, name: Bob")
	})

	t.Run("TruncatedResult", func(t *testing.T) {
		result := &ExecuteQueryResult{
			Columns: []string{"id"},
			Rows: [][]any{
				{int64(1)},
			},
			RowCount:      1,
			Truncated:     true,
			ExecutionTime: 2.0,
		}
		output := formatQueryResult(result)
		assert.Contains(t, output, "Query executed in 2.00 seconds")
		assert.Contains(t, output, "showing first 1 of more results")
		assert.Contains(t, output, "use max_rows parameter to adjust")
	})

	t.Run("LargeResult", func(t *testing.T) {
		// Create 150 rows
		rows := make([][]any, 150)
		for i := range rows {
			rows[i] = []any{int64(i)}
		}

		result := &ExecuteQueryResult{
			Columns:       []string{"id"},
			Rows:          rows,
			RowCount:      150,
			Truncated:     false,
			ExecutionTime: 3.5,
		}
		output := formatQueryResult(result)
		assert.Contains(t, output, "Query executed in 3.50 seconds")
		assert.Contains(t, output, "Query returned 150 rows:")
		assert.Contains(t, output, "Row 100:")
		assert.Contains(t, output, "showing first 100 of 150 returned rows")
		assert.NotContains(t, output, "Row 101:")
	})

	t.Run("WithNullValues", func(t *testing.T) {
		result := &ExecuteQueryResult{
			Columns: []string{"id", "value"},
			Rows: [][]any{
				{int64(1), nil},
				{int64(2), "test"},
			},
			RowCount:      2,
			Truncated:     false,
			ExecutionTime: 0.1,
		}
		output := formatQueryResult(result)
		assert.Contains(t, output, "value: null")
		assert.Contains(t, output, "value: test")
	})
}
