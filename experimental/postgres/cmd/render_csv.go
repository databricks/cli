package postgrescmd

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5/pgconn"
)

// csvSink streams query results as CSV. Header row is written on Begin, each
// data row is written and flushed individually so large exports do not buffer
// in memory.
//
// For command-only statements CSV has nothing meaningful to emit (no header,
// no rows): we write the command tag to stderr so machine consumers reading
// stdout still receive an empty document, while humans get a confirmation.
type csvSink struct {
	out    io.Writer
	stderr io.Writer
	w      *csv.Writer

	// rowsProducing is true once Begin saw a non-empty fields slice. End
	// uses it to decide whether to write the command-tag stderr line.
	rowsProducing bool
}

func newCSVSink(out, stderr io.Writer) *csvSink {
	return &csvSink{out: out, stderr: stderr, w: csv.NewWriter(out)}
}

func (s *csvSink) Begin(fields []pgconn.FieldDescription) error {
	if len(fields) == 0 {
		return nil
	}
	s.rowsProducing = true

	header := make([]string, len(fields))
	for i, f := range fields {
		header[i] = f.Name
	}
	if err := s.w.Write(header); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}
	s.w.Flush()
	return s.w.Error()
}

func (s *csvSink) Row(values []any) error {
	row := make([]string, len(values))
	for i, v := range values {
		// CSV represents NULL as an empty field, matching `psql --csv`.
		if v == nil {
			row[i] = ""
			continue
		}
		row[i] = textValue(v)
	}
	if err := s.w.Write(row); err != nil {
		return fmt.Errorf("write CSV row: %w", err)
	}
	s.w.Flush()
	return s.w.Error()
}

func (s *csvSink) End(commandTag string) error {
	if !s.rowsProducing {
		_, err := fmt.Fprintln(s.stderr, commandTag)
		return err
	}
	s.w.Flush()
	return s.w.Error()
}

// OnError flushes whatever is buffered in the csv.Writer so the partial result
// is visible to the consumer. csv.Writer has no concept of "open structure",
// so there is nothing more to do.
func (s *csvSink) OnError(err error) {
	s.w.Flush()
}
