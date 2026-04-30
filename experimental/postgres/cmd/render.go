package postgrescmd

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// queryResult is the rendered shape of a single SQL execution. PR 1 only
// renders text; later PRs add JSON and CSV against the same struct.
//
// columns is empty for command-only statements (INSERT, CREATE DATABASE, ...);
// rows is empty when no rows were returned (or for command-only statements).
type queryResult struct {
	SQL string
	// CommandTag is the Postgres command tag for the statement (e.g. "INSERT 0 5",
	// "CREATE DATABASE"). Always set; used for command-only statements and as a
	// trailer for rows-producing ones.
	CommandTag string
	Columns    []string
	Rows       [][]string
}

// IsRowsProducing reports whether the statement returned a row description.
// Determined at runtime via FieldDescriptions() rather than by parsing the
// leading SQL keyword: `INSERT ... RETURNING` and CTEs ending in a SELECT are
// rows-producing despite their leading keywords.
func (r *queryResult) IsRowsProducing() bool {
	return len(r.Columns) > 0
}

// renderText writes a result in plain text.
//
// For rows-producing statements we use a tabwriter-aligned table followed by
// a `(N rows)` footer, mimicking psql's compact text shape. For command-only
// statements we just print the command tag.
//
// PR 1 always uses the static (buffered) shape. The interactive table viewer
// for >30 rows lands in a later PR alongside the multi-input output shapes.
func renderText(out io.Writer, r *queryResult) error {
	if !r.IsRowsProducing() {
		_, err := fmt.Fprintln(out, r.CommandTag)
		return err
	}

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(r.Columns, "\t"))
	fmt.Fprintln(tw, strings.Join(headerSeparator(r.Columns), "\t"))
	for _, row := range r.Rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	_, err := fmt.Fprintf(out, "(%d %s)\n", len(r.Rows), pluralize(len(r.Rows), "row", "rows"))
	return err
}

func headerSeparator(cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = strings.Repeat("-", max(len(c), 3))
	}
	return out
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
