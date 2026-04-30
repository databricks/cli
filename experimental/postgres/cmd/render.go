package postgrescmd

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/jackc/pgx/v5/pgconn"
)

// textSink renders results as plain text: a tabwriter-aligned table for
// rows-producing statements, the command tag for command-only ones.
//
// Text output buffers all rows because tabwriter needs the widest cell in each
// column before it can align. Streaming output is provided by the JSON and CSV
// sinks; users with huge result sets should pick those.
type textSink struct {
	out     io.Writer
	columns []string
	rows    [][]string
}

func newTextSink(out io.Writer) *textSink {
	return &textSink{out: out}
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

// OnError for text sinks is a no-op: text output prints whatever rows have
// already been collected, with no open structure to close. The caller
// surfaces the error separately (cobra's default error rendering).
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
