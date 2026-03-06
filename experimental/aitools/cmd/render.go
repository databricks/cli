package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

const (
	// maxColumnWidth is the maximum display width for any single column in static table output.
	maxColumnWidth = 40
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
func renderJSON(ctx context.Context, w io.Writer, columns []string, rows [][]string) error {
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
	cmdio.LogString(ctx, fmt.Sprintf("\nRow count: %d", len(rows)))
	return nil
}

// renderStaticTable writes query results as a formatted text table.
func renderStaticTable(w io.Writer, columns []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	// Header row.
	fmt.Fprintln(tw, strings.Join(columns, "\t"))

	// Separator.
	seps := make([]string, len(columns))
	for i, col := range columns {
		width := len(col)
		for _, row := range rows {
			if i < len(row) {
				width = max(width, len(row[i]))
			}
		}
		width = min(width, maxColumnWidth)
		seps[i] = strings.Repeat("-", width)
	}
	fmt.Fprintln(tw, strings.Join(seps, "\t"))

	// Data rows.
	for _, row := range rows {
		vals := make([]string, len(columns))
		for i := range columns {
			if i < len(row) {
				vals[i] = row[i]
			}
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintf(w, "\n%d rows\n", len(rows))
	return nil
}

// renderInteractiveTable displays query results in the interactive table browser.
func renderInteractiveTable(w io.Writer, columns []string, rows [][]string) error {
	return tableview.Run(w, columns, rows)
}
