package mcp

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

// extractColumns returns column names from the query result manifest.
func extractColumns(manifest *sql.ResultManifest) []string {
	if manifest == nil || manifest.Schema == nil {
		return nil
	}
	columns := make([]string, len(manifest.Schema.Columns))
	for i, col := range manifest.Schema.Columns {
		columns[i] = col.Name
	}
	return columns
}

// renderJSON writes query results as a parseable JSON array to stdout.
// Row count is written to stderr so stdout remains valid JSON for piping.
func renderJSON(w io.Writer, columns []string, rows [][]string) error {
	objects := make([]map[string]any, len(rows))
	for i, row := range rows {
		obj := make(map[string]any, len(columns))
		for j, val := range row {
			if j < len(columns) {
				obj[columns[j]] = val
			}
		}
		objects[i] = obj
	}

	output, err := json.MarshalIndent(objects, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	fmt.Fprintf(w, "%s\n", output)
	return nil
}

// renderStaticTable writes query results as a formatted text table.
func renderStaticTable(w io.Writer, columns []string, rows [][]string) error {
	return tableview.RenderStaticTable(w, columns, rows)
}

// renderInteractiveTable displays query results in the interactive table browser.
func renderInteractiveTable(w io.Writer, columns []string, rows [][]string) error {
	return tableview.Run(w, columns, rows)
}
