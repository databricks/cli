package databricks

import (
	"strings"
	"testing"
)

func TestFormatCatalogsResult(t *testing.T) {
	tests := []struct {
		name   string
		result *ListCatalogsResult
		want   []string
	}{
		{
			name: "empty catalogs",
			result: &ListCatalogsResult{
				Catalogs: []string{},
			},
			want: []string{"No catalogs found"},
		},
		{
			name: "single catalog",
			result: &ListCatalogsResult{
				Catalogs: []string{"main"},
			},
			want: []string{"Found 1 catalogs", "main"},
		},
		{
			name: "multiple catalogs",
			result: &ListCatalogsResult{
				Catalogs: []string{"main", "dev", "prod"},
			},
			want: []string{"Found 3 catalogs", "main", "dev", "prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCatalogsResult(tt.result)
			for _, substr := range tt.want {
				if !strings.Contains(got, substr) {
					t.Errorf("formatCatalogsResult() missing expected substring %q in output:\n%s", substr, got)
				}
			}
		})
	}
}

func TestFormatSchemasResult(t *testing.T) {
	tests := []struct {
		name   string
		result *ListSchemasResult
		want   []string
	}{
		{
			name: "empty schemas",
			result: &ListSchemasResult{
				Schemas:    []string{},
				TotalCount: 0,
				ShownCount: 0,
				Offset:     0,
				Limit:      1000,
			},
			want: []string{"Showing all 0 items"},
		},
		{
			name: "paginated schemas",
			result: &ListSchemasResult{
				Schemas:    []string{"schema1", "schema2"},
				TotalCount: 10,
				ShownCount: 2,
				Offset:     0,
				Limit:      2,
			},
			want: []string{"Showing 2 items", "Total: 10", "schema1", "schema2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSchemasResult(tt.result)
			for _, substr := range tt.want {
				if !strings.Contains(got, substr) {
					t.Errorf("formatSchemasResult() missing expected substring %q in output:\n%s", substr, got)
				}
			}
		})
	}
}

func TestFormatTablesResult(t *testing.T) {
	comment := "Test table"
	tests := []struct {
		name   string
		result *ListTablesResult
		want   []string
	}{
		{
			name: "empty tables",
			result: &ListTablesResult{
				Tables: []TableInfo{},
			},
			want: []string{"No tables found"},
		},
		{
			name: "table with comment",
			result: &ListTablesResult{
				Tables: []TableInfo{
					{
						Name:        "users",
						CatalogName: "main",
						SchemaName:  "default",
						FullName:    "main.default.users",
						TableType:   "MANAGED",
						Comment:     &comment,
					},
				},
			},
			want: []string{"Found 1 tables", "main.default.users", "MANAGED", "Test table"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTablesResult(tt.result)
			for _, substr := range tt.want {
				if !strings.Contains(got, substr) {
					t.Errorf("formatTablesResult() missing expected substring %q in output:\n%s", substr, got)
				}
			}
		})
	}
}

func TestFormatTableDetails(t *testing.T) {
	owner := "alice@example.com"
	comment := "User data table"
	storageLocation := "s3://bucket/path"
	dataSourceFormat := "DELTA"
	rowCount := int64(1000)

	tests := []struct {
		name    string
		details *TableDetails
		want    []string
	}{
		{
			name: "minimal table details",
			details: &TableDetails{
				FullName:  "main.default.users",
				TableType: "MANAGED",
				Columns: []ColumnMetadata{
					{Name: "id", DataType: "BIGINT"},
					{Name: "name", DataType: "STRING"},
				},
			},
			want: []string{"Table: main.default.users", "Type: MANAGED", "Columns (2)", "id: BIGINT", "name: STRING"},
		},
		{
			name: "full table details",
			details: &TableDetails{
				FullName:         "main.default.users",
				TableType:        "MANAGED",
				Owner:            &owner,
				Comment:          &comment,
				StorageLocation:  &storageLocation,
				DataSourceFormat: &dataSourceFormat,
				RowCount:         &rowCount,
				Columns: []ColumnMetadata{
					{Name: "id", DataType: "BIGINT"},
					{Name: "name", DataType: "STRING"},
				},
			},
			want: []string{
				"Table: main.default.users",
				"Type: MANAGED",
				"Owner: alice@example.com",
				"Comment: User data table",
				"Storage Location: s3://bucket/path",
				"Data Source Format: DELTA",
				"Row Count: 1000",
				"Columns (2)",
				"id: BIGINT",
				"name: STRING",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTableDetails(tt.details)
			for _, substr := range tt.want {
				if !strings.Contains(got, substr) {
					t.Errorf("formatTableDetails() missing expected substring %q in output:\n%s", substr, got)
				}
			}
		})
	}
}

func TestFormatQueryResult(t *testing.T) {
	tests := []struct {
		name string
		rows []map[string]any
		want []string
	}{
		{
			name: "empty result",
			rows: []map[string]any{},
			want: []string{"Query executed successfully but returned no results"},
		},
		{
			name: "single row",
			rows: []map[string]any{
				{"id": 1, "name": "Alice"},
			},
			want: []string{"Query returned 1 rows", "id", "name", "Alice"},
		},
		{
			name: "multiple rows",
			rows: []map[string]any{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			},
			want: []string{"Query returned 2 rows", "id", "name", "Alice", "Bob"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatQueryResult(tt.rows)
			for _, substr := range tt.want {
				if !strings.Contains(got, substr) {
					t.Errorf("formatQueryResult() missing expected substring %q in output:\n%s", substr, got)
				}
			}
		})
	}
}
