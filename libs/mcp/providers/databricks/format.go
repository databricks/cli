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
		lines = append(lines, fmt.Sprintf("• %s", catalog))
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
		lines = append(lines, fmt.Sprintf("• %s", schema))
	}
	return strings.Join(lines, "\n")
}

// formatTablesResult formats the tables result into a readable string
func formatTablesResult(result *ListTablesResult) string {
	if len(result.Tables) == 0 {
		return "No tables found."
	}

	lines := []string{fmt.Sprintf("Found %d tables:", len(result.Tables)), ""}
	for _, table := range result.Tables {
		info := fmt.Sprintf("• %s (%s)", table.FullName, table.TableType)
		if table.Owner != nil {
			info += fmt.Sprintf(" - Owner: %s", *table.Owner)
		}
		if table.Comment != nil {
			info += fmt.Sprintf(" - %s", *table.Comment)
		}
		lines = append(lines, info)
	}
	return strings.Join(lines, "\n")
}

// formatTableDetails formats the table details into a readable string
func formatTableDetails(details *TableDetails) string {
	var lines []string

	lines = append(lines, fmt.Sprintf("Table: %s", details.FullName))
	lines = append(lines, fmt.Sprintf("Table Type: %s", details.TableType))

	if details.Owner != nil {
		lines = append(lines, fmt.Sprintf("Owner: %s", *details.Owner))
	}

	if details.Comment != nil {
		lines = append(lines, fmt.Sprintf("Comment: %s", *details.Comment))
	}

	if details.RowCount != nil {
		lines = append(lines, fmt.Sprintf("Row Count: %d", *details.RowCount))
	}

	if details.StorageLocation != nil {
		lines = append(lines, fmt.Sprintf("Storage Location: %s", *details.StorageLocation))
	}

	if details.DataSourceFormat != nil {
		lines = append(lines, fmt.Sprintf("Data Source Format: %s", *details.DataSourceFormat))
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
		sampleLimit := 5
		if len(details.SampleData) < sampleLimit {
			sampleLimit = len(details.SampleData)
		}
		for i := 0; i < sampleLimit; i++ {
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
func formatQueryResult(rows []map[string]any) string {
	if len(rows) == 0 {
		return "Query executed successfully but returned no results."
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Query returned %d rows:", len(rows)))
	lines = append(lines, "")

	if len(rows) > 0 {
		// Get column names from first row
		var columns []string
		for key := range rows[0] {
			columns = append(columns, key)
		}

		lines = append(lines, fmt.Sprintf("Columns: %s", strings.Join(columns, ", ")))
		lines = append(lines, "")
		lines = append(lines, "Results:")

		limit := 100
		if len(rows) < limit {
			limit = len(rows)
		}

		for i := 0; i < limit; i++ {
			row := rows[i]
			var rowParts []string
			for _, col := range columns {
				if val, ok := row[col]; ok {
					rowParts = append(rowParts, fmt.Sprintf("%s: %s", col, formatValue(val)))
				}
			}
			lines = append(lines, fmt.Sprintf("  Row %d: %s", i+1, strings.Join(rowParts, ", ")))
		}

		if len(rows) > 100 {
			lines = append(lines, fmt.Sprintf("\n... showing first 100 of %d total rows", len(rows)))
		}
	}

	return strings.Join(lines, "\n")
}
