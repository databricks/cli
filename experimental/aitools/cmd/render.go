package aitools

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/databricks/cli/libs/tableview"
)

// renderBatchJSON writes batch results as a JSON array. The array preserves
// input order and includes one object per submitted statement.
func renderBatchJSON(w io.Writer, results []batchResult) error {
	output, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal batch results: %w", err)
	}
	fmt.Fprintf(w, "%s\n", output)
	return nil
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

// renderCSV writes query results as CSV with column headers as the first row.
func renderCSV(w io.Writer, columns []string, rows [][]string) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = true
	if err := cw.Write(columns); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}
	for _, row := range rows {
		record := make([]string, len(columns))
		for i := range columns {
			if i < len(row) {
				record[i] = row[i]
			}
		}
		if err := cw.Write(record); err != nil {
			return fmt.Errorf("write CSV row: %w", err)
		}
	}
	cw.Flush()
	return cw.Error()
}

// renderInteractiveTable displays query results in the interactive table browser.
func renderInteractiveTable(ctx context.Context, w io.Writer, columns []string, rows [][]string) error {
	return tableview.Run(ctx, w, columns, rows)
}
