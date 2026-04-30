package postgrescmd

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/databricks/cli/libs/tableview"
	"github.com/jackc/pgx/v5/pgconn"
)

// staticTableThreshold is the row count above which we hand off to
// libs/tableview's interactive viewer (when stdout is interactive). Smaller
// results stay in the static tabwriter path so they stream to a pipe
// unchanged. Matches the threshold aitools query uses.
const staticTableThreshold = 30

// textSink renders results as plain text: a tabwriter-aligned table for
// rows-producing statements, the command tag for command-only ones.
//
// Text output buffers all rows because tabwriter needs the widest cell in each
// column before it can align. Streaming output is provided by the JSON and CSV
// sinks; users with huge result sets should pick those.
//
// When interactive is true and the result has more than staticTableThreshold
// rows, End hands off to libs/tableview's scrollable viewer instead of
// emitting the static table. The interactive path requires a real TTY and a
// prompt-capable terminal; the caller decides.
type textSink struct {
	out         io.Writer
	interactive bool
	columns     []string
	rows        [][]string
}

func newTextSink(out io.Writer) *textSink {
	return &textSink{out: out}
}

// newInteractiveTextSink returns a text sink that uses the interactive table
// viewer for results larger than staticTableThreshold.
func newInteractiveTextSink(out io.Writer) *textSink {
	return &textSink{out: out, interactive: true}
}

func (s *textSink) Begin(fields []pgconn.FieldDescription) error {
	s.columns = make([]string, len(fields))
	for i, f := range fields {
		s.columns[i] = f.Name
	}
	return nil
}

func (s *textSink) Row(values []any) error {
	row := make([]string, len(values))
	for i, v := range values {
		row[i] = textValue(v)
	}
	s.rows = append(s.rows, row)
	return nil
}

func (s *textSink) End(commandTag string) error {
	if len(s.columns) == 0 {
		_, err := fmt.Fprintln(s.out, commandTag)
		return err
	}

	if s.interactive && len(s.rows) > staticTableThreshold {
		return tableview.Run(s.out, s.columns, s.rows)
	}

	tw := tabwriter.NewWriter(s.out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(s.columns, "\t"))
	fmt.Fprintln(tw, strings.Join(headerSeparator(s.columns), "\t"))
	for _, row := range s.rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	_, err := fmt.Fprintf(s.out, "(%d %s)\n", len(s.rows), pluralize(len(s.rows), "row", "rows"))
	return err
}

// OnError for text sinks is a no-op. Text mode buffers all rows for
// tabwriter alignment, so a partial result is discarded on iteration error;
// only cobra's error message reaches the user. The streaming sinks (json,
// csv) handle the partial-result case themselves.
func (s *textSink) OnError(err error) {}

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
