package databricks

import (
	"fmt"
	"strings"
)

// formatCatalogsResult formats the catalogs result into a readable string
func formatCatalogsResult(result *ListCatalogsResult) string {
	if len(result.Catalogs) == 0 {
		return "No catalogs found."
	}

	lines := []string{fmt.Sprintf("Found %d catalogs:", len(result.Catalogs)), ""}
	for _, catalog := range result.Catalogs {
		lines = append(lines, "• "+catalog)
	}
	return strings.Join(lines, "\n")
}

// formatSchemasResult formats the schemas result into a readable string
func formatSchemasResult(result *ListSchemasResult) string {
	var paginationInfo string
	if result.TotalCount > result.Limit+result.Offset {
		paginationInfo = fmt.Sprintf("Showing %d items (offset %d, limit %d). Total: %d",
			result.ShownCount, result.Offset, result.Limit, result.TotalCount)
	} else if result.Offset > 0 {
		paginationInfo = fmt.Sprintf("Showing %d items (offset %d). Total: %d",
			result.ShownCount, result.Offset, result.TotalCount)
	} else if result.TotalCount > result.Limit {
		paginationInfo = fmt.Sprintf("Showing %d items (limit %d). Total: %d",
			result.ShownCount, result.Limit, result.TotalCount)
	} else {
		paginationInfo = fmt.Sprintf("Showing all %d items", result.TotalCount)
	}

	if len(result.Schemas) == 0 {
		return paginationInfo
	}

	lines := []string{paginationInfo, ""}
	for _, schema := range result.Schemas {
		lines = append(lines, "• "+schema)
	}
	return strings.Join(lines, "\n")
}

// formatTablesResult formats the tables result into a readable string
func formatTablesResult(result *ListTablesResult) string {
	var header string
	if result.NextPageToken != nil {
		header = fmt.Sprintf("Found %d tables (more results available - use page_token to fetch next page):", result.TotalCount)
	} else {
		header = fmt.Sprintf("Found %d tables:", result.TotalCount)
	}

	if len(result.Tables) == 0 {
		if result.NextPageToken != nil {
			return header + "\n(No tables on this page, but more results may be available)"
		}
		return "No tables found."
	}

	lines := []string{header, ""}
	for _, table := range result.Tables {
		info := fmt.Sprintf("• %s (%s)", table.FullName, table.TableType)
		if table.Owner != nil {
			info += " - Owner: " + *table.Owner
		}
		if table.Comment != nil {
			info += " - " + *table.Comment
		}
		lines = append(lines, info)
	}

	if result.NextPageToken != nil {
		lines = append(lines, "")
		lines = append(lines, "Next page token: "+*result.NextPageToken)
	}

	return strings.Join(lines, "\n")
}

// formatTableDetails formats the table details into a readable string
func formatTableDetails(details *TableDetails) string {
	var lines []string

	lines = append(lines, "Table: "+details.FullName)
	lines = append(lines, "Table Type: "+details.TableType)

	if details.Owner != nil {
		lines = append(lines, "Owner: "+*details.Owner)
	}

	if details.Comment != nil {
		lines = append(lines, "Comment: "+*details.Comment)
	}

	if details.RowCount != nil {
		lines = append(lines, fmt.Sprintf("Row Count: %d", *details.RowCount))
	}

	if details.StorageLocation != nil {
		lines = append(lines, "Storage Location: "+*details.StorageLocation)
	}

	if details.DataSourceFormat != nil {
		lines = append(lines, "Data Source Format: "+*details.DataSourceFormat)
	}

	if len(details.Columns) > 0 {
		lines = append(lines, fmt.Sprintf("\nColumns (%d):", len(details.Columns)))
		for _, col := range details.Columns {
			colInfo := fmt.Sprintf("  - %s: %s", col.Name, col.DataType)
			if col.Comment != nil {
				colInfo += fmt.Sprintf(" (%s)", *col.Comment)
			}
			lines = append(lines, colInfo)
		}
	}

	if len(details.SampleData) > 0 {
		lines = append(lines, fmt.Sprintf("\nSample Data (%d rows):", len(details.SampleData)))
		sampleLimit := min(5, len(details.SampleData))
		for i := range sampleLimit {
			row := details.SampleData[i]
			var rowParts []string
			for k, v := range row {
				rowParts = append(rowParts, fmt.Sprintf("%s: %s", k, formatValue(v)))
			}
			lines = append(lines, fmt.Sprintf("  Row %d: %s", i+1, strings.Join(rowParts, ", ")))
		}
		if len(details.SampleData) > 5 {
			lines = append(lines, "...")
		}
	}

	return strings.Join(lines, "\n")
}

func formatValue(v any) string {
	if v == nil {
		return "null"
	}
	return fmt.Sprintf("%v", v)
}

// formatQueryResult formats query results into a readable string
func formatQueryResult(result *ExecuteQueryResult) string {
	var lines []string

	lines = append(lines, fmt.Sprintf("Query executed in %.2f seconds", result.ExecutionTime))
	lines = append(lines, "")

	if result.RowCount == 0 {
		lines = append(lines, "Query executed successfully but returned no results.")
		return strings.Join(lines, "\n")
	}

	if result.Truncated {
		lines = append(lines, fmt.Sprintf("Query returned %d rows (showing first %d of more results - use max_rows parameter to adjust):", result.RowCount, result.RowCount))
	} else {
		lines = append(lines, fmt.Sprintf("Query returned %d rows:", result.RowCount))
	}
	lines = append(lines, "")

	if len(result.Columns) > 0 {
		lines = append(lines, "Columns: "+strings.Join(result.Columns, ", "))
		lines = append(lines, "")
		lines = append(lines, "Results:")

		displayLimit := min(100, result.RowCount)

		for i := range displayLimit {
			row := result.Rows[i]
			var rowParts []string
			for j, val := range row {
				if j < len(result.Columns) {
					rowParts = append(rowParts, fmt.Sprintf("%s: %s", result.Columns[j], formatValue(val)))
				}
			}
			lines = append(lines, fmt.Sprintf("  Row %d: %s", i+1, strings.Join(rowParts, ", ")))
		}

		if result.RowCount > 100 {
			lines = append(lines, fmt.Sprintf("\n... showing first 100 of %d returned rows", result.RowCount))
		}
	}

	return strings.Join(lines, "\n")
}
